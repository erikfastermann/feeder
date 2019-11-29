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
	if len(os.Args) != 6 {
		return fmt.Errorf("USAGE: %s ADDRESS CERT_FILE KEY_FILE TEMPLATE_GLOB DB_PATH", os.Args[0])
	}
	addr, certFile, keyFile, tmpltGlob, dbPath := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]
	username := os.Getenv("FEEDER_USERNAME")
	if username == "" {
		return fmt.Errorf("environment variable FEEDER_USERNAME empty or unset")
	}
	password := os.Getenv("FEEDER_PASSWORD")
	if password == "" {
		return fmt.Errorf("environment variable FEEDER_PASSWORD empty or unset")
	}

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
		Username:     username,
		Password:     password,
		TemplateGlob: tmpltGlob,
		DB:           sqlDB,
	}
	return http.ListenAndServeTLS(addr, certFile, keyFile, httpwrap.LogCustom(httpwrap.HandleError(h), l))
}
