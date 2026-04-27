// tests/lyrics_kg.go
// 单文件实现从酷狗获取歌词的测试命令行工具
// Usage: go run ./tests/lyrics_kg.go krc_decrypt.go "歌手名" "歌名"

package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SongInfo 歌曲信息结构体
// 存储从酷狗搜索结果中解析的歌曲基本信息
type SongInfo struct {
	ID        string // 歌曲专辑音频ID
	Hash      string // 歌曲哈希值，用于唯一标识歌曲
	Title     string // 歌曲标题
	Singer    string // 歌手名称
	Album     string // 专辑名称
	Duration  int    // 歌曲时长（毫秒）
	AccessKey string // 访问密钥，用于获取歌词
}

// LyricsInfo 歌词信息结构体
// 存储搜索结果中的歌词候选信息
type LyricsInfo struct {
	ID        string // 歌词ID
	AccessKey string // 访问密钥，用于下载歌词
	Duration  int    // 歌词对应歌曲时长（毫秒）
	Score     int    // 歌词匹配得分
	Nickname  string // 贡献者昵称
}

// KGClient 酷狗API客户端
// 封装了与酷狗API交互的所有方法
type KGClient struct {
	client *http.Client // HTTP客户端，用于发送请求
	dfid   string       // 设备标识符，通过init()方法获取
}

// newKGClient 创建并初始化KGClient实例
// 返回一个带有默认配置的客户端，dfid会在首次调用init时获取
func newKGClient() *KGClient {
	return &KGClient{
		client: &http.Client{Timeout: 15 * time.Second}, // 15秒超时
		dfid:   "",                                      // 初始为空，会在init时填充
	}
}

// md5Hash 计算给定字符串的MD5哈希值
// 参数s: 要计算哈希的字符串
// 返回: 32位小写十六进制哈希字符串
func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", h)
}

// KRC_KEY 酷狗歌词解密密钥
// 固定16字节，用于XOR解密KRC格式歌词
var KRC_KEY = []byte{0x40, 0x47, 0x61, 0x77, 0x5e, 0x32, 0x74, 0x47, 0x51, 0x36, 0x31, 0x2d, 0xce, 0xd2, 0x6e, 0x69}

// krcDecrypt 解密KRC格式歌词
// KRC是酷狗专用的歌词格式：前4字节是标志，后面是XOR加密+zlib压缩的数据
// 解密流程：跳过前4字节 → XOR解密 → zlib解压 → 返回明文
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

	output, err := ioutil.ReadAll(zlibReader)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// krcDecryptFromBase64 从Base64编码的密文解密歌词
func krcDecryptFromBase64(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return krcDecrypt(decoded)
}

// krc2lrc 将KRC格式歌词转换为LRC格式
// KRC格式: [行起始,持续]<字偏移,字持续,0>字内容
// LRC格式: [mm:ss.xx]字内容
func krc2lrc(krc string) string {
	var result strings.Builder
	lines := strings.Split(krc, "\n")

	linePattern := regexp.MustCompile(`^\[(\d+),(\d+)\](.*)$`)
	// 两种模式：1)无行内标签 <偏移,持续,0>字  2)有行内标签 [行内偏移,行内持续]<偏移,持续,0>字
	wordPatternNoInline := regexp.MustCompile(`<(\d+),(\d+),\d+>([^<]+)`)
	wordPatternInline := regexp.MustCompile(`\[(\d+),(\d+)\]<(\d+),(\d+),\d+>([^<]+)`)

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		// 解析行时间标签: [起始时间,持续时间]内容
		lineMatch := linePattern.FindStringSubmatch(line)
		if lineMatch == nil {
			// 不是时间标签行，可能是标签如 [ti:xxx]，直接复制
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		lineStart, _ := strconv.Atoi(lineMatch[1])
		lineContent := lineMatch[3]

		// 先尝试匹配有行内标签的格式
		wordsInline := wordPatternInline.FindAllStringSubmatch(lineContent, -1)
		// 再尝试匹配无行内标签的格式
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

// init 初始化客户端，获取设备标识符dfid
// 酷狗API需要设备标识符来识别请求来源
// 该方法会调用酷狗的用户服务API获取dfid
// 如果获取失败，会设置默认值为"-"
func (k *KGClient) init() error {
	// 如果已经有dfid，直接返回
	if k.dfid != "" {
		return nil
	}

	// 生成时间戳和mid（设备标识）
	mid := md5Hash(strconv.FormatInt(time.Now().UnixNano()/1e6, 10))

	// 构建API请求参数
	params := url.Values{}
	params.Set("appid", "1014") // 酷狗应用ID
	params.Set("platid", "4")   // 平台ID，4表示移动端
	params.Set("mid", mid)      // 设备mid

	// 计算签名：将appid、mid、appid按顺序拼接后MD5
	// 签名规则：md5(appid + mid + appid)，中间空字符串作为占位
	sortedVals := []string{"1014", mid, "1014"}
	sortedVals = append(sortedVals, "")
	signature := md5Hash(strings.Join(sortedVals, ""))
	params.Set("signature", signature)

	// 构建请求体
	data := base64.StdEncoding.EncodeToString([]byte(`{"uuid":""}`))

	// 发送POST请求获取设备标识
	req, _ := http.NewRequest("POST", "https://userservice.kugou.com/risk/v1/r_register_dev?"+params.Encode(),
		strings.NewReader(data))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 解析响应，提取dfid
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if rd, ok := result["data"].(map[string]interface{}); ok {
		if dfid, ok := rd["dfid"].(string); ok {
			k.dfid = dfid
			return nil
		}
	}
	// 如果解析失败，使用默认值
	k.dfid = "-"
	return nil
}

// getString 安全地从interface{}类型中提取字符串
// 如果类型匹配返回字符串，否则返回空字符串
func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// getString 安全地从interface{}类型中提取整数
// 支持int和float64类型，其他类型返回0
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

// searchSongs 根据关键词搜索歌曲
// 参数keyword: 搜索关键词，通常格式为"歌手名 歌名"
// 返回: 匹配的歌曲列表
// API说明: 使用酷狗移动端API v3版本进行搜索
func (k *KGClient) searchSongs(keyword string, durationSec int) ([]SongInfo, error) {
	baseURL := "http://mobiles.kugou.com/api/v3/search/song"

	// 构建搜索参数
	params := url.Values{}
	params.Set("showtype", "14")   // 显示类型
	params.Set("highlight", "")    // 高亮标记
	params.Set("pagesize", "10")   // 每页返回数量
	params.Set("tag_aggr", "1")    // 标签聚合
	params.Set("plat", "0")        // 平台
	params.Set("sver", "5")        // 软件版本
	params.Set("keyword", keyword) // 搜索关键词
	params.Set("correct", "1")     // 纠错
	params.Set("api_ver", "1")     // API版本
	params.Set("version", "9108")  // 客户端版本
	params.Set("page", "1")        // 页码

	urlWithParams := baseURL + "?" + params.Encode()

	req, _ := http.NewRequest("GET", urlWithParams, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 处理响应，可能返回gzip压缩数据
	var body []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		body, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	} else {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	// 解析JSON响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 提取歌曲列表
	var songs []SongInfo
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return songs, nil
	}
	infoList, ok := data["info"].([]interface{})
	if !ok {
		return songs, nil
	}

	// 遍历并提取每首歌曲的信息
	for _, item := range infoList {
		info, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		// 转换时长从秒转换为毫秒
		songs = append(songs, SongInfo{
			ID:       strconv.Itoa(getInt(info["album_audio_id"])),
			Hash:     getString(info["hash"]),
			Title:    getString(info["songname"]),
			Singer:   getString(info["singername"]),
			Album:    getString(info["album_name"]),
			Duration: getInt(info["duration"]) * 1000,
		})
	}
	// 根据可选的目标时长进行筛选：±2秒的容忍度
	if durationSec > 0 {
		filtered := make([]SongInfo, 0, len(songs))
		for _, s := range songs {
			target := s.Duration / 1000
			diff := target - durationSec
			if diff < 0 {
				diff = -diff
			}
			if diff <= 2 {
				filtered = append(filtered, s)
			}
		}
		// 按与目标时长的差值排序
		sort.Slice(filtered, func(i, j int) bool {
			di := filtered[i].Duration/1000 - durationSec
			if di < 0 {
				di = -di
			}
			dj := filtered[j].Duration/1000 - durationSec
			if dj < 0 {
				dj = -dj
			}
			if di == dj {
				return filtered[i].Duration < filtered[j].Duration
			}
			return di < dj
		})
		songs = filtered
	}
	return songs, nil
}

// getLyricsList 获取指定歌曲的歌词列表
// 参数info: 歌曲信息，包含Hash、Duration等用于查询歌词
// 返回: 歌词候选列表，可能有多个匹配结果
// API说明: 调用酷狗歌词搜索API http://krcs.kugou.com/search
func (k *KGClient) getLyricsList(info SongInfo) ([]LyricsInfo, error) {
	// 构建歌词搜索请求参数（新接口）
	params := url.Values{}
	params.Set("ver", "1")                // API版本
	params.Set("man", "no")               // 手动搜索标记
	params.Set("client", "pc")            // 客户端类型
	params.Set("hash", info.Hash)         // 歌曲哈希
	params.Set("album_audio_id", info.ID) // 专辑音频ID
	params.Set("lrctxt", "1")             // 请求歌词文本

	urlStr := "http://krcs.kugou.com/search?" + params.Encode()

	// 发送GET请求
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 处理响应（gzip压缩）
	var body []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		body, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	} else {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 提取歌词候选列表（从candidates字段）
	var lyricsList []LyricsInfo
	candidates, ok := result["candidates"].([]interface{})
	if !ok {
		return lyricsList, nil
	}

	for _, item := range candidates {
		info, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		lyricsList = append(lyricsList, LyricsInfo{
			ID:        getString(info["id"]),
			AccessKey: getString(info["accesskey"]),
			Duration:  getInt(info["duration"]),
			Score:     getInt(info["score"]),
			Nickname:  getString(info["nickname"]),
		})
	}

	return lyricsList, nil
}

// getLyrics 下载指定歌词
// 参数accesskey: 歌词访问密钥
// 参数id: 歌词ID
// 返回: 解码后的歌词内容
// API说明: 调用酷狗歌词下载API获取KRC格式歌词
func (k *KGClient) getLyrics(accesskey string, id string) (string, error) {
	mid := md5Hash(strconv.FormatInt(time.Now().UnixNano()/1e6, 10))
	clientTime := time.Now().Unix()

	// 构建歌词下载请求参数
	params := map[string]interface{}{
		"accesskey":  accesskey,  // 访问密钥
		"id":         id,         // 歌词ID
		"fmt":        "krc",      // 歌词格式（酷狗专用格式）
		"ver":        "1",        // 版本
		"client":     "mobi",     // 客户端类型
		"charset":    "utf8",     // 字符编码
		"userid":     "0",        // 用户ID
		"clienttime": clientTime, // 客户端时间戳
		"mid":        mid,        // 设备mid
		"dfid":       k.dfid,     // 设备标识
		"clientver":  "11070",    // 客户端版本
		"appid":      "3116",     // 应用ID
	}

	// 转换为URL编码字符串
	qs := url.Values{}
	for pk, pv := range params {
		switch vt := pv.(type) {
		case string:
			qs.Set(pk, vt)
		case int:
			qs.Set(pk, strconv.Itoa(vt))
		case int64:
			qs.Set(pk, strconv.FormatInt(vt, 10))
		}
	}

	urlStr := "https://lyrics.kugou.com/download?" + qs.Encode()

	// 发送GET请求
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", "Android14-1070-11070-201-0-Lyric-wifi")
	req.Header.Set("KG-Rec", "1")
	req.Header.Set("KG-CLIENTTIMEMS", strconv.FormatInt(time.Now().UnixMilli(), 10))
	req.Header.Set("mid", mid)
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := k.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 处理响应（gzip压缩）
	var body []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", err
		}
		defer reader.Close()
		body, err = ioutil.ReadAll(reader)
		if err != nil {
			return "", err
		}
	} else {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// 提取加密的歌词内容（content在根级别）
	encryptedContent, ok := result["content"].(string)
	if !ok {
		return "", fmt.Errorf("响应中无content字段")
	}

	// 解密歌词
	lyrics, err := krcDecryptFromBase64(encryptedContent)
	if err != nil {
		return "", fmt.Errorf("解密失败: %v", err)
	}

	return lyrics, nil
}

// main 主函数，演示从酷狗获取歌词的完整流程
// 使用方法: go run ./tests/lyrics_kg.go "歌手名" "歌名"
func main() {
	// 检查命令行参数，支持可选的时长参数
	if len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Println("Usage: go run ./tests/lyrics_kg.go \"<歌手名>\" \"<歌名>\" [时长秒]")
		fmt.Println("Example: go run ./tests/lyrics_kg.go \"王力宏\" \"心中的日月\" 180")
		os.Exit(2)
	}

	// 获取歌手名和歌曲名
	artist := os.Args[1]
	title := os.Args[2]
	keyword := artist + " " + title

	// 解析可选的时长参数（秒）
	durationSec := 0
	if len(os.Args) == 4 {
		if v, err := strconv.Atoi(os.Args[3]); err == nil {
			durationSec = v
		} else {
			// 非法输入时仍继续，以防止中断搜索
			fmt.Printf("警告: 无法解析时长参数 '%s', 使用默认搜索无时长筛选\n", os.Args[3])
		}
	}

	fmt.Printf("搜索: %s\n", keyword)

	// 创建客户端并初始化
	client := newKGClient()
	if err := client.init(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	songs, err := client.searchSongs(keyword, durationSec)
	if err != nil {
		log.Fatalf("搜索失败: %v", err)
	}

	// 显示搜索结果
	fmt.Printf("找到 %d 首歌曲\n\n", len(songs))
	if len(songs) > 0 {
		for i, s := range songs {
			durationSec := s.Duration / 1000
			fmt.Printf("[%d] ID: %s, Hash: %s, Title: %s, Singer: %s, Album: %s, Duration: %d s\n",
				i+1, s.ID, s.Hash, s.Title, s.Singer, s.Album, durationSec)
		}
	} else {
		fmt.Println("没有找到歌曲")
	}

	// 如果没有找到歌曲，直接返回
	if len(songs) == 0 {
		return
	}

	// 选择第一首歌曲，获取其歌词列表
	firstSong := songs[0]
	fmt.Printf("\n获取歌词列表: %s - %s\n", firstSong.Singer, firstSong.Title)

	lyricsList, err := client.getLyricsList(firstSong)
	if err != nil {
		log.Fatalf("获取歌词列表失败: %v", err)
	}

	// 显示歌词候选列表
	fmt.Printf("找到 %d 个歌词候选\n\n", len(lyricsList))
	for i, l := range lyricsList {
		fmt.Printf("[%d] ID: %s, AccessKey: %s, Duration: %d, Score: %d, Nickname: %s\n",
			i+1, l.ID, l.AccessKey, l.Duration, l.Score, l.Nickname)
	}
	if len(lyricsList) == 0 {
		fmt.Println("没有找到歌词")
		return
	}

	// 选择第一个歌词候选，尝试下载
	firstLyrics := lyricsList[0]
	fmt.Printf("\n下载歌词: ID=%s\n", firstLyrics.ID)

	lyricContent, err := client.getLyrics(firstLyrics.AccessKey, firstLyrics.ID)
	if err != nil {
		log.Fatalf("获取歌词失败: %v", err)
	}

	// 显示原始KRC歌词内容
	fmt.Println("原始KRC歌词内容:")
	fmt.Println(lyricContent)

	// 转换为LRC格式
	fmt.Println("\n转换后的LRC歌词内容:")
	lrcContent := krc2lrc(lyricContent)
	fmt.Println(lrcContent)
}
