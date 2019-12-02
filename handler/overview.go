package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/erikfastermann/feeder/db"
	"github.com/erikfastermann/httpwrap"
)

func (h *Handler) overview(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	page := uint(0)
	if pageStr := r.FormValue("page"); pageStr != "" {
		page64, err := strconv.ParseUint(pageStr, 10, 0)
		page = uint(page64)
		const uintMax = ^uint(0)
		if err != nil || page > uintMax/30 {
			return httpwrap.Error{
				StatusCode: http.StatusBadRequest,
				Err:        fmt.Errorf("overview: ivalid page %s", strconv.Quote(pageStr)),
			}
		}
	}

	count, err := h.DB.ItemCount(ctx)
	if err != nil {
		return err
	}
	const itemsPerPage = 30
	offset := page * itemsPerPage
	items, err := h.DB.Newest(ctx, offset, itemsPerPage)
	if err != nil {
		return err
	}

	next := int(page) + 1
	if offset+itemsPerPage > uint(count) {
		next = -1
	}

	return h.tmplts.ExecuteTemplate(w, "overview.html", struct {
		Prev  int
		Next  int
		Items []db.ItemWithHost
	}{
		Prev:  int(page) - 1,
		Next:  next,
		Items: items,
	})
}
