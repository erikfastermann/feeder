package handler

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/erikfastermann/feeder/db"
	"github.com/erikfastermann/kvparser"
	"github.com/mmcdole/gofeed"
)

type Handler struct {
	Logger *log.Logger

	once     sync.Once
	FeedPath string
	mu       sync.RWMutex
	feeds    []kvparser.KeyValue
	parser   *gofeed.Parser

	TemplateGlob string
	tmplts       *template.Template

	DB db.DB
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	h.once.Do(func() {
		if h.Logger == nil {
			h.Logger = log.New(ioutil.Discard, "", 0)
		}
		h.tmplts = template.Must(template.ParseGlob(h.TemplateGlob))

		h.parser = gofeed.NewParser()
		go func() {
			h.updateAllFeeds()
			for range time.Tick(5 * time.Minute) {
				h.updateAllFeeds()
			}
		}()
	})

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	switch path.Clean(r.URL.Path) {
	case "/":
		return h.overview(ctx, w, r)
	case "/update":
		return h.updateFeeds(ctx, w, r)
	default:
		return http.StatusNotFound, fmt.Errorf("router: invalid URL %s", r.URL.Path)
	}
}
