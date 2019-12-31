package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/erikfastermann/feeder/db"
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
	if len(os.Args) != 8 {
		return fmt.Errorf("USAGE: %s ADDRESS CERT_FILE KEY_FILE TEMPLATE_GLOB CSV_CTR CSV_FEEDS CSV_ITEMS", os.Args[0])
	}
	addr := os.Args[1]
	crt, key := os.Args[2], os.Args[3]
	tmplt := os.Args[4]
	ctr, feeds, items := os.Args[5], os.Args[6], os.Args[7]

	username := os.Getenv("FEEDER_USERNAME")
	if username == "" {
		return fmt.Errorf("environment variable FEEDER_USERNAME empty or unset")
	}
	password := os.Getenv("FEEDER_PASSWORD")
	if password == "" {
		return fmt.Errorf("environment variable FEEDER_PASSWORD empty or unset")
	}

	csv, err := db.Open(ctr, feeds, items)
	if err != nil {
		return err
	}
	defer csv.Close()

	h := &handler.Handler{
		Logger:       log.New(os.Stderr, "ERROR ", log.LstdFlags),
		Username:     username,
		Password:     password,
		TemplateGlob: tmplt,
		DB:           csv,
	}
	return http.ListenAndServeTLS(addr, crt, key, httpwrap.Log(httpwrap.HandleError(h)))
}
