package handler

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/erikfastermann/feeder/db"
	"github.com/erikfastermann/httpwrap"
	"github.com/mmcdole/gofeed"
)

const (
	routeOverview = "/"
	routeFeeds    = "/feeds"
	routeAdd      = "/add"
)

type Handler struct {
	once         sync.Once
	Logger       *log.Logger
	parser       *gofeed.Parser
	TemplateGlob string
	AddSuffix    string
	tmplts       *template.Template
	DB           db.DB
}

func (h *Handler) ServeHTTPWithErr(w http.ResponseWriter, r *http.Request) error {
	h.once.Do(func() {
		if h.Logger == nil {
			h.Logger = log.New(ioutil.Discard, "", 0)
		}

		funcMap := template.FuncMap{
			"FormatHost": formatHost,
		}
		h.tmplts = template.Must(template.New("").Funcs(funcMap).ParseGlob(h.TemplateGlob))
		if h.AddSuffix != "" {
			h.AddSuffix = "-" + h.AddSuffix
		}

		h.parser = gofeed.NewParser()
		go func() {
			h.updateAllFeedItems()
			for range time.Tick(5 * time.Minute) {
				h.updateAllFeedItems()
			}
		}()
	})

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	split := strings.Split(path.Clean(r.URL.Path), "/")
	route := "/"
	if len(split) > 1 {
		route = split[1]
	}

	switch route {
	case routeOverview:
		return h.overview(ctx, w, r)
	case routeFeeds:
		return h.feeds(ctx, w, r)
	case routeAdd + h.AddSuffix:
		return h.addFeed(ctx, w, r)
	default:
		return httpwrap.Error{
			StatusCode: http.StatusNotFound,
			Err:        fmt.Errorf("router: invalid URL %s", r.URL.Path),
		}
	}
}

func formatHost(uri string) string {
	parsed, err := url.ParseRequestURI(uri)
	if err != nil {
		return ""
	}
	return parsed.Host
}
