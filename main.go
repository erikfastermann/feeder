package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/erikfastermann/feeder/db/sqlite3"
	"github.com/erikfastermann/feeder/handler"
	"github.com/erikfastermann/httpwrap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 5 {
		return fmt.Errorf("USAGE: %s ADDRESS TEMPLATE_GLOB ADD_PREFIX DB_PATH", os.Args[0])
	}
	addr, tmpltGlob, addPrefix, dbPath := os.Args[1], os.Args[2], os.Args[3], os.Args[4]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	sqlDB, err := sqlite3.Open(ctx, dbPath)
	cancel()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	l := log.New(os.Stderr, "", log.LstdFlags)
	h := &handler.Handler{
		Logger:       l,
		TemplateGlob: tmpltGlob,
		AddPrefix:    addPrefix,
		DB:           sqlDB,
	}
	return http.ListenAndServe(addr, httpwrap.LogCustom(httpwrap.HandleError(h), l))
}
