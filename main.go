package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/erikfastermann/feeder/db/sqlite3"
	"github.com/erikfastermann/feeder/handler"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintf(os.Stderr, "USAGE: %s ADDRESS FEED_PATH TEMPLATE_GLOB DB_PATH\n", os.Args[0])
		os.Exit(1)
	}
	addr, feedPath, tmpltGlob, dbPath := os.Args[1], os.Args[2], os.Args[3], os.Args[4]

	sqlDB, err := sqlite3.Open(context.TODO(), dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	l := log.New(os.Stderr, "", log.LstdFlags)
	h := &handler.Handler{
		Logger:       l,
		FeedPath:     feedPath,
		TemplateGlob: tmpltGlob,
		DB:           sqlDB,
	}
	if err := h.ReadFeedPath(); err != nil {
		log.Fatal(err)
	}

	l.Fatal(http.ListenAndServe(addr, LogWrapper(ErrorWrapper(h.ServeHTTP), l)))
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) (status int, internalErr error)

func ErrorWrapper(fn HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) (int, error) {
		customWriter, ok := w.(*ResponseWriter)
		if !ok {
			customWriter = &ResponseWriter{Orig: w}
		}
		status, err := fn(customWriter, r)
		if !customWriter.WroteHeader {
			customWriter.WriteHeader(status)
		}

		if status >= 400 {
			fmt.Fprintf(customWriter, "%d - %s", status, http.StatusText(status))
		}
		return status, err
	}
}

type ResponseWriter struct {
	Orig        http.ResponseWriter
	WroteHeader bool
}

func (w *ResponseWriter) Header() http.Header {
	return w.Orig.Header()
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.WroteHeader = true
	w.Orig.WriteHeader(statusCode)
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	w.WroteHeader = true
	return w.Orig.Write(p)
}

func LogWrapper(fn HandlerFunc, l *log.Logger) http.HandlerFunc {
	if l == nil {
		l = log.New(ioutil.Discard, "", 0)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		status, err := fn(w, r)
		l.Printf("%s|%s %s|%d - %s|%v",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			status,
			http.StatusText(status),
			err,
		)
	}
}
