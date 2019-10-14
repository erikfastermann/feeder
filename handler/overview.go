package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/erikfastermann/httpwrap"
)

func (h *Handler) overview(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	pageStr := r.FormValue("page")
	page := uint64(0)
	if pageStr != "" {
		var err error
		page, err = strconv.ParseUint(pageStr, 10, 0)
		if err != nil {
			return httpwrap.Error{
				StatusCode: http.StatusBadRequest,
				Err:        fmt.Errorf("overview: ivalid page %s", strconv.Quote(pageStr)),
			}
		}
	}
	items, err := h.DB.Newest(ctx, uint(page*30), 30)
	if err != nil {
		return err
	}
	return h.tmplts.ExecuteTemplate(w, "overview.html", items)
}
