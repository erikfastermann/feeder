package handler

import (
	"context"
	"crypto/subtle"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/erikfastermann/feeder/db"
	"github.com/erikfastermann/httpwrap"
)

const (
	routeOverview = "/"
	routeFeeds    = "/feeds"
	routeAdd      = "/add"
	routeRemove   = "/remove"
	routeEdit     = "/edit"
)

type Handler struct {
	once               sync.Once
	Logger             *log.Logger
	Username, Password string
	TemplateGlob       string
	tmplts             *template.Template
	DB                 *db.DB
}

func (h *Handler) ServeHTTPWithErr(w http.ResponseWriter, r *http.Request) error {
	h.once.Do(func() {
		if h.Logger == nil {
			h.Logger = log.New(ioutil.Discard, "", 0)
		}

		h.tmplts = template.Must(template.ParseGlob(h.TemplateGlob))
		go func() {
			h.update()
			for range time.Tick(5 * time.Minute) {
				h.update()
			}
		}()
	})

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, pass, ok := r.BasicAuth()
	userOk := subtle.ConstantTimeCompare([]byte(user), []byte(h.Username))
	passOk := subtle.ConstantTimeCompare([]byte(pass), []byte(h.Password))
	if !ok || userOk != 1 || passOk != 1 {
		w.Header().Set("WWW-Authenticate", "Basic")
		return httpwrap.Error{
			StatusCode: http.StatusUnauthorized,
			Err:        fmt.Errorf("router: invalid login credentials"),
		}
	}

	split := strings.Split(path.Clean(r.URL.Path), "/")
	route := "/"
	if len(split) > 1 {
		route = "/" + split[1]
	}

	switch route {
	case routeOverview:
		return h.overview(ctx, w, r)
	case routeFeeds:
		return h.feeds(ctx, w, r)
	case routeAdd:
		return h.addFeed(ctx, w, r)
	case routeEdit:
		return h.edit(ctx, w, r)
	case routeRemove:
		return h.remove(ctx, w, r)
	default:
		return httpwrap.Error{
			StatusCode: http.StatusNotFound,
			Err:        fmt.Errorf("router: invalid URL %s", r.URL.Path),
		}
	}
}

func badRequestf(format string, a ...interface{}) error {
	return httpwrap.Error{
		StatusCode: http.StatusBadRequest,
		Err:        fmt.Errorf(format, a...),
	}
}
