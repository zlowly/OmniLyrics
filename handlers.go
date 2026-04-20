package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/omnilyrics/bridge/smtc"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
}

type StatusHandler func(w http.ResponseWriter, r *http.Request, smtc smtc.SMTC)

func makeStatusHandler(s smtc.SMTC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleStatus(w, r, s)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request, s smtc.SMTC) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	data, err := s.GetData()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"title":    "",
			"artist":   "",
			"status":   "Error",
			"position": 0,
			"duration": 0,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"title":    data.Title,
		"artist":   data.Artist,
		"status":   data.Status,
		"position": data.PositionMs,
		"duration": data.DurationMs,
	})
}

type CacheRequest struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	LRC      string `json:"lrc_content"`
}

func handleCheckCacheWrapper(cacheDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleCheckCache(w, r, cacheDir)
	}
}

func handleCheckCache(w http.ResponseWriter, r *http.Request, cacheDir string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")

	if title == "" && artist == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"found":   false,
			"content": "",
		})
		return
	}

	safeName := sanitizeFilename(artist + "_" + title)
	filePath := filepath.Join(cacheDir, safeName+".lrc")

	if _, err := os.Stat(filePath); err == nil {
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[Error] Read cache file failed: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"found":   false,
				"content": "",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"found":   true,
			"content": string(content),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"found":   false,
		"content": "",
	})
}

func handleUpdateCacheWrapper(cacheDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleUpdateCache(w, r, cacheDir)
	}
}

func handleUpdateCache(w http.ResponseWriter, r *http.Request, cacheDir string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Error] Failed to read body: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request body"})
		return
	}
	defer r.Body.Close()

	log.Printf("[Debug] update_cache received: %s", string(body))

	var req CacheRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[Error] JSON parse failed: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid JSON"})
		return
	}

	if req.Title == "" || req.Artist == "" || req.LRC == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Missing required fields"})
		return
	}

	safeName := sanitizeFilename(req.Artist + "_" + req.Title)
	filePath := filepath.Join(cacheDir, safeName+".lrc")

	if err := os.WriteFile(filePath, []byte(req.LRC), 0644); err != nil {
		log.Printf("[Error] Write cache file failed: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    filePath,
	})
}

func makeSMTCHandler(s smtc.SMTC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		data, err := s.GetData()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "Error",
				"hasSession": false,
				"error":      err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(data)
	}
}

func sanitizeFilename(name string) string {
	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = reg.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if name == "" {
		return "_empty_"
	}
	return name
}

func handleConfigWrapper(configDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleConfig(w, r, configDir)
	}
}

func handleConfig(w http.ResponseWriter, r *http.Request, configDir string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	configPath := filepath.Join(configDir, "renderer.json")

	if r.Method == "GET" {
		if _, err := os.Stat(configPath); err == nil {
			content, err := os.ReadFile(configPath)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.Write(content)
		} else {
			defaultConfig := map[string]interface{}{
				"mode": "karaoke",
				"colors": map[string]interface{}{
					"text":       "#ffffff",
					"glow":       "#00ffff",
					"bg":         "#000000",
					"enableGlow": true,
				},
				"font": map[string]interface{}{
					"size":   "2.4rem",
					"family": "system-ui, -apple-system, Arial",
				},
				"bg": map[string]interface{}{
					"color": "#000000",
				},
				"modeParams": map[string]interface{}{
					"karaoke": map[string]interface{}{
						"wordAnimation":     true,
						"animationDuration": 0.3,
						"currentScale":      1.05,
					},
					"scroll": map[string]interface{}{
						"showNext":      true,
						"nextOpacity":   0.6,
						"scrollDuration": 0.4,
					},
					"blur": map[string]interface{}{
						"visibleLines":  9,
						"lineSpacing":   1.5,
						"opacityDecay":  0.15,
						"blurIncrement": 0.5,
						"scaleDecay":    0.1,
						"blurMax":       6,
						"scrollSpeed":   "linear",
					},
				},
			}
			json.NewEncoder(w).Encode(defaultConfig)
		}
		return
	}

	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read request body"})
			return
		}
		if err := os.WriteFile(configPath, body, 0644); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}
}

func init() {
	fmt.Println("[Handlers] HTTP handlers initialized")
}
