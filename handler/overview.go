package handler

import (
	"context"
	"net/http"
)

func (h *Handler) overview(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	items, err := h.DB.Newest(ctx, 30)
	if err != nil {
		return err
	}
	return h.tmplts.ExecuteTemplate(w, "overview.html", items)
}
