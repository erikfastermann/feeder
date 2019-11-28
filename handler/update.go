package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/erikfastermann/feeder/db"
)

func (h *Handler) update() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	feeds, err := h.DB.AllFeeds(ctx)
	cancel()
	if err != nil {
		h.Logger.Print(err)
	}

	for _, feed := range feeds {
		items, err := h.getItems(feed.FeedURL)
		if err != nil {
			h.Logger.Printf("failed parsing feed %s, %v", feed.FeedURL, err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := h.DB.AddItems(ctx, feed.ID, items)
		cancel()
		if err != nil {
			h.Logger.Printf("failed updating db %v", err)
		}
	}
}

func (h *Handler) getItems(feedURL string) ([]db.Item, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Get(feedURL)
	if err != nil {
		return nil, err
	}
	feed, err := h.parser.Parse(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	items := make([]db.Item, 0)
	for _, item := range feed.Items {
		if item.UpdatedParsed == nil && item.PublishedParsed == nil {
			h.Logger.Printf("item %s has an invalid time", item.Title) // TODO: How to handle this error?
			continue
		}
		var t time.Time
		if item.PublishedParsed != nil {
			t = *item.PublishedParsed
		} else {
			t = *item.UpdatedParsed
		}

		items = append(items, db.Item{
			ID:     -1,
			FeedID: -1,
			Title:  item.Title,
			URL:    item.Link,
			Added:  t,
		})
	}
	return items, nil
}
