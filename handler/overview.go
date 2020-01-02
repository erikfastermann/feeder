package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/erikfastermann/feeder/db"
)

func (h *Handler) overview(w http.ResponseWriter, r *http.Request) error {
	const itemsPerPage = 30
	page := uint(0)
	if pageStr := r.FormValue("page"); pageStr != "" {
		page64, err := strconv.ParseUint(pageStr, 10, 0)
		page = uint(page64)
		const uintMax = ^uint(0)
		if err != nil || page > uintMax/itemsPerPage {
			return badRequestf("overview: ivalid page %s", strconv.Quote(pageStr))
		}
	}

	type data struct {
		Prev  int
		Next  int
		Items []db.ItemWithHost
	}

	count, err := h.DB.ItemCount()
	if err != nil {
		return err
	}
	if count == 0 && page == 0 {
		contentTypeHTML(w)
		return h.tmplts.ExecuteTemplate(w, "overview.html", data{Prev: -1, Next: -1})
	}

	offset := page * itemsPerPage
	items, err := h.DB.Newest(offset, itemsPerPage)
	if err != nil {
		if err == sql.ErrNoRows {
			return badRequestf("overview: invalid page %d", page)
		}
		return err
	}

	next := int(page) + 1
	if offset+itemsPerPage >= uint(count) {
		next = -1
	}

	contentTypeHTML(w)
	return h.tmplts.ExecuteTemplate(w, "overview.html", data{
		Prev:  int(page) - 1,
		Next:  next,
		Items: items,
	})
}
