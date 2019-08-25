package handler

import (
	"context"
	"encoding/json"
	"net/http"
)

func (h *Handler) overview(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	items, err := h.DB.Newest(ctx, 30)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	if err := enc.Encode(items); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
