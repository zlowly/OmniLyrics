package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// main 是程序的入口点，启动 HTTP 服务器并注册所有路由处理器。
// 该函数执行以下操作：
// 1. 初始化配置系统（加载配置文件和命令行参数）
// 2. 确定可执行文件所在的基础目录
// 3. 创建必要的缓存和配置目录
// 4. 初始化 SMTC 后端（根据平台选择 WinRT 或 Mock）
// 5. 注册 HTTP 路由和静态文件服务
// 6. 启动 HTTP 服务器监听指定端口
func main() {
	// 初始化配置系统（包含日志配置）
	if err := initConfig(); err != nil {
		log.Fatalf("[Fatal] Failed to initialize config: %v", err)
	}

	// 获取缓存和配置目录（来自配置系统，支持命令行和配置文件自定义）
	cacheDir := GetCacheDir()
	configDir := GetConfigDir()

	// 创建缓存目录，用于存储歌词文件
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Printf("[Warn] Cannot create Cache dir: %v", err)
	}
	// 创建配置目录，用于存储渲染器配置
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("[Warn] Cannot create Config dir: %v", err)
	}

	// 根据操作系统选择合适的 SMTC 后端
	smtc := NewSMTC()

	// CORS 中间件，为所有响应添加跨域资源共享头
	// 这允许前端从不同域访问 API
	corsHandler := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 允许所有来源的跨域请求
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			// 预检请求直接返回成功
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h(w, r)
		}
	}

	// 注册各路由处理器，路径映射到对应的处理函数
	http.HandleFunc("/health", corsHandler(handleHealth))
	http.HandleFunc("/status", corsHandler(makeStatusHandler(smtc)))
	http.HandleFunc("/hold", corsHandler(handleHold))
	http.HandleFunc("/check_cache", corsHandler(handleCheckCacheWrapper(cacheDir)))
	http.HandleFunc("/update_cache", corsHandler(handleUpdateCacheWrapper(cacheDir)))
	http.HandleFunc("/smtc", corsHandler(makeSMTCHandler(smtc)))
	http.HandleFunc("/decrypt", corsHandler(handleDecrypt))
	http.HandleFunc("/proxy/qqmusic/search", corsHandler(handleQQMusicSearch))
	http.HandleFunc("/proxy/qqmusic/lyric", corsHandler(handleQQMusicLyric))
	http.HandleFunc("/shutdown", corsHandler(handleShutdown))
	http.HandleFunc("/index.html", corsHandler(handleIndex))
	http.HandleFunc("/config", corsHandler(handleConfigWrapper(configDir)))
	http.HandleFunc("/config/lyrics", corsHandler(handleLyricsConfigWrapper(configDir)))
	http.HandleFunc("/fonts", corsHandler(handleFonts))

	// 获取基础目录用于定位 web 资源
	baseDir := getBaseDir()
	webDir := filepath.Join(baseDir, "web")

	// 通配路由处理静态文件服务
	// 根路径 "/" 会尝试查找 index.html，其他路径会映射到 web 目录下的文件
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 安全检查：防止路径遍历攻击
		// 攻击者可能尝试使用 "../" 访问 web 目录外的文件
		path := r.URL.Path
		if strings.Contains(path, "..") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		// 根路径默认返回 index.html
		if path == "/" {
			path = "/index.html"
		}
		// 拼接完整文件路径并尝试服务
		filePath := filepath.Join(webDir, path)
		if _, err := os.Stat(filePath); err == nil {
			http.ServeFile(w, r, filePath)
		} else {
			// 尝试作为目录处理，查找目录下的 index.html
			indexPath := filepath.Join(webDir, path, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
			} else {
				http.NotFound(w, r)
			}
		}
	})

	port := GetPort()
	addr := ":" + port
	log.Printf("[Info] OmniLyrics Bridge starting on http://localhost:%s/", port)
	log.Printf("[Info] Cache dir: %s", cacheDir)
	log.Printf("[Info] Config dir: %s", configDir)

	// 创建带超时的上下文，用于优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 启动 HTTP 服务器
	server := &http.Server{Addr: addr, Handler: nil}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Error] Server error: %v", err)
		}
	}()

	// 监听退出信号：HTTP shutdown 或系统信号
	select {
	case <-ctx.Done():
		log.Println("[Info] Signal received, stopping server...")
	case <-shutdownCh:
		log.Println("[Info] Shutdown requested via HTTP, stopping server...")
	}

	// 强制关闭服务器，立即中断所有连接
	if err := server.Close(); err != nil {
		log.Printf("[Error] Server close error: %v", err)
	}
	log.Println("[Info] Server stopped")
}

// getBaseDir 获取程序的基础目录。
// 优先级：当前工作目录 > 可执行文件所在目录 > 当前目录
// @return string 返回基础目录路径
func getBaseDir() string {
	// 优先使用当前工作目录，便于开发调试
	// 当使用 go run . 时，会使用运行命令时所在目录
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	// 其次使用可执行文件所在目录，适用于打包后的二进制
	if exePath, err := os.Executable(); err == nil {
		return filepath.Dir(exePath)
	}
	// 最后使用当前目录
	return "."
}

// shutdownCh 用于协调关闭信号（HTTP shutdown 或系统信号）
var shutdownCh = make(chan struct{})

// handleShutdown 处理关机请求的 HTTP 端点。
// 接受 GET、POST 等所有 HTTP 方法，执行服务器关闭。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
func handleShutdown(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"status":"shutting_down"}`))
	close(shutdownCh)
}

// handleIndex 处理 index.html 文件请求的 HTTP 端点。
// @param w HTTP 响应写入器
// @param r HTTP 请求对象
func handleIndex(w http.ResponseWriter, r *http.Request) {
	baseDir := getBaseDir()
	http.ServeFile(w, r, filepath.Join(baseDir, "web", "index.html"))
}
