package handler

import (
	"context"
	"net/http"
)

func (h *Handler) feeds(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	items, err := h.DB.AllFeeds(ctx)
	if err != nil {
		return err
	}
	return h.tmplts.ExecuteTemplate(w, "feeds.html", items)
}
