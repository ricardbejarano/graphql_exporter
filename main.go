package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"errors"

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
	EXPORTER_GRAPHQL_AUTH  = getenv("EXPORTER_GRAPHQL_AUTH", "")
	EXPORTER_CACHE_MINUTES = getenvInt("EXPORTER_CACHE_MINUTES", 60)
)

var (
	client          = &http.Client{Timeout: 20 * time.Second}
	cacheExpiration = parseDuration(fmt.Sprintf("%dm", EXPORTER_CACHE_MINUTES))
	cacheMutex      sync.RWMutex
)

var ErrEnvVarEmpty = errors.New("getenv: environment variable empty")

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		fmt.Println("Error parsing duration:", err)
	}
	return d
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); len(value) > 0 {
		return value
	}
	return fallback
}

func getenvStr(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return v, ErrEnvVarEmpty
	}
	return v, nil
}

func getenvInt(key string, fallback int) int {
	s, err := getenvStr(key)
	if err != nil {
		fmt.Printf("getenvInt: Error getting value for key %s. Using default %d\n", key, fallback)
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		fmt.Printf("getenvInt: Error converting key %s\n using default %d\n", key, fallback)
		return fallback
	}
	return v
}

func main() {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		err := os.Mkdir(cacheDir, os.ModePerm)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

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

	//authToken := r.Header.Get("Authorization")
	result, err := executeGraphQLQuery(string(queryData), EXPORTER_GRAPHQL_AUTH)
	if err != nil {
		http.Error(w, "Failed to execute GraphQL query", http.StatusInternalServerError)
		return
	}

	err = writeCachedData(cachePath, result)
	if err != nil {
		fmt.Println("Failed to write cache:", err)
	}

	fmt.Printf("Refreshed cache for path: %s", queryPath)

	fmt.Fprintf(w, string(result))
}

// TODO authToken
func executeGraphQLQuery(query, authToken string) ([]byte, error) {
	url := EXPORTER_GRAPHQL_URL

	reqBody, err := json.Marshal(map[string]string{
		"query": string(query),
	})

	if err != nil {
		fmt.Println("Error constructing request body:", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(reqBody)))
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
