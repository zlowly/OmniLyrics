package sources

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zlowly/OmniLyrics/logger"
)

// KRC_KEY 用于 KRC 解密的异或密钥。
var KRC_KEY = []byte{0x40, 0x47, 0x61, 0x77, 0x5e, 0x32, 0x74, 0x47, 0x51, 0x36, 0x31, 0x2d, 0xce, 0xd2, 0x6e, 0x69}

// KGMusicSource 实现酷狗音乐歌词源。
// 通过酷狗 API 搜索歌曲并获取歌词，支持 KRC 和 LRC 格式。
type KGMusicSource struct {
	httpClient *http.Client
}

// NewKGMusicSource 创建新的酷狗音乐歌词源实例。
func NewKGMusicSource() *KGMusicSource {
	return &KGMusicSource{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Name 返回歌词源名称标识。
func (s *KGMusicSource) Name() string {
	return "kgmusic"
}

// Search 根据歌曲信息从酷狗音乐搜索歌词。
// 参数：
//   - ctx: 上下文，用于取消和超时控制
//   - title: 歌曲标题
//   - artist: 艺术家名称
//   - duration: 歌曲时长（秒）
//
// 返回：
//   - lyrics: LRC 格式的歌词文本
//   - err: 错误信息
func (s *KGMusicSource) Search(ctx context.Context, title, artist string, duration int) (string, error) {
	// 步骤1：搜索歌曲
	songID, hash, _, err := s.searchSong(ctx, title, artist)
	if err != nil {
		return "", fmt.Errorf("搜索歌曲失败: %w", err)
	}
	if songID == "" {
		return "", nil // 未找到歌曲
	}

	logger.Infof("[KGMusic] 找到歌曲: songId=%s, hash=%s", songID, hash)

	// 步骤2：获取歌词
	lyrics, err := s.getLyrics(ctx, songID, hash)
	if err != nil {
		return "", fmt.Errorf("获取歌词失败: %w", err)
	}

	return lyrics, nil
}

// searchResult 搜索结果。
type searchResult struct {
	SongID   string `json:"songId"`
	Hash     string `json:"hash"`
	Duration int    `json:"duration"`
}

// searchSong 搜索歌曲并返回歌曲信息。
func (s *KGMusicSource) searchSong(ctx context.Context, title, artist string) (songID, hash string, duration int, err error) {
	baseURL := "http://mobiles.kugou.com/api/v3/search/song"
	params := url.Values{}
	params.Set("showtype", "14")
	params.Set("highlight", "")
	params.Set("pagesize", "30")
	params.Set("tag_aggr", "1")
	params.Set("plat", "0")
	params.Set("sver", "5")
	keyword := strings.TrimSpace(title + " " + artist)
	params.Set("keyword", keyword)
	params.Set("correct", "1")
	params.Set("api_ver", "1")
	params.Set("version", "9108")
	params.Set("page", "1")

	urlWithParams := baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", urlWithParams, nil)
	if err != nil {
		return "", "", 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", 0, err
	}

	data, _ := result["data"].(map[string]interface{})
	infoList, _ := data["info"].([]interface{})
	if len(infoList) == 0 {
		return "", "", 0, nil // 无结果
	}

	first, _ := infoList[0].(map[string]interface{})
	id := getString(first["album_audio_id"])
	if id == "" {
		// 尝试转换数字类型
		if v, ok := first["album_audio_id"]; ok {
			switch t := v.(type) {
			case float64:
				id = strconv.FormatFloat(t, 'f', 0, 64)
			case int:
				id = strconv.Itoa(t)
			case int64:
				id = strconv.FormatInt(t, 10)
			}
		}
	}

	if id == "" {
		return "", "", 0, fmt.Errorf("无法获取 songId")
	}

	hash = getString(first["hash"])
	duration = getInt(first["duration"])

	return id, hash, duration, nil
}

// getLyrics 获取歌词内容，优先 KRC 格式，降级到 LRC 格式。
func (s *KGMusicSource) getLyrics(ctx context.Context, songID, hash string) (string, error) {
	// 先获取歌词候选信息
	id, accesskey, err := s.getLyricCandidate(ctx, songID, hash)
	if err != nil || id == "" {
		return "", err
	}

	// 优先下载 KRC 格式（逐字）
	krcURL := fmt.Sprintf("http://lyrics.kugou.com/download?ver=1&client=pc&id=%s&accesskey=%s&fmt=krc",
		url.QueryEscape(id), url.QueryEscape(accesskey))

	req, err := http.NewRequestWithContext(ctx, "GET", krcURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err == nil {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)

		var dlResult map[string]interface{}
		if json.Unmarshal(data, &dlResult) == nil {
			if content := getString(dlResult["content"]); content != "" {
				if krcLyrics, err := krcDecryptFromBase64(content); err == nil && krcLyrics != "" {
					return krc2lrc(krcLyrics), nil
				}
			}
		}
	}

	// 降级到 LRC 格式（逐行）
	lrcURL := fmt.Sprintf("http://lyrics.kugou.com/download?ver=1&client=pc&id=%s&accesskey=%s&fmt=lrc",
		url.QueryEscape(id), url.QueryEscape(accesskey))

	req, err = http.NewRequestWithContext(ctx, "GET", lrcURL, nil)
	if err != nil {
		return "", err
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var dlResult map[string]interface{}
	if json.Unmarshal(data, &dlResult) == nil {
		if content := getString(dlResult["content"]); content != "" {
			if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
				return string(decoded), nil
			}
		}
	}

	return "", nil
}

// getLyricCandidate 获取歌词候选信息。
func (s *KGMusicSource) getLyricCandidate(ctx context.Context, songID, hash string) (id, accesskey string, err error) {
	urlStr := fmt.Sprintf("http://krcs.kugou.com/search?ver=1&man=no&client=pc&hash=%s&album_audio_id=%s&lrctxt=1",
		url.QueryEscape(hash), url.QueryEscape(songID))

	logger.Debugf("[KGMusic] 歌词候选请求: songId=%s, hash=%s", songID, hash)

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", "", err
	}

	candidates, _ := result["candidates"].([]interface{})
	if len(candidates) == 0 {
		return "", "", nil
	}

	c0, _ := candidates[0].(map[string]interface{})
	id = getString(c0["id"])
	accesskey = getString(c0["accesskey"])

	return id, accesskey, nil
}

// getString 从 interface{} 提取字符串值。
func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// getInt 从 interface{} 提取整数值。
func getInt(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	default:
		return 0
	}
}

// krcDecrypt 解密 KRC 格式歌词。
func krcDecrypt(encryptedLyrics []byte) (string, error) {
	if len(encryptedLyrics) < 4 {
		return "", nil
	}
	encryptedData := encryptedLyrics[4:]
	decryptedData := make([]byte, len(encryptedData))
	for i, item := range encryptedData {
		decryptedData[i] = item ^ KRC_KEY[i%len(KRC_KEY)]
	}
	r := bytes.NewReader(decryptedData)
	zlibReader, err := zlib.NewReader(r)
	if err != nil {
		return "", err
	}
	defer zlibReader.Close()

	output, err := io.ReadAll(zlibReader)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// krcDecryptFromBase64 从 Base64 或十六进制字符串解密 KRC。
func krcDecryptFromBase64(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	// 尝试 Base64 解码
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
		return krcDecrypt(decoded)
	}
	// 尝试十六进制解码
	if bytesVal, err := hexToBytes(encoded); err == nil {
		return krcDecrypt(bytesVal)
	}
	return "", fmt.Errorf("无法解密 KRC 数据")
}

// hexToBytes 将十六进制字符串转换为字节数组。
func hexToBytes(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("无效的十六进制长度")
	}
	res := make([]byte, 0, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		hi := s[i]
		lo := s[i+1]
		var high, low byte
		switch {
		case hi >= '0' && hi <= '9':
			high = hi - '0'
		case hi >= 'a' && hi <= 'f':
			high = hi - 'a' + 10
		case hi >= 'A' && hi <= 'F':
			high = hi - 'A' + 10
		default:
			return nil, fmt.Errorf("无效十六进制字符")
		}
		switch {
		case lo >= '0' && lo <= '9':
			low = lo - '0'
		case lo >= 'a' && lo <= 'f':
			low = lo - 'a' + 10
		case lo >= 'A' && lo <= 'F':
			low = lo - 'A' + 10
		default:
			return nil, fmt.Errorf("无效十六进制字符")
		}
		res = append(res, (high<<4)|low)
	}
	return res, nil
}

// krc2lrc 将 KRC 格式歌词转换为 LRC 格式。
func krc2lrc(krc string) string {
	var result strings.Builder
	lines := strings.Split(krc, "\n")

	linePattern := regexp.MustCompile(`^\[(\d+),(\d+)\](.*)$`)
	wordPatternInline := regexp.MustCompile(`\[(\d+),(\d+)\]<(\d+),(\d+),\d+>([^<]+)`)
	wordPatternNoInline := regexp.MustCompile(`<(\d+),(\d+),\d+>([^<]+)`)

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		lineMatch := linePattern.FindStringSubmatch(line)
		if lineMatch == nil {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		lineStart, _ := strconv.Atoi(lineMatch[1])
		lineContent := lineMatch[3]

		wordsInline := wordPatternInline.FindAllStringSubmatch(lineContent, -1)
		wordsNoInline := wordPatternNoInline.FindAllStringSubmatch(lineContent, -1)

		if len(wordsInline) > 0 || len(wordsNoInline) > 0 {
			var lrcLine strings.Builder
			for _, w := range wordsInline {
				var wordStart int
				inlineOffset, _ := strconv.Atoi(w[1])
				wordOffset, _ := strconv.Atoi(w[3])
				wordStart = lineStart + inlineOffset + wordOffset
				wordContent := strings.TrimSpace(w[6])

				mins := wordStart / 60000
				secs := (wordStart % 60000) / 1000
				ms := (wordStart % 1000) / 10
				fmt.Fprintf(&lrcLine, "[%02d:%02d.%02d]%s", mins, secs, ms, wordContent)
			}
			for _, w := range wordsNoInline {
				var wordStart int
				wordOffset, _ := strconv.Atoi(w[1])
				wordStart = lineStart + wordOffset
				wordContent := strings.TrimSpace(w[3])

				mins := wordStart / 60000
				secs := (wordStart % 60000) / 1000
				ms := (wordStart % 1000) / 10
				fmt.Fprintf(&lrcLine, "[%02d:%02d.%02d]%s", mins, secs, ms, wordContent)
			}
			if lrcLine.Len() > 0 {
				result.WriteString(lrcLine.String())
				result.WriteString("\n")
			}
		} else {
			if lineContent != "" {
				mins := lineStart / 60000
				secs := (lineStart % 60000) / 1000
				ms := (lineStart % 1000) / 10
				fmt.Fprintf(&result, "[%02d:%02d.%02d]%s\n", mins, secs, ms, lineContent)
			}
		}
	}

	return result.String()
}
