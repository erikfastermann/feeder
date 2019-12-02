package db

import (
	"context"
	"database/sql"
	"time"
)

type DB interface {
	AllFeeds(ctx context.Context) ([]Feed, error)
	AddFeed(ctx context.Context, host, feedURL string) (feedID int, err error)
	EditFeedHost(ctx context.Context, id int, newHost string) error
	RemoveFeed(ctx context.Context, id int) error

	Newest(ctx context.Context, offset, limit uint) ([]ItemWithHost, error)
	ItemCount(ctx context.Context) (count int, err error)
	AddItems(ctx context.Context, feedID int, items []Item) error

	Close() error
}

type Feed struct {
	ID          int
	Host        string
	FeedURL     string
	LastChecked sql.NullTime
	LastUpdated sql.NullTime
}

type Item struct {
	ID     int
	FeedID int
	Title  string
	URL    string
	Added  time.Time
}

type ItemWithHost struct {
	Item
	Host string
}
