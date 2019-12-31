package db

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Feed struct {
	ID          int
	Host        string
	FeedURL     string
	LastChecked sql.NullTime
	LastUpdated sql.NullTime
}

const (
	fID          = 0
	fHost        = 1
	fFeedURL     = 2
	fLastChecked = 3
	fLastUpdated = 4
	fLen         = 5
)

func feedsToRecs(feeds ...Feed) [][]string {
	date := func(t sql.NullTime) string {
		if !t.Valid {
			return ""
		}
		return t.Time.Format(timeFormat)
	}

	recs := make([][]string, 0)
	for _, f := range feeds {
		r := make([]string, fLen)
		r[fID] = strconv.Itoa(f.ID)
		r[fHost] = f.Host
		r[fFeedURL] = f.FeedURL
		r[fLastChecked] = date(f.LastChecked)
		r[fLastUpdated] = date(f.LastUpdated)
		recs = append(recs, r)
	}
	return recs
}

func sortFeeds(feeds []Feed) {
	sort.Slice(feeds, func(i, j int) bool {
		// TODO: check valid
		return feeds[i].LastUpdated.Time.After(feeds[j].LastUpdated.Time)
	})
}

type Item struct {
	FeedID int
	Title  string
	URL    string
	Added  time.Time
}

const (
	iFeedID = 0
	iTitle  = 1
	iURL    = 2
	iAdded  = 3
	iLen    = 4
)

func itemsToRecs(items ...Item) [][]string {
	recs := make([][]string, 0)
	for _, item := range items {
		r := make([]string, iLen)
		r[iFeedID] = strconv.Itoa(item.FeedID)
		r[iTitle] = item.Title
		r[iURL] = item.URL
		r[iAdded] = item.Added.Format(timeFormat)
		recs = append(recs, r)
	}
	return recs
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Added.After(items[j].Added)
	})
}

type ItemWithHost struct {
	Item
	Host string
}

type DB struct {
	mu sync.RWMutex

	ctr      *os.File
	csvFeeds *os.File
	csvItems *os.File

	feeds []Feed
	items []Item
}

const timeFormat = time.RFC3339

func Open(ctrPath, feedsPath, itemsPath string) (*DB, error) {
	open := func(path string) (*os.File, error) {
		return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_SYNC, 0644)
	}

	ctr, err := open(ctrPath)
	if err != nil {
		return nil, err
	}
	f, err := open(feedsPath)
	if err != nil {
		ctr.Close()
		return nil, err
	}
	f2, err := open(itemsPath)
	if err != nil {
		ctr.Close()
		f.Close()
		return nil, err
	}
	db := &DB{ctr: ctr, csvFeeds: f, csvItems: f2}

	err = func() error {
		date := func(s string) (sql.NullTime, error) {
			if s == "" {
				return sql.NullTime{}, nil
			}
			t, err := time.Parse(timeFormat, s)
			if err != nil {
				return sql.NullTime{}, err
			}
			return sql.NullTime{
				Valid: true,
				Time:  t,
			}, nil
		}

		fi, err := ctr.Stat()
		if err != nil {
			return err
		}
		if fi.Size() == 0 {
			_, err := ctr.Write([]byte("1"))
			return err
		}

		recs, err := csv.NewReader(f).ReadAll()
		if err != nil {
			return err
		}
		for _, r := range recs {
			if len(r) != fLen {
				return errors.New("feeds: unexpected row length")
			}

			feed := Feed{
				Host:    r[fHost],
				FeedURL: r[fFeedURL],
			}

			feed.ID, err = strconv.Atoi(r[fID])
			if err != nil {
				return err
			}

			feed.LastChecked, err = date(r[fLastChecked])
			if err != nil {
				return err
			}
			feed.LastUpdated, err = date(r[fLastUpdated])
			if err != nil {
				return err
			}

			db.feeds = append(db.feeds, feed)
		}

		recs, err = csv.NewReader(f2).ReadAll()
		if err != nil {
			return err
		}
		for _, r := range recs {
			if len(r) != iLen {
				return errors.New("items: unexpected row length")
			}

			item := Item{
				Title: r[iTitle],
				URL:   r[iURL],
			}

			item.FeedID, err = strconv.Atoi(r[iFeedID])
			if err != nil {
				return err
			}

			item.Added, err = time.Parse(timeFormat, r[iAdded])
			if err != nil {
				return err
			}

			db.items = append(db.items, item)
		}

		return nil
	}()
	if err != nil {
		ctr.Close()
		f.Close()
		f2.Close()
		return nil, err
	}

	sortFeeds(db.feeds)
	sortItems(db.items)
	return db, nil
}

func (db *DB) Close() error {
	var outer error
	for _, c := range []io.Closer{db.ctr, db.csvFeeds, db.csvItems} {
		if err := c.Close(); err != nil {
			outer = err
		}
	}
	return outer
}

var ErrFound = errors.New("feed already exists in the database")

func (db *DB) AddFeed(_ context.Context, host, feedURL string) (int, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, f := range db.feeds {
		if f.FeedURL == feedURL {
			return -1, ErrFound
		}
	}

	id, err := db.bumpCtr()
	if err != nil {
		return -1, err
	}
	feed := Feed{
		ID:      id,
		Host:    host,
		FeedURL: feedURL,
	}

	if err := insert(db.csvFeeds, feedsToRecs(feed)); err != nil {
		return -1, err
	}
	db.feeds = append(db.feeds, feed)
	sortFeeds(db.feeds)
	return id, nil
}

var timeNow = time.Now

func (db *DB) AddItems(_ context.Context, feedID int, items []Item) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	now := sql.NullTime{
		Valid: true,
		Time:  timeNow(),
	}
	idx := -1
	for i, f := range db.feeds {
		if f.ID == feedID {
			idx = i
			db.feeds[idx].LastChecked = now
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("unknown feed id %d", feedID)
	}

	known := make(map[int]struct{})
	for _, dbItem := range db.items {
		for i, item := range items {
			if dbItem.URL == item.URL {
				known[i] = struct{}{}
			}
		}
	}

	add := make([]Item, 0)
	for i, item := range items {
		if _, ok := known[i]; !ok {
			item.FeedID = feedID
			add = append(add, item)
		}
	}

	if len(add) > 0 {
		if err := insert(db.csvItems, itemsToRecs(add...)); err != nil {
			return err
		}
		db.items = append(db.items, add...)
		sortItems(db.items)

		db.feeds[idx].LastUpdated = now
		sortFeeds(db.feeds)
	}

	return rewrite(db.csvFeeds, feedsToRecs(db.feeds...))
}

func (db *DB) AllFeeds(_ context.Context) ([]Feed, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	feeds := make([]Feed, len(db.feeds))
	copy(feeds, db.feeds)
	return feeds, nil
}

func (db *DB) EditFeedHost(_ context.Context, id int, newHost string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, f := range db.feeds {
		if f.ID == id {
			db.feeds[i].Host = newHost
			return rewrite(db.csvFeeds, feedsToRecs(db.feeds...))
		}
	}
	return sql.ErrNoRows
}

func (db *DB) ItemCount(_ context.Context) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.items), nil
}

func (db *DB) Newest(_ context.Context, offset, limit uint) ([]ItemWithHost, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	count := uint(len(db.items))
	if offset >= count {
		return nil, sql.ErrNoRows
	}

	limit += offset
	if limit >= count {
		limit = count
	}

	m := make(map[int]Feed)
	for _, f := range db.feeds {
		m[f.ID] = f
	}

	iwh := make([]ItemWithHost, 0)
	for _, item := range db.items[offset:limit] {
		f, ok := m[item.FeedID]
		if !ok {
			return nil, fmt.Errorf("unknown feed id %d", item.FeedID)
		}
		iwh = append(iwh, ItemWithHost{Item: item, Host: f.Host})
	}
	return iwh, nil
}

func (db *DB) RemoveFeed(_ context.Context, id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	found := false
	for i, f := range db.feeds {
		if f.ID == id {
			found = true
			db.feeds[i] = db.feeds[len(db.feeds)-1]
			db.feeds = db.feeds[:len(db.feeds)-1]
			sortFeeds(db.feeds)
		}
	}
	if !found {
		return fmt.Errorf("unknown feed id %d", id)
	}
	if err := rewrite(db.csvFeeds, feedsToRecs(db.feeds...)); err != nil {
		return err
	}

	keep := make([]Item, 0)
	for _, item := range db.items {
		if item.FeedID != id {
			keep = append(keep, item)
		}
	}
	db.items = keep
	sortItems(db.items)

	return rewrite(db.csvItems, itemsToRecs(db.items...))
}

func (db *DB) bumpCtr() (int, error) {
	var ctr int
	err := func() error {
		if _, err := db.ctr.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err := fmt.Fscanf(db.ctr, "%d", &ctr); err != nil {
			return err
		}
		if err := db.ctr.Truncate(0); err != nil {
			return err
		}
		if _, err := db.ctr.Seek(0, io.SeekStart); err != nil {
			return err
		}
		_, err := fmt.Fprintf(db.ctr, "%d", ctr+1)
		return err
	}()
	if err != nil {
		return -1, err
	}
	return ctr, nil
}

func insert(f *os.File, recs [][]string) error {
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return csv.NewWriter(f).WriteAll(recs)
}

func rewrite(f *os.File, recs [][]string) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return csv.NewWriter(f).WriteAll(recs)
}
