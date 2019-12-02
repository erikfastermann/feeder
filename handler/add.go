package handler

import (
	"context"
	"net/http"
	"net/url"
)

func (h *Handler) addFeed(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	feedURL := r.FormValue("url")

	items, err := h.getItems(feedURL)
	if err != nil {
		return badRequestf("add: failed parsing feed %s, %v", feedURL, err)
	}

	url, err := url.Parse(feedURL)
	if err != nil {
		return err
	}
	id, err := h.DB.AddFeed(ctx, url.Scheme+"://"+url.Host, feedURL)
	if err != nil {
		return err
	}
	if err := h.DB.AddItems(ctx, id, items); err != nil {
		return err
	}

	http.Redirect(w, r, routeFeeds, http.StatusTemporaryRedirect)
	return nil
}
