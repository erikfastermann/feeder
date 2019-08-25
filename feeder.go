package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/erikfastermann/feeder/db"
	"github.com/erikfastermann/feeder/db/sqlite3"
	"github.com/erikfastermann/kvparser"
	"github.com/mmcdole/gofeed"
)

var templateOverview = []byte(``)

type Server struct {
	Logger *log.Logger

	once     sync.Once
	FeedPath string
	mu       sync.RWMutex
	feeds    []kvparser.KeyValue
	parser   *gofeed.Parser

	DB db.DB
}

func (srv *Server) ReadFeedPath() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	f, err := os.Open(srv.FeedPath)
	if err != nil {
		return err
	}
	defer f.Close()

	feeds, err := kvparser.Parse(f)
	if err != nil {
		return err
	}
	srv.feeds = feeds
	return nil
}

func (srv *Server) updateFeed(name, url string) ([]db.Item, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(req.Context(), 15*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	feed, err := srv.parser.Parse(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	feedAuthor := checkAuthor(feed.Author, name)

	if len(feed.Items) == 0 {
		return nil, nil
	}
	items := make([]db.Item, 0)
	for _, item := range feed.Items {
		if item.UpdatedParsed == nil && item.PublishedParsed == nil {
			srv.Logger.Printf("item %s has an invalid time")
			continue
		}
		var t time.Time
		if item.PublishedParsed != nil {
			t = *item.PublishedParsed
		} else {
			t = *item.UpdatedParsed
		}

		author := checkAuthor(item.Author, feedAuthor)

		desc := item.Description
		if desc == "" {
			desc = item.Content
		}
		if len(desc) > 300 {
			desc = desc[:300] + "..."
		}

		items = append(items, db.Item{
			FeedTitle:   name,
			Author:      author,
			Title:       item.Title,
			Description: desc,
			Link:        item.Link,
			Updated:     t,
		})
	}

	return srv.DB.AddItems(ctx, items)
}

func checkAuthor(author *gofeed.Person, name string) string {
	if author != nil && author.Name != "" {
		return author.Name
	}
	return name
}

func (srv *Server) updateAllFeeds() {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	for _, feed := range srv.feeds {
		added, err := srv.updateFeed(feed.Key, feed.Value)
		for _, item := range added {
			srv.Logger.Printf("feed %s: added %s (%s)",
				feed.Key,
				strconv.Quote(item.Title),
				item.Updated,
			)
		}
		if err != nil {
			srv.Logger.Printf("failed parsing feed %s, %v", feed.Value, err)
		}
	}
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	srv.once.Do(func() {
		if srv.Logger == nil {
			srv.Logger = log.New(ioutil.Discard, "", 0)
		}

		srv.parser = gofeed.NewParser()
		go func() {
			srv.updateAllFeeds()
			for range time.Tick(5 * time.Minute) {
				srv.updateAllFeeds()
			}
		}()
	})

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	switch path.Clean(r.URL.Path) {
	case "/":
		return srv.overview(ctx, w, r)
	case "/update":
		return srv.updateFeeds(ctx, w, r)
	default:
		return http.StatusNotFound, fmt.Errorf("router: invalid URL %s", r.URL.Path)
	}
}

func (srv *Server) overview(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	items, err := srv.DB.Newest(ctx, 30)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	if err := enc.Encode(items); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (srv *Server) updateFeeds(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := srv.ReadFeedPath(); err != nil {
		return http.StatusBadRequest, err
	}
	go srv.updateAllFeeds()
	return http.StatusOK, nil
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) (status int, internalErr error)

func LogWrapper(fn HandlerFunc, l *log.Logger) http.HandlerFunc {
	if l == nil {
		l = log.New(ioutil.Discard, "", 0)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		status, err := fn(w, r)
		l.Printf("%s|%s %s|%d - %s|%v",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			status,
			http.StatusText(status),
			err,
		)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "USAGE: %s FEED_PATH DB_PATH", os.Args[0])
		os.Exit(1)
	}

	sqlDB, err := sqlite3.Open(context.TODO(), os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	l := log.New(os.Stderr, "", log.LstdFlags)
	srv := &Server{
		Logger:   l,
		FeedPath: os.Args[1],
		DB:       sqlDB,
	}
	if err := srv.ReadFeedPath(); err != nil {
		log.Fatal(err)
	}

	l.Fatal(http.ListenAndServe("localhost:8080", LogWrapper(srv.ServeHTTP, l)))
}
