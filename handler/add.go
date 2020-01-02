package handler

import (
	"net/http"
	"net/url"

	"github.com/erikfastermann/feeder/parser"
)

func (h *Handler) addFeed(w http.ResponseWriter, r *http.Request) error {
	feedURL := r.FormValue("url")

	items, err := parser.Parse(feedURL)
	if err != nil {
		return badRequestf("add: failed parsing feed %s, %v", feedURL, err)
	}

	url, err := url.Parse(feedURL)
	if err != nil {
		return err
	}
	id, err := h.DB.AddFeed(url.Scheme+"://"+url.Host, feedURL)
	if err != nil {
		return err
	}
	if err := h.DB.AddItems(id, items); err != nil {
		return err
	}

	http.Redirect(w, r, routeFeeds, http.StatusTemporaryRedirect)
	return nil
}
