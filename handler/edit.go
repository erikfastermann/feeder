package handler

import (
	"database/sql"
	"net/http"
	"strconv"
)

func (h *Handler) edit(w http.ResponseWriter, r *http.Request) error {
	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return badRequestf("%s is an invalid id, %v", strconv.Quote(idStr), err)
	}
	host := r.FormValue("host")

	if err := h.DB.EditFeedHost(id, host); err != nil {
		if err == sql.ErrNoRows {
			return badRequestf("id %d not found in db, %v", id, err)
		}
		return err
	}

	http.Redirect(w, r, routeFeeds, http.StatusTemporaryRedirect)
	return nil
}
