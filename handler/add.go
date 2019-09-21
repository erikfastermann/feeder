package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/erikfastermann/httpwrap"
)

func (h *Handler) addFeed(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	_, remainder := splitURL(r.URL.Path)
	feedLink := buildFeedURL(remainder[1:], r.URL.RawQuery, r.URL.Fragment)
	feed, err := h.getFeed(feedLink)
	if err != nil {
		return httpwrap.Error{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("add: failed parsing feed %s, %v", feedLink, err),
		}
	}
	id, err := h.DB.AddFeed(ctx, parseFeed(feed, feedLink))
	if err != nil {
		return fmt.Errorf("add: failed storing feed %s, %v", feedLink, err)
	}
	go h.updateFeedItems(id, feedLink)

	http.Redirect(w, r, routeOverview, http.StatusTemporaryRedirect)
	return nil
}

func buildFeedURL(path, rawQuery, fragment string) string {
	if rawQuery != "" {
		path += "?" + rawQuery
	}
	if fragment != "" {
		path += "#" + fragment
	}
	return path
}
