package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zlowly/OmniLyrics/logger"
)

// LRCLibSource 实现 LRCLib 歌词源。
// LRCLib 是公开的歌词 API，无需认证即可使用。
type LRCLibSource struct {
	httpClient *http.Client
}

// NewLRCLibSource 创建新的 LRCLib 歌词源实例。
func NewLRCLibSource() *LRCLibSource {
	return &LRCLibSource{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Name 返回歌词源名称标识。
func (s *LRCLibSource) Name() string {
	return "lrclib"
}

// lrclibResponse LRCLib API 响应结构。
type lrclibResponse struct {
	ID             int    `json:"id"`
	TrackName      string `json:"trackName"`
	ArtistName     string `json:"artistName"`
	AlbumName      string `json:"albumName"`
	Duration       int    `json:"duration"`
	Instrumental   bool   `json:"instrumental"`
	PlainLyrics    string `json:"plainLyrics"`
	SyncedLyrics   string `json:"syncedLyrics"`
}

// Search 根据歌曲信息从 LRCLib 搜索歌词。
// 参数：
//   - ctx: 上下文，用于取消和超时控制
//   - title: 歌曲标题
//   - artist: 艺术家名称
//   - duration: 歌曲时长（秒）
//
// 返回：
//   - lyrics: LRC 格式的歌词文本（优先返回 synced，否则返回 plain）
//   - err: 错误信息
func (s *LRCLibSource) Search(ctx context.Context, title, artist string, duration int) (string, error) {
	// 构造查询参数
	params := url.Values{}
	params.Set("artist_name", artist)
	params.Set("track_name", title)
	if duration > 0 {
		params.Set("duration", fmt.Sprintf("%d", duration))
	}

	apiURL := "https://lrclib.net/api/get?" + params.Encode()
	logger.Debugf("[LRCLib] 搜索请求: %s", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", nil // 未找到歌词
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API 返回状态码 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var result lrclibResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 优先返回同步歌词（LRC 格式），否则返回纯文本
	if result.SyncedLyrics != "" {
		return result.SyncedLyrics, nil
	}
	if result.PlainLyrics != "" {
		// 将纯文本转换为简单 LRC 格式
		return plainToLRC(result.PlainLyrics), nil
	}

	return "", nil
}

// plainToLRC 将纯文本歌词转换为简单的 LRC 格式。
// 每行歌词添加相同的时间标签。
func plainToLRC(plain string) string {
	lines := strings.Split(plain, "\n")
	var lrc strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// 使用固定时间标签，因为纯文本没有时间信息
			lrc.WriteString("[00:00.00]")
			lrc.WriteString(line)
		}
		lrc.WriteString("\n")
	}
	return lrc.String()
}
