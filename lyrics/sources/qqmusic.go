package sources

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/des"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// QRCKey 是 QQ 音乐歌词解密的固定密钥。
const QRCKey = "!@#)(*$%123ZXC!@!@#)(NHL"

// DecryptQRC 解密 QRC 加密的歌词。
// QRC 格式：Hex 编码 → 3DES-EDE 解密 → zlib 解压 → UTF-8 文本。
// 参数：
//   - encryptedHex: 十六进制编码的加密歌词
//
// 返回：
//   - string: 解密后的歌词文本
//   - error: 解密过程中的错误
func DecryptQRC(encryptedHex string) (string, error) {
	// Step 1: Hex 字符串解码为字节数组
	data, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return "", err
	}

	// Step 2: 3DES-EDE 解密
	decrypted, err := tripledesDecrypt(data, []byte(QRCKey))
	if err != nil {
		return "", err
	}

	// Step 3: zlib 解压
	uncompressed, err := zlibInflate(decrypted)
	if err != nil {
		return "", err
	}

	// Step 4: 去除 UTF-8 BOM (如果有)
	uncompressed = bytes.TrimPrefix(uncompressed, []byte{0xEF, 0xBB, 0xBF})

	return string(uncompressed), nil
}

// tripledesDecrypt 执行 3DES-EDE 解密。
func tripledesDecrypt(data, key []byte) ([]byte, error) {
	if len(key) != 24 {
		return nil, errInvalidKeyLength
	}

	blockSize := des.BlockSize
	result := make([]byte, len(data))

	for i := 0; i < len(data); i += blockSize {
		block := data[i : i+blockSize]
		out := make([]byte, blockSize)

		// D(K1)
		desDecrypt(block[:8], key[:8], out[:8])
		// E(K2)
		desEncrypt(out[:8], key[8:16], out[:8])
		// D(K1)
		desDecrypt(out[:8], key[:8], out[:8])

		copy(result[i:], out)
	}

	return result, nil
}

// desDecrypt DES 解密单块
func desDecrypt(in, key, out []byte) {
	block, _ := des.NewCipher(key)
	block.Decrypt(out, in)
}

// desEncrypt DES 加密单块
func desEncrypt(in, key, out []byte) {
	block, _ := des.NewCipher(key)
	block.Encrypt(out, in)
}

// zlibInflate 解压 zlib 压缩的数据。
func zlibInflate(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// errInvalidKeyLength 密钥长度错误
var errInvalidKeyLength = errInvalidKeyLengthErr{}

type errInvalidKeyLengthErr struct{}

func (e errInvalidKeyLengthErr) Error() string {
	return "invalid key length: must be 24 bytes"
}

// QQMusicSource 实现 QQ音乐歌词源。
// 通过 QQ音乐 API 搜索歌曲并获取歌词，支持 QRC 加密歌词解密。
type QQMusicSource struct {
	httpClient *http.Client
}

// NewQQMusicSource 创建新的 QQ音乐歌词源实例。
func NewQQMusicSource() *QQMusicSource {
	return &QQMusicSource{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Name 返回歌词源名称标识。
func (s *QQMusicSource) Name() string {
	return "qqmusic"
}

// qqMusicSearchResponse QQ音乐搜索响应的数据结构。
type qqMusicSearchResponse struct {
	SongMid string `json:"songMid,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Search 根据歌曲信息从 QQ音乐搜索歌词。
// 参数：
//   - ctx: 上下文，用于取消和超时控制
//   - title: 歌曲标题
//   - artist: 艺术家名称
//   - duration: 歌曲时长（秒，QQ音乐源暂未使用此参数）
//
// 返回：
//   - lyrics: LRC 格式的歌词文本
//   - err: 错误信息
func (s *QQMusicSource) Search(ctx context.Context, title, artist string, duration int) (string, error) {
	// 步骤1：搜索歌曲获取 songMid
	songMid, err := s.searchSong(ctx, title, artist)
	if err != nil {
		return "", fmt.Errorf("搜索歌曲失败: %w", err)
	}
	if songMid == "" {
		return "", nil // 未找到歌曲
	}

	log.Printf("[QQMusic] 找到歌曲: songMid=%s", songMid)

	// 步骤2：获取加密歌词
	encrypted, err := s.getEncryptedLyrics(ctx, songMid)
	if err != nil {
		return "", fmt.Errorf("获取歌词失败: %w", err)
	}
	if encrypted == "" {
		return "", nil // 未找到歌词
	}

	// 步骤3：解密 QRC 格式歌词
	lyricsText, err := DecryptQRC(encrypted)
	if err != nil {
		return "", fmt.Errorf("解密歌词失败: %w", err)
	}

	return lyricsText, nil
}

// searchSong 搜索歌曲并返回 songMid。
func (s *QQMusicSource) searchSong(ctx context.Context, title, artist string) (string, error) {
	query := artist + " " + title

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

	body, err := json.Marshal(searchData)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://u.y.qq.com/cgi-bin/musicu.fcg", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("[QQMusic] 搜索响应: %s", string(respBody))

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	// 提取 songMid
	req1, ok := result["req_1"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("响应结构无效: 缺少 req_1")
	}

	data, ok := req1["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("响应结构无效: 缺少 data")
	}

	bodyData, ok := data["body"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("响应结构无效: 缺少 body")
	}

	songData, ok := bodyData["song"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("响应结构无效: 缺少 song")
	}

	songs, ok := songData["list"].([]interface{})
	if !ok || len(songs) == 0 {
		return "", nil // 无搜索结果
	}

	songMid, ok := songs[0].(map[string]interface{})["songmid"].(string)
	if !ok {
		return "", fmt.Errorf("无法提取 songMid")
	}

	return songMid, nil
}

// getEncryptedLyrics 获取加密的歌词内容。
func (s *QQMusicSource) getEncryptedLyrics(ctx context.Context, songMid string) (string, error) {
	// 构造请求参数
	params := map[string]string{
		"callback":    "MusicJsonCallback_lrc",
		"pcachetime":  "0",
		"songmid":     songMid,
		"g_tk":        "5381",
		"jsonpCallback": "MusicJsonCallback_lrc",
		"loginUin":    "0",
		"hostUin":     "0",
		"format":      "jsonp",
		"inCharset":   "utf8",
		"outCharset":  "utf8",
		"notice":      "0",
		"platform":    "yqq",
		"needNewCode": "0",
	}

	// 构建查询字符串
	var queryStr string
	for k, v := range params {
		if queryStr != "" {
			queryStr += "&"
		}
		queryStr += k + "=" + v
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://c.y.qq.com/lyric/fcgi-bin/fcg_query_lyric_new.fcg?"+queryStr, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	lyric, ok := result["lyric"].(string)
	if !ok {
		return "", nil
	}

	return lyric, nil
}
