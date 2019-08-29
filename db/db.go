package db

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"
)

type DB interface {
	Close() error

	AddFeed(ctx context.Context, feed Feed) (feedID int64, err error)
	AddItems(ctx context.Context, items []Item) (addedItems []Item, err error)

	AllFeeds(ctx context.Context) ([]Feed, error)
	Newest(ctx context.Context, n uint) ([]Item, error)
}

type Feed struct {
	ID          int64    `json:"id"`
	Author      string   `json:"author"`
	Title       string   `json:"title"`
	Language    string   `json:"language"`
	Description string   `json:"description"`
	Link        string   `json:"link"`
	FeedLink    string   `json:"feed_link"`
	ImageURL    string   `json:"image_url"` // currently unused by the handler
	LastUpdated NullTime `json:"last_updated"`
}

type Item struct {
	ID          int64     `json:"id"`
	FeedID      int64     `json:"feed_id"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Link        string    `json:"link"`
	ImageURL    string    `json:"image_url"` // currently unused by the handler
	Added       time.Time `json:"added"`
}

type NullTime struct {
	Time  time.Time
	Valid bool
}

func (nt *NullTime) Scan(value interface{}) error {
	if value == nil {
		nt.Time = time.Time{}
		nt.Valid = false
		return nil
	}
	t, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("%T is not nil or time.Time", value)
	}
	nt.Time = t
	nt.Valid = true
	return nil
}

func (nt NullTime) Value() (driver.Value, error) {
	if nt.Valid {
		return nt.Time, nil
	}
	return nil, nil
}
