package db

import (
	"context"
	"time"
)

type DB interface {
	Close() error

	AddItems(ctx context.Context, items []Item) (addedItems []Item, err error)
	Newest(ctx context.Context, n uint) ([]Item, error)
}

type Item struct {
	FeedTitle   string    `json:"feed_title"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	ImageURL    string    `json:"image_url"` // currently unused by the handler
	Updated     time.Time `json:"updated"`
}
