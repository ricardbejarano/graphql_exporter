package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	cacheDir       = "/tmp/query-caches"
	queriesDir     = "queries"
	cacheExtension = ".json"
)

var (
	EXPORTER_LISTEN_ADDR   = getenv("EXPORTER_LISTEN_ADDR", "127.0.0.1:9199")
	EXPORTER_TLS_CERT_FILE = getenv("EXPORTER_TLS_CERT_FILE", "")
	EXPORTER_TLS_KEY_FILE  = getenv("EXPORTER_TLS_KEY_FILE", "")
	EXPORTER_GRAPHQL_URL   = getenv("EXPORTER_GRAPHQL_URL", "")
)

var (
	client          = &http.Client{Timeout: 10 * time.Second}
	cacheExpiration = 1 * time.Hour // Set your desired cache expiration time here
	cacheMutex      sync.RWMutex
)

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); len(value) > 0 {
		return value
	}
	return fallback
}

func main() {
	http.HandleFunc("/queries/", handleQuery)

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		// Returns some basic info about the exporter.
		w.Write([]byte("GraphQL exporter for Prometheus.\n"))
		w.Write([]byte("Exporter metrics available at /metrics.\n"))
		w.Write([]byte("Querying available at /queries/<queryfile>.\n\n"))
		w.Write([]byte(fmt.Sprintf("Copyright (c) %s Eduard Marbach\n", time.Now().Format("2006"))))
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Returns metrics for the exporter itself.
		promhttp.Handler().ServeHTTP(w, r)
	})

	log.Printf("info: listening on http://%s", EXPORTER_LISTEN_ADDR)
	log.Fatalf("critical: %s", http.ListenAndServe(EXPORTER_LISTEN_ADDR, nil))
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	queryName := filepath.Base(r.URL.Path)
	queryPath := filepath.Join(queriesDir, queryName+".gql")
	cachePath := filepath.Join(cacheDir, queryName+cacheExtension)

	cachedData, err := readCachedData(cachePath)
	if err == nil && !isCacheExpired(cachePath) {
		fmt.Fprintf(w, string(cachedData))
		return
	}

	queryData, err := ioutil.ReadFile(queryPath)
	if err != nil {
		http.Error(w, "Failed to read query file", http.StatusInternalServerError)
		return
	}

	authToken := r.Header.Get("Authorization")
	result, err := executeGraphQLQuery(string(queryData), authToken)
	if err != nil {
		http.Error(w, "Failed to execute GraphQL query", http.StatusInternalServerError)
		return
	}

	err = writeCachedData(cachePath, result)
	if err != nil {
		fmt.Println("Failed to write cache:", err)
	}

	fmt.Fprintf(w, string(result))
}

func executeGraphQLQuery(query, authToken string) ([]byte, error) {
	url := EXPORTER_GRAPHQL_URL

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func readCachedData(path string) ([]byte, error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func writeCachedData(path string, data []byte) error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	return ioutil.WriteFile(path, data, 0644)
}

func isCacheExpired(path string) bool {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	info, err := os.Stat(path)
	if err != nil {
		return true // Assume cache is expired if unable to get file info
	}

	expirationTime := info.ModTime().Add(cacheExpiration)
	return time.Now().After(expirationTime)
}
