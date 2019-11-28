package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/erikfastermann/feeder/db"
	"github.com/mmcdole/gofeed"
)

func (h *Handler) updateFeeds(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	go h.updateAllFeedItems()
	http.Redirect(w, r, routeOverview, http.StatusTemporaryRedirect)
	return nil
}

func (h *Handler) updateAllFeedItems() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	feeds, err := h.DB.AllFeeds(ctx)
	cancel()
	if err != nil {
		h.Logger.Print(err)
		if feeds == nil {
			return
		}
	}

	for _, feed := range feeds {
		h.updateFeedItems(feed.ID, feed.FeedURL)
	}
}

func (h *Handler) updateFeedItems(feedID int64, feedLink string) {
	added, err := h.doUpdateFeedItems(feedID, feedLink)
	for _, item := range added {
		h.Logger.Printf("feed %s (%d): added %s (%s), ID: %d",
			feedLink,
			feedID,
			strconv.Quote(item.Title),
			item.Added,
			item.ID,
		)
	}
	if err != nil {
		h.Logger.Printf("failed parsing feed %s, %v", feedLink, err)
	}
}

func (h *Handler) doUpdateFeedItems(feedID int64, feedLink string) ([]db.Item, error) {
	feed, err := h.getFeed(feedLink)
	if err != nil {
		return nil, err
	}

	items, err := h.parseItems(feed, feedID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.DB.AddItems(ctx, items)
}

func (h *Handler) getFeed(url string) (*gofeed.Feed, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	feed, err := h.parser.Parse(res.Body)
	res.Body.Close()
	return feed, err
}

func parseFeed(feed *gofeed.Feed, feedLink string) db.Feed {
	return db.Feed{
		URL:     feed.Link,
		FeedURL: feedLink,
		LastUpdated: sql.NullTime{
			Valid: true,
			Time:  time.Now(),
		},
	}
}
func (h *Handler) parseItems(feed *gofeed.Feed, feedID int64) ([]db.Item, error) {
	if len(feed.Items) == 0 {
		return nil, nil
	}
	items := make([]db.Item, 0)
	for _, item := range feed.Items {
		if item.UpdatedParsed == nil && item.PublishedParsed == nil {
			h.Logger.Printf("item %s has an invalid time", item.Title)
			continue
		}
		var t time.Time
		if item.PublishedParsed != nil {
			t = *item.PublishedParsed
		} else {
			t = *item.UpdatedParsed
		}

		items = append(items, db.Item{
			FeedID: feedID,
			Title:  item.Title,
			URL:    item.Link,
			Added:  t,
		})
	}
	return items, nil
}
