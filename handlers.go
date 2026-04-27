package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/omnilyrics/bridge/fonts"
	"github.com/omnilyrics/bridge/smtc"
)

// holdFrozen 标记是否冻结状态（暂停获取新数据）
var holdFrozen bool

// lastStatus 缓存最后一次获取的状态数据
var lastStatus map[string]interface{}

// handleHealth 处理健康检查请求的 HTTP 端点。
// 该端点用于服务监控和负载均衡探测，始终返回 OK 状态。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
}

// StatusHandler 是处理媒体状态请求的函数类型定义。
// 该类型定义了需要访问 SMTC 接口的处理函数签名。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
// @param smtc SMTC 接口实例
type StatusHandler func(w http.ResponseWriter, r *http.Request, smtc smtc.SMTC)

// makeStatusHandler 创建处理媒体状态请求的 HTTP 处理器。
// 该函数返回一个闭包，捕获 SMTC 实例供后续处理使用。
// @param s SMTC 接口实例
// @return http.HandlerFunc 返回配置好的 HTTP 处理函数
func makeStatusHandler(s smtc.SMTC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleStatus(w, r, s)
	}
}

// handleStatus 处理获取媒体状态请求的 HTTP 端点。
// 该端点返回简化版的媒体信息（title、artist、status、position、duration）。
// 当 holdFrozen 为 true 时，返回缓存的最后一帧数据。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
// @param s SMTC 接口实例
func handleStatus(w http.ResponseWriter, r *http.Request, s smtc.SMTC) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 冻结状态：返回缓存数据
	if holdFrozen && lastStatus != nil {
		json.NewEncoder(w).Encode(lastStatus)
		return
	}

	// 获取媒体数据，如果发生错误返回默认空值
	data, err := s.GetData()
	if err != nil {
		result := map[string]interface{}{
			"title":    "",
			"artist":   "",
			"status":   "Error",
			"position": 0,
			"duration": 0,
		}
		lastStatus = result
		json.NewEncoder(w).Encode(result)
		return
	}

	// 返回媒体的简化信息
	result := map[string]interface{}{
		"title":    data.Title,
		"artist":   data.Artist,
		"status":   data.Status,
		"position": data.PositionMs,
		"duration": data.DurationMs,
		"appName":  data.AppName,
	}

	// 如果 title 是 "Unknown"，说明属性获取失败，返回缓存的 title/artist
	if data.Title == "Unknown" && lastStatus != nil {
		result["title"] = lastStatus["title"]
		result["artist"] = lastStatus["artist"]
	}

	lastStatus = result
	json.NewEncoder(w).Encode(result)
}

// handleHold 处理暂停/恢复状态更新的 HTTP 端点。
// 切换 holdFrozen 状态，实现冻结或恢复 status 接口的数据返回。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
func handleHold(w http.ResponseWriter, r *http.Request) {
	holdFrozen = !holdFrozen

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"hold": holdFrozen,
	})
}

// CacheRequest 表示歌词缓存的请求数据结构。
// 前端 POST 歌词数据时使用此结构进行解析。
type CacheRequest struct {
	// Title 媒体标题
	Title string `json:"title"`
	// Artist 艺术家名称
	Artist string `json:"artist"`
	// LRC 歌词内容（LRC 格式）
	LRC string `json:"lrc"`
}

// handleCheckCacheWrapper 创建处理歌词检查请求的 HTTP 处理器。
// 闭包捕获 cacheDir 参数以确定缓存目录位置。
// @param cacheDir 缓存目录路径
// @return http.HandlerFunc 返回配置好的 HTTP 处理函数
func handleCheckCacheWrapper(cacheDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleCheckCache(w, r, cacheDir)
	}
}

// handleCheckCache 处理检查歌词缓存请求的 HTTP 端点。
// 通过 title 和 artist 查询参数查找对应的 .lrc 文件。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
// @param cacheDir 缓存目录路径
func handleCheckCache(w http.ResponseWriter, r *http.Request, cacheDir string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从查询参数获取标题和艺术家
	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")

	// 如果缺少必要参数，返回未找到状态
	if title == "" && artist == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"found":   false,
			"content": "",
		})
		return
	}

	// 将艺术家和标题组合作为文件名，注意去除非法字符
	safeName := sanitizeFilename(artist + "_" + title)
	filePath := filepath.Join(cacheDir, safeName+".lrc")

	// 检查文件是否存在
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
		// 返回找到的歌词内容
		json.NewEncoder(w).Encode(map[string]interface{}{
			"found":   true,
			"content": string(content),
		})
		return
	}

	// 文件不存在，返回未找到
	json.NewEncoder(w).Encode(map[string]interface{}{
		"found":   false,
		"content": "",
	})
}

// handleUpdateCacheWrapper 创建处理歌词更新请求的 HTTP 处理器。
// 闭包捕获 cacheDir 参数以确定缓存目录位置。
// @param cacheDir 缓存目录路径
// @return http.HandlerFunc 返回配置好的 HTTP 处理函数
func handleUpdateCacheWrapper(cacheDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleUpdateCache(w, r, cacheDir)
	}
}

// handleUpdateCache 处理更新歌词缓存请求的 HTTP 端点。
// 接收 POST 请求，JSON body 包含 title、artist 和 lrc_content 字段。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
// @param cacheDir 缓存目录路径
func handleUpdateCache(w http.ResponseWriter, r *http.Request, cacheDir string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 仅接受 POST 方法
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Error] Failed to read body: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request body"})
		return
	}
	defer r.Body.Close()

	log.Printf("[Debug] update_cache received: %s", string(body))

	// 解析 JSON 请求体
	var req CacheRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[Error] JSON parse failed: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid JSON"})
		return
	}

	// 验证必要字段
	if req.Title == "" || req.Artist == "" || req.LRC == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Missing required fields"})
		return
	}

	// 生成安全的文件名并写入缓存
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

// makeSMTCHandler 创建处理 SMTC 原始数据请求的 HTTP 处理器。
// 该端点返回完整的 SMTCData 结构，包含所有字段。
// @param s SMTC 接口实例
// @return http.HandlerFunc 返回配置好的 HTTP 处理函数
func makeSMTCHandler(s smtc.SMTC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 获取完整的 SMTC 数据
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

// sanitizeFilename 将文件名中的非法字符替换为下划线。
// Windows 文件系统不允许以下字符：\ / : * ? " < > |
// @param name 原始文件名
// @return string 返回清理后的安全文件名
func sanitizeFilename(name string) string {
	// 正则匹配所有非法字符并替换为下划线
	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = reg.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	// 确保文件名不为空
	if name == "" {
		return "_empty_"
	}
	return name
}

// handleConfigWrapper 创建处理配置请求的 HTTP 处理器。
// 闭包捕获 configDir 参数以确定配置目录位置。
// @param configDir 配置目录路径
// @return http.HandlerFunc 返回配置好的 HTTP 处理函数
func handleConfigWrapper(configDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleConfig(w, r, configDir)
	}
}

// handleConfig 处理渲染器配置请求的 HTTP 端点。
// GET 请求返回配置（文件中的自定义配置或默认配置）。
// POST 请求保存自定义配置到 renderer.json 文件。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
// @param configDir 配置目录路径
func handleConfig(w http.ResponseWriter, r *http.Request, configDir string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	configPath := filepath.Join(configDir, "renderer.json")

	// GET 请求：读取配置
	if r.Method == "GET" {
		// 尝试读取已保存的配置文件
		if _, err := os.Stat(configPath); err == nil {
			content, err := os.ReadFile(configPath)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			// 直接返回文件内容（保持原格式）
			w.Write(content)
		} else {
			// 文件不存在，返回默认配置
			defaultConfig := map[string]interface{}{
				"mode": "karaoke",
				"colors": map[string]interface{}{
					"text":          "#ffffff",
					"bg":            "#000000",
					"glowRange":     1,
					"outlineWidth":  1,
					"outlineColor": "#ffffff",
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

	// POST 请求：保存配置
	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read request body"})
			return
		}
		// 写入配置文件（保持原始 JSON 格式）
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

// handleLyricsConfigWrapper 创建处理歌词配置请求的 HTTP 处理器。
// 闭包捕获 configDir 参数以确定配置目录位置。
// @param configDir 配置目录路径
// @return http.HandlerFunc 返回配置好的 HTTP 处理函数
func handleLyricsConfigWrapper(configDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleLyricsConfig(w, r, configDir)
	}
}

// handleLyricsConfig 处理歌词配置请求的 HTTP 端点。
// GET 请求返回当前配置（文件中的自定义配置或默认配置）。
// POST 请求保存自定义配置到 lyrics.json 文件。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
// @param configDir 配置目录路径
func handleLyricsConfig(w http.ResponseWriter, r *http.Request, configDir string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	configPath := filepath.Join(configDir, "lyrics.json")

	// GET 请求：读取配置
	if r.Method == "GET" {
		// 尝试读取已保存的配置文件
		if _, err := os.Stat(configPath); err == nil {
			content, err := os.ReadFile(configPath)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.Write(content)
		} else {
			// 文件不存在，返回默认配置
			defaultConfig := map[string]interface{}{
				"timeout": 5000,
				"retry":   1,
				"sources": []map[string]interface{}{
					{"name": "lrclib", "enabled": true, "priority": 1, "apps": []string{"*"}},
					{"name": "qqmusic", "enabled": true, "priority": 2, "apps": []string{"QQMusic.exe", "*"}},
				},
			}
			json.NewEncoder(w).Encode(defaultConfig)
		}
		return
	}

	// POST 请求：保存配置
	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read request body"})
			return
		}
		// 写入配置文件（保持原始 JSON 格式）
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

// handleFonts 处理获取系统字体列表的 HTTP 端点。
// 返回系统已安装的字体名称列表。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
func handleFonts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fonts, err := fonts.GetSystemFonts()
	if err != nil {
		log.Printf("[Error] Failed to get system fonts: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"fonts":   []string{},
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"fonts":   fonts,
	})
}
