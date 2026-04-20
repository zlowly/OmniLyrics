package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	baseDir := getBaseDir()
	cacheDir := filepath.Join(baseDir, "Cache")
	configDir := filepath.Join(baseDir, "Config")

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Printf("[Warn] Cannot create Cache dir: %v", err)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("[Warn] Cannot create Config dir: %v", err)
	}

	smtc := NewSMTC()

	// CORS middleware
	corsHandler := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h(w, r)
		}
	}

	http.HandleFunc("/health", corsHandler(handleHealth))
	http.HandleFunc("/status", corsHandler(makeStatusHandler(smtc)))
	http.HandleFunc("/check_cache", corsHandler(handleCheckCacheWrapper(cacheDir)))
	http.HandleFunc("/update_cache", corsHandler(handleUpdateCacheWrapper(cacheDir)))
	http.HandleFunc("/smtc", corsHandler(makeSMTCHandler(smtc)))
	http.HandleFunc("/shutdown", corsHandler(handleShutdown))
	http.HandleFunc("/index.html", corsHandler(handleIndex))
	http.HandleFunc("/config", corsHandler(handleConfigWrapper(configDir)))

	webDir := filepath.Join(baseDir, "web")

	// Serve static files from web directory
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Security check: prevent path traversal
		path := r.URL.Path
		if strings.Contains(path, "..") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if path == "/" {
			path = "/index.html"
		}
		// Remove leading slash for file serving
		filePath := filepath.Join(webDir, path)
		if _, err := os.Stat(filePath); err == nil {
			http.ServeFile(w, r, filePath)
		} else {
			// Try index.html for directory-like paths
			indexPath := filepath.Join(webDir, path, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
			} else {
				http.NotFound(w, r)
			}
		}
	})

	addr := ":8080"
	log.Printf("[Info] OmniLyrics Bridge starting on http://localhost%s/", addr)
	log.Printf("[Info] Cache dir: %s", cacheDir)
	log.Printf("[Info] Config dir: %s", configDir)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Printf("[Error] Server error: %v", err)
	}
}

func getBaseDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exePath)
}

var serverRunning = true

func handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" || r.URL.Query().Get("confirm") == "1" {
		w.Write([]byte(`{"status":"shutting_down"}`))
		serverRunning = false
		go func() {
			log.Println("[Info] Shutdown requested")
			os.Exit(0)
		}()
	}
	w.Write([]byte(`{"status":"ok"}`))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	baseDir := getBaseDir()
	http.ServeFile(w, r, filepath.Join(baseDir, "web", "index.html"))
}
