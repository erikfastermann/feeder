package handler

import (
	"net/http"
)

func (h *Handler) feeds(w http.ResponseWriter, r *http.Request) error {
	items, err := h.DB.AllFeeds()
	if err != nil {
		return err
	}
	contentTypeHTML(w)
	return h.tmplts.ExecuteTemplate(w, "feeds.html", items)
}
