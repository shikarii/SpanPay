package main

import (
	"log"
	"net/http"
	"os"

	"github.com/shikarii/spanpay/server/internal/httpapi"
)

func main() {
	addr := os.Getenv("SPANPAY_HTTP_ADDR")
	if addr == "" {
		addr = ":6940"
	}

	server := &http.Server{
		Addr:    addr,
		Handler: httpapi.NewRouter(),
	}

	log.Printf("SpanPay listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
