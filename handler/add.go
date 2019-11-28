package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/erikfastermann/httpwrap"
)

func (h *Handler) addFeed(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	feedURL := r.FormValue("url")
	items, err := h.getItems(feedURL)
	if err != nil {
		return httpwrap.Error{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("add: failed parsing feed %s, %v", feedURL, err),
		}
	}

	id, err := h.DB.AddFeed(ctx, url, feedURL)
	if err != nil {
		return err
	}
	if err := h.DB.AddItems(ctx, id, items); err != nil {
		return err
	}

	http.Redirect(w, r, routeOverview, http.StatusTemporaryRedirect)
	return nil
}
