package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/omnilyrics/bridge/lyrics"
)

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