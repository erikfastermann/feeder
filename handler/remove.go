package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/erikfastermann/httpwrap"
)

func (h *Handler) remove(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return httpwrap.Error{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("%s is an invalid id, %v", strconv.Quote(idStr), err),
		}
	}

	if err := h.DB.RemoveFeed(ctx, id); err != nil {
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
