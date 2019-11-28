package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/erikfastermann/httpwrap"
)

func (h *Handler) addFeed(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	feedURL := r.FormValue("url")
	feed, err := h.getFeed(feedURL)
	if err != nil {
		return httpwrap.Error{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("add: failed parsing feed %s, %v", feedURL, err),
		}
	}
	id, err := h.DB.AddFeed(ctx, parseFeed(feed, feedURL))
	if err != nil {
		return fmt.Errorf("add: failed storing feed %s, %v", feedURL, err)
	}
	go h.updateFeedItems(id, feedURL)

	http.Redirect(w, r, routeOverview, http.StatusTemporaryRedirect)
	return nil
}
