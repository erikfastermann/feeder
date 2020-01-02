package handler

import (
	"database/sql"
	"net/http"
	"strconv"
)

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) error {
	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return badRequestf("%s is an invalid id, %v", strconv.Quote(idStr), err)
	}

	if err := h.DB.RemoveFeed(id); err != nil {
		if err == sql.ErrNoRows {
			return badRequestf("id %d not found in db, %v", id, err)
		}
		return err
	}

	http.Redirect(w, r, routeFeeds, http.StatusTemporaryRedirect)
	return nil
}
