package main

import (
    "bytes"
    "compress/zlib"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "regexp"
    "strconv"
    "strings"
)

// KRC_KEY：用于 KRC 解密的异或密钥
var KRC_KEY = []byte{0x40, 0x47, 0x61, 0x77, 0x5e, 0x32, 0x74, 0x47, 0x51, 0x36, 0x31, 0x2d, 0xce, 0xd2, 0x6e, 0x69}

// kgmusic 单文件实现的独立歌词源集成（后端代理 + 解密）
// 设计目标：与 QQMusic 源保持风格一致，通过后端代理完成搜索、歌词候选与解密流程

// init 注册 kgmusic 的路由，确保不依赖外部改动即可生效
// RegisterKGMusicRoutes 注册 kgmusic 的后端路由
func RegisterKGMusicRoutes() {
    http.HandleFunc("/proxy/kgmusic/search", kgmusicSearchHandler)
    http.HandleFunc("/proxy/kgmusic/lyric", kgmusicLyricHandler)
    http.HandleFunc("/decrypt-krc", kgmusicDecryptHandler)
}

// kgmusicSearchHandler 处理 kgmusic 的搜索请求
// 请求体：{"title":"<歌名>", "artist":"<歌手>", "duration":<秒>}
// 返回：{ "songId": "", "hash": "", "duration": <ms> }
func kgmusicSearchHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req struct {
        Title    string `json:"title"`
        Artist   string `json:"artist"`
        Duration int    `json:"duration"`
    }
    body, _ := ioutil.ReadAll(r.Body)
    if err := json.Unmarshal(body, &req); err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid JSON"})
        return
    }
    // 调用 Kugou 移动端搜索 API
    baseURL := "http://mobiles.kugou.com/api/v3/search/song"
    params := url.Values{}
    params.Set("showtype", "14")
    params.Set("highlight", "")
    params.Set("pagesize", "30")
    params.Set("tag_aggr", "1")
    params.Set("plat", "0")
    params.Set("sver", "5")
    keyword := strings.TrimSpace(req.Title + " " + req.Artist)
    params.Set("keyword", keyword)
    params.Set("correct", "1")
    params.Set("api_ver", "1")
    params.Set("version", "9108")
    params.Set("page", "1")
    urlWithParams := baseURL + "?" + params.Encode()

    httpReq, _ := http.NewRequest("GET", urlWithParams, nil)
    httpReq.Header.Set("User-Agent", "Mozilla/5.0")
    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
        return
    }
    defer resp.Body.Close()
    respBody, _ := ioutil.ReadAll(resp.Body)
    var result map[string]interface{}
    if err := json.Unmarshal(respBody, &result); err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
        return
    }
    data, _ := result["data"].(map[string]interface{})
    infoList, _ := data["info"].([]interface{})
    if len(infoList) == 0 {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": "no results"})
        return
    }
    first, _ := infoList[0].(map[string]interface{})
    // 直接使用 album_audio_id 作为 songId，并将数值类型转换为字符串
    id := getString(first["album_audio_id"])
    if id == "" {
        // album_audio_id 可能是数字，需要转成字符串
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
    // 若 album_audio_id 无法转为有效字符串，直接返回错误
    if id == "" {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": "songId derivation failed: album_audio_id is not string"})
        return
    }
    hash := getString(first["hash"])
    duration := getInt(first["duration"])
    json.NewEncoder(w).Encode(map[string]interface{}{
        "songId":   id,
        "hash":     hash,
        "duration": duration,
    })
}

// kgmusicLyricHandler 获取歌词候选信息
// 请求体：{ "songId": "", "hash": "", "duration": <ms> }
// 返回：{ "id": "", "accesskey": "", "duration": <ms>, "encrypted": "" }
func kgmusicLyricHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req struct {
        SongID   string `json:"songId"`
        Hash     string `json:"hash"`
        Duration int    `json:"duration"`
    }
    body, _ := ioutil.ReadAll(r.Body)
    if err := json.Unmarshal(body, &req); err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid JSON"})
        return
    }
    urlStr := "http://krcs.kugou.com/search?ver=1&man=no&client=pc&hash=" + url.QueryEscape(req.Hash) + "&album_audio_id=" + url.QueryEscape(req.SongID) + "&lrctxt=1"
    log.Printf("[KGMusic] lyric request: songId=%s, hash=%s, url=%s", req.SongID, req.Hash, urlStr)
    resp, err := http.Get(urlStr)
    if err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
        return
    }
    defer resp.Body.Close()
    data, _ := ioutil.ReadAll(resp.Body)
    log.Printf("[KGMusic] lyric response status=%d, headers=%v, body=%s", resp.StatusCode, resp.Header, string(data))
    var result map[string]interface{}
    if err := json.Unmarshal(data, &result); err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
        return
    }
    candidates, _ := result["candidates"].([]interface{})
    if len(candidates) == 0 {
        json.NewEncoder(w).Encode(map[string]interface{}{"error": "no candidates"})
        return
    }
    c0, _ := candidates[0].(map[string]interface{})
    id := getString(c0["id"])
    accesskey := getString(c0["accesskey"])
    duration := getInt(c0["duration"])

    if id == "" || accesskey == "" {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "id": id, "accesskey": accesskey, "duration": duration, "lyrics": "",
        })
        return
    }

    // 优先下载 KRC 格式（逐字）
    downloadURL := "http://lyrics.kugou.com/download?ver=1&client=pc&id=" + url.QueryEscape(id) + "&accesskey=" + url.QueryEscape(accesskey) + "&fmt=krc"
    resp, err = http.Get(downloadURL)
    if err == nil {
        defer resp.Body.Close()
        data, _ = ioutil.ReadAll(resp.Body)
        var dlResult map[string]interface{}
        if json.Unmarshal(data, &dlResult) == nil {
            if content := getString(dlResult["content"]); content != "" {
                if krcLyrics, err := krcDecryptFromBase64(content); err == nil && krcLyrics != "" {
                    lrcLyrics := krc2lrc(krcLyrics)
                    json.NewEncoder(w).Encode(map[string]interface{}{
                        "id": id, "accesskey": accesskey, "duration": duration, "lyrics": lrcLyrics,
                    })
                    return
                }
            }
        }
    }

    // 降级到 LRC 格式（逐行）
    downloadURL = "http://lyrics.kugou.com/download?ver=1&client=pc&id=" + url.QueryEscape(id) + "&accesskey=" + url.QueryEscape(accesskey) + "&fmt=lrc"
    resp, err = http.Get(downloadURL)
    if err == nil {
        defer resp.Body.Close()
        data, _ = ioutil.ReadAll(resp.Body)
        var dlResult map[string]interface{}
        if json.Unmarshal(data, &dlResult) == nil {
            if content := getString(dlResult["content"]); content != "" {
                if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
                    json.NewEncoder(w).Encode(map[string]interface{}{
                        "id": id, "accesskey": accesskey, "duration": duration, "lyrics": string(decoded),
                    })
                    return
                }
            }
        }
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "id": id, "accesskey": accesskey, "duration": duration, "lyrics": "",
    })
}

// kgmusicDecryptHandler 提供独立的 KRC 解密入口
// 请求体：{ "encrypted": "<base64|hex>" }
// 返回：{ "lyrics": "<明文>" }
func kgmusicDecryptHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req struct{ Encrypted string `json:"encrypted"` }
    body, _ := ioutil.ReadAll(r.Body)
    log.Printf("[KGMusic] decrypt request: headers=%v, body=%s", r.Header, string(body))
    if err := json.Unmarshal(body, &req); err != nil {
        json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
        return
    }
    if req.Encrypted == "" {
        log.Printf("[KGMusic] decrypt: encrypted is empty, returning empty lyrics")
        json.NewEncoder(w).Encode(map[string]string{"lyrics": ""})
        return
    }
    // 使用占位的解密实现，后续替换为完整的 KRC 解密
    log.Printf("[KGMusic] decrypt: encrypted length=%d, first 100 chars=%s", len(req.Encrypted), func() string {
        if len(req.Encrypted) > 100 { return req.Encrypted[:100] }
        return req.Encrypted
    }())
    out, err := krcDecryptFromBase64(req.Encrypted)
    if err != nil {
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"lyrics": out})
}

// ----------------- 共同工具 -----------------
func getString(v interface{}) string {
    if s, ok := v.(string); ok { return s }
    return ""
}
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

// 解密占位实现：直接返回 base64 解码后的文本，实际应替换为完整 KRC 解密
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

func krcDecryptFromBase64(encoded string) (string, error) {
    // 先尝试 base64
    if encoded == "" {
        return "", nil
    }
    if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
        return krcDecrypt(decoded)
    }
    // 否则尝试十六进制字符串解码
    if bytesVal, err2 := hexToBytes(encoded); err2 == nil {
        return krcDecrypt(bytesVal)
    } else {
        return "", err2
    }
}

// downloadLRCFromKugou 使用 id 和 accesskey 从酷狗下载 LRC 格式歌词
func downloadLRCFromKugou(id, accesskey string) (string, error) {
    urlStr := "http://lyrics.kugou.com/download?ver=1&client=pc&id=" + url.QueryEscape(id) + "&accesskey=" + url.QueryEscape(accesskey) + "&fmt=lrc"
    log.Printf("[KGMusic] download LRC request: id=%s, accesskey=%s, url=%s", id, accesskey, urlStr)
    resp, err := http.Get(urlStr)
    if err != nil {
        log.Printf("[KGMusic] download LRC error: %v", err)
        return "", err
    }
    defer resp.Body.Close()

    data, _ := ioutil.ReadAll(resp.Body)
    log.Printf("[KGMusic] download LRC response: status=%d, headers=%v, body=%s", resp.StatusCode, resp.Header, string(data))
    var result map[string]interface{}
    if err := json.Unmarshal(data, &result); err != nil {
        return "", err
    }

    content := getString(result["content"])
    if content == "" {
        return "", nil
    }

    decoded, err := base64.StdEncoding.DecodeString(content)
    if err != nil {
        return "", err
    }
    return string(decoded), nil
}

// krc2lrc 将KRC格式歌词转换为LRC格式
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
