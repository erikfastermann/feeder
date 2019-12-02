package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/erikfastermann/httpwrap"
)

func (h *Handler) edit(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return httpwrap.Error{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("%s is an invalid id, %v", strconv.Quote(idStr), err),
		}
	}
	host := r.FormValue("host")

	if err := h.DB.EditFeedHost(ctx, id, host); err != nil {
		if err == sql.ErrNoRows {
			return httpwrap.Error{
				StatusCode: http.StatusBadRequest,
				Err:        fmt.Errorf("id %d not found in db, %v", id, err),
			}
		}
		return err
	}

	http.Redirect(w, r, routeFeeds, http.StatusTemporaryRedirect)
	return nil
}
