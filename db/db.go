package db

import (
	"context"
	"database/sql"
	"time"
)

type DB interface {
	Close() error

	AddFeed(ctx context.Context, host, feedURL string) (feedID int64, err error)
	AddItems(ctx context.Context, feedID int64, items []Item) (err error)

	AllFeeds(ctx context.Context) ([]Feed, error)
	Newest(ctx context.Context, offset, limit uint) ([]ItemWithHost, error)
}

type Feed struct {
	ID          int64
	Host        string
	FeedURL     string
	LastChecked sql.NullTime
	LastUpdated sql.NullTime
}

type Item struct {
	ID     int64
	FeedID int64
	Title  string
	URL    string
	Added  time.Time
}

type ItemWithHost struct {
	Item
	Host string
}
