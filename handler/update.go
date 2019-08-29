package handler

import (
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/erikfastermann/feeder/db"
	"github.com/erikfastermann/kvparser"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

func (h *Handler) updateFeeds(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := h.ReadFeedPath(); err != nil {
		return http.StatusBadRequest, err
	}
	go h.updateAllFeeds()
	http.Redirect(w, r, routeOverview, http.StatusTemporaryRedirect)
	return http.StatusTemporaryRedirect, nil
}

func (h *Handler) ReadFeedPath() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	f, err := os.Open(h.FeedPath)
	if err != nil {
		return err
	}
	defer f.Close()

	feeds, err := kvparser.Parse(f)
	if err != nil {
		return err
	}
	h.feeds = feeds
	return nil
}

func (h *Handler) updateAllFeeds() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, feed := range h.feeds {
		added, err := h.updateFeed(feed.Key, feed.Value)
		for _, item := range added {
			h.Logger.Printf("feed %s: added %s (%s)",
				feed.Key,
				strconv.Quote(item.Title),
				item.Added,
			)
		}
		if err != nil {
			h.Logger.Printf("failed parsing feed %s, %v", feed.Value, err)
		}
	}
}

func (h *Handler) updateFeed(name, url string) ([]db.Item, error) {
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

	feed, err := h.parser.Parse(res.Body)
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
			h.Logger.Printf("item %s has an invalid time")
			continue
		}
		var t time.Time
		if item.PublishedParsed != nil {
			t = *item.PublishedParsed
		} else {
			t = *item.UpdatedParsed
		}

		author := checkAuthor(item.Author, feedAuthor)

		items = append(items, db.Item{
			FeedTitle:   name,
			Author:      author,
			Title:       item.Title,
			Description: item.Description,
			Content:     item.Content,
			Link:        item.Link,
			Added:       t,
		})
	}

	return h.DB.AddItems(ctx, items)
}

func checkAuthor(author *gofeed.Person, name string) string {
	if author != nil && author.Name != "" {
		return author.Name
	}
	return name
}

func removeHTMLTags(content string) string {
	var sanitized strings.Builder
	r := strings.NewReader(content)
	z := html.NewTokenizer(r)
	for {
		switch t := z.Next(); t {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return sanitized.String()
			}
			continue
		case html.TextToken:
			sanitized.Write(z.Text())
		}
	}
}
