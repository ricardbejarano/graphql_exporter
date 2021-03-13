package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/usewormhol/env"
)

var (
	EXPORTER_LISTEN_ADDR   = env.String("EXPORTER_LISTEN_ADDR", "127.0.0.1:9199", env.Optional)
	EXPORTER_TLS_CERT_FILE = env.String("EXPORTER_TLS_CERT_FILE", "", env.Optional)
	EXPORTER_TLS_KEY_FILE  = env.String("EXPORTER_TLS_KEY_FILE", "", env.Optional)
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		// Returns some basic info about the exporter.
		w.Write([]byte("GraphQL exporter for Prometheus.\n"))
		w.Write([]byte("Exporter metrics available at /metrics.\n"))
		w.Write([]byte("Querying available at /query.\n\n"))
		w.Write([]byte(fmt.Sprintf("Copyright (c) %s Ricard Bejarano\n", time.Now().Format("2006"))))
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Returns metrics for the exporter itself.
		promhttp.Handler().ServeHTTP(w, r)
	})

	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		// Queries a GraphQL endpoint and returns the results as Prometheus metrics.
		if err := queryHandler(w, r); err != nil {
			log.Printf("error: %s", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	if EXPORTER_TLS_CERT_FILE != "" && EXPORTER_TLS_KEY_FILE != "" {
		log.Printf("info: listening on https://%s", EXPORTER_LISTEN_ADDR)
		log.Fatalf("critical: %s", http.ListenAndServeTLS(EXPORTER_LISTEN_ADDR, EXPORTER_TLS_CERT_FILE, EXPORTER_TLS_KEY_FILE, nil))
	}
	log.Printf("info: listening on http://%s", EXPORTER_LISTEN_ADDR)
	log.Fatalf("critical: %s", http.ListenAndServe(EXPORTER_LISTEN_ADDR, nil))
}
