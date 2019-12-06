package handler

import (
	"context"
	"time"

	"github.com/erikfastermann/feeder/parser"
)

func (h *Handler) update() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	feeds, err := h.DB.AllFeeds(ctx)
	cancel()
	if err != nil {
		h.Logger.Print(err)
	}

	for _, feed := range feeds {
		items, err := parser.Items(feed.FeedURL)
		if err != nil {
			h.Logger.Printf("failed parsing feed %s, %v", feed.FeedURL, err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = h.DB.AddItems(ctx, feed.ID, items)
		cancel()
		if err != nil {
			h.Logger.Printf("failed updating db %v", err)
		}
	}
}
