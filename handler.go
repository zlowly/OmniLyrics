package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/omnilyrics/bridge/lyrics"
)

// QQMusicSearchRequest 表示 QQ音乐搜索请求的数据结构。
type QQMusicSearchRequest struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

// QQMusicSearchResponse 表示 QQ音乐搜索响应的数据结构。
type QQMusicSearchResponse struct {
	SongMid string `json:"songMid,omitempty"`
	Error  string `json:"error,omitempty"`
}

// QQMusicLyricRequest 表示 QQ音乐歌词请求的数据结构。
type QQMusicLyricRequest struct {
	SongMid string `json:"songMid"`
}

// QQMusicLyricResponse 表示 QQ音乐歌词响应的数据结构。
type QQMusicLyricResponse struct {
	Encrypted string `json:"encrypted,omitempty"`
	Error     string `json:"error,omitempty"`
}

// handleQQMusicSearch 处理 QQ音乐歌曲搜索请求的 HTTP 端点。
func handleQQMusicSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QQMusicSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(QQMusicSearchResponse{Error: "Invalid JSON"})
		return
	}

	query := req.Artist + " " + req.Title
	searchData := map[string]interface{}{
		"req_1": map[string]interface{}{
			"method": "DoSearchForQQMusicDesktop",
			"module": "music.search.SearchCgiService",
			"param": map[string]interface{}{
				"num_per_page": 10,
				"page_num":    1,
				"query":      query,
				"search_type": 0,
			},
		},
	}

	body, _ := json.Marshal(searchData)
	resp, err := http.Post("https://u.y.qq.com/cgi-bin/musicu.fcg",
		"application/json",
		bytes.NewReader(body))
	if err != nil {
		json.NewEncoder(w).Encode(QQMusicSearchResponse{Error: err.Error()})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("[QQMusic] Search response status: %d, body: %s", resp.StatusCode, string(bodyBytes))

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		log.Printf("[QQMusic] Decode error: %v", err)
		json.NewEncoder(w).Encode(QQMusicSearchResponse{Error: err.Error()})
		return
	}

	log.Printf("[QQMusic] Search response: %+v", result)

	// 安全地提取 songs 列表
	req1, ok := result["req_1"].(map[string]interface{})
	if !ok {
		json.NewEncoder(w).Encode(QQMusicSearchResponse{Error: "Invalid response structure"})
		return
	}

	data, ok := req1["data"].(map[string]interface{})
	if !ok {
		json.NewEncoder(w).Encode(QQMusicSearchResponse{Error: "No data in response"})
		return
	}

	bodyData, ok := data["body"].(map[string]interface{})
	if !ok {
		json.NewEncoder(w).Encode(QQMusicSearchResponse{Error: "No body in response"})
		return
	}

	songData, ok := bodyData["song"].(map[string]interface{})
	if !ok {
		json.NewEncoder(w).Encode(QQMusicSearchResponse{Error: "No song data"})
		return
	}

	songs, ok := songData["list"].([]interface{})
	if !ok || len(songs) == 0 {
		json.NewEncoder(w).Encode(QQMusicSearchResponse{Error: "No results"})
		return
	}

	songMid := songs[0].(map[string]interface{})["songmid"].(string)
	json.NewEncoder(w).Encode(QQMusicSearchResponse{SongMid: songMid})
}

// handleQQMusicLyric 处理 QQ音乐歌词获取请求的 HTTP 端点。
func handleQQMusicLyric(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QQMusicLyricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(QQMusicLyricResponse{Error: "Invalid JSON"})
		return
	}

	if req.SongMid == "" {
		json.NewEncoder(w).Encode(QQMusicLyricResponse{Error: "Missing songMid"})
		return
	}

	data := url.Values{}
	data.Set("callback", "MusicJsonCallback_lrc")
	data.Set("pcachetime", "0")
	data.Set("songmid", req.SongMid)
	data.Set("g_tk", "5381")
	data.Set("jsonpCallback", "MusicJsonCallback_lrc")
	data.Set("loginUin", "0")
	data.Set("hostUin", "0")
	data.Set("format", "jsonp")
	data.Set("inCharset", "utf8")
	data.Set("outCharset", "utf8")
	data.Set("notice", "0")
	data.Set("platform", "yqq")
	data.Set("needNewCode", "0")

	resp, err := http.Get("https://c.y.qq.com/lyric/fcgi-bin/fcg_query_lyric_new.fcg?" + data.Encode())
	if err != nil {
		json.NewEncoder(w).Encode(QQMusicLyricResponse{Error: err.Error()})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		json.NewEncoder(w).Encode(QQMusicLyricResponse{Error: err.Error()})
		return
	}

	lyric := result["lyric"].(string)
	json.NewEncoder(w).Encode(QQMusicLyricResponse{Encrypted: lyric})
}

// DecryptRequest 表示解密请求的数据结构。
type DecryptRequest struct {
	// Encrypted 十六进制编码的加密歌词
	Encrypted string `json:"encrypted"`
}

// DecryptResponse 表示解密响应的数据结构。
type DecryptResponse struct {
	// Lyrics 解密后的歌词文本
	Lyrics string `json:"lyrics"`
	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`
}

// handleDecrypt 处理歌词解密请求的 HTTP 端点。
// 接收 POST 请求，JSON body 包含 encrypted 字段（QRC 加密的十六进制字符串）。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
func handleDecrypt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 仅接受 POST 方法
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req DecryptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Decrypt] Parse failed: %v", err)
		json.NewEncoder(w).Encode(DecryptResponse{Error: "Invalid JSON"})
		return
	}

	// 验证必要字段
	if req.Encrypted == "" {
		json.NewEncoder(w).Encode(DecryptResponse{Error: "Missing encrypted field"})
		return
	}

	// 执行解密
	result, err := lyrics.DecryptQRC(req.Encrypted)
	if err != nil {
		log.Printf("[Decrypt] Decrypt failed: %v", err)
		json.NewEncoder(w).Encode(DecryptResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(DecryptResponse{Lyrics: result})
}