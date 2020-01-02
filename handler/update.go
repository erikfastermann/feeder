package handler

import (
	"github.com/erikfastermann/feeder/parser"
)

func (h *Handler) update() {
	feeds, err := h.DB.AllFeeds()
	if err != nil {
		h.Logger.Print(err)
	}

	for _, feed := range feeds {
		items, err := parser.Parse(feed.FeedURL)
		if err != nil {
			h.Logger.Printf("failed parsing feed %s, %v", feed.FeedURL, err)
			continue
		}

		err = h.DB.AddItems(feed.ID, items)
		if err != nil {
			h.Logger.Printf("failed updating db %v", err)
		}
	}
}
