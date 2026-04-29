// tests/lyrics_qm.go
// QQ音乐歌词获取（精简版）
// Usage: go run ./tests/lyrics_qm.go "歌手名" "歌名" [时长秒]

package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ==================== 3DES 解密（保留已验证的自定义实现）====================

var QRC_KEY = []byte("!@#)(*$%123ZXC!@!@#)(NHL") // 24字节密钥

// 预计算的 3DES 解密密钥调度（解密模式，共3组，每组16轮6整数）
var desSchedule [][][]int

func init() {
	desSchedule = tripledesKeySetup(QRC_KEY, DECRYPT)
}

// sbox 定义（完整）
var sbox = [8][64]int{
	{
		14, 4, 13, 1, 2, 15, 11, 8, 3, 10, 6, 12, 5, 9, 0, 7,
		0, 15, 7, 4, 14, 2, 13, 1, 10, 6, 12, 11, 9, 5, 3, 8,
		4, 1, 14, 8, 13, 6, 2, 11, 15, 12, 9, 7, 3, 10, 5, 0,
		15, 12, 8, 2, 4, 9, 1, 7, 5, 11, 3, 14, 10, 0, 6, 13,
	},
	{
		15, 1, 8, 14, 6, 11, 3, 4, 9, 7, 2, 13, 12, 0, 5, 10,
		3, 13, 4, 7, 15, 2, 8, 15, 12, 0, 1, 10, 6, 9, 11, 5,
		0, 14, 7, 11, 10, 4, 13, 1, 5, 8, 12, 6, 9, 3, 2, 15,
		13, 8, 10, 1, 3, 15, 4, 2, 11, 6, 7, 12, 0, 5, 14, 9,
	},
	{
		10, 0, 9, 14, 6, 3, 15, 5, 1, 13, 12, 7, 11, 4, 2, 8,
		13, 7, 0, 9, 3, 4, 6, 10, 2, 8, 5, 14, 12, 11, 15, 1,
		13, 6, 4, 9, 8, 15, 3, 0, 11, 1, 2, 12, 5, 10, 14, 7,
		1, 10, 13, 0, 6, 9, 8, 7, 4, 15, 14, 3, 11, 5, 2, 12,
	},
	{
		7, 13, 14, 3, 0, 6, 9, 10, 1, 2, 8, 5, 11, 12, 4, 15,
		13, 8, 11, 5, 6, 15, 0, 3, 4, 7, 2, 12, 1, 10, 14, 9,
		10, 6, 9, 0, 12, 11, 7, 13, 15, 1, 3, 14, 5, 2, 8, 4,
		3, 15, 0, 6, 10, 10, 13, 8, 9, 4, 5, 11, 12, 7, 2, 14,
	},
	{
		2, 12, 4, 1, 7, 10, 11, 6, 8, 5, 3, 15, 13, 0, 14, 9,
		14, 11, 2, 12, 4, 7, 13, 1, 5, 0, 15, 10, 3, 9, 8, 6,
		4, 2, 1, 11, 10, 13, 7, 8, 15, 9, 12, 5, 6, 3, 0, 14,
		11, 8, 12, 7, 1, 14, 2, 13, 6, 15, 0, 9, 10, 4, 5, 3,
	},
	{
		12, 1, 10, 15, 9, 2, 6, 8, 0, 13, 3, 4, 14, 7, 5, 11,
		10, 15, 4, 2, 7, 12, 9, 5, 6, 1, 13, 14, 0, 11, 3, 8,
		9, 14, 15, 5, 2, 8, 12, 3, 7, 0, 4, 10, 1, 13, 11, 6,
		4, 3, 2, 12, 9, 5, 15, 10, 11, 14, 1, 7, 6, 0, 8, 13,
	},
	{
		4, 11, 2, 14, 15, 0, 8, 13, 3, 12, 9, 7, 5, 10, 6, 1,
		13, 0, 11, 7, 4, 9, 1, 10, 14, 3, 5, 12, 2, 15, 8, 6,
		1, 4, 11, 13, 12, 3, 7, 14, 10, 15, 6, 8, 0, 5, 9, 2,
		6, 11, 13, 8, 1, 4, 10, 7, 9, 5, 0, 15, 14, 2, 3, 12,
	},
	{
		13, 2, 8, 4, 6, 15, 11, 1, 10, 9, 3, 14, 5, 0, 12, 7,
		1, 15, 13, 8, 10, 3, 7, 4, 12, 5, 6, 11, 0, 14, 9, 2,
		7, 11, 4, 1, 9, 12, 14, 2, 0, 6, 10, 13, 15, 3, 5, 8,
		2, 1, 14, 7, 4, 10, 8, 13, 15, 12, 9, 0, 3, 5, 6, 11,
	},
}

const DECRYPT = 0

func bitNum(a []byte, b int, c int) int {
	return int((a[(b/32)*4+3-(b%32)/8]>>(7-b%8))&1) << c
}

func bitNumIntr(a int, b int, c int) int {
	return ((a >> (31 - b)) & 1) << c
}

func bitNumIntl(a int, b int, c int) int {
	return ((a << b) & 0x80000000) >> c
}

func sboxBit(a int) int {
	return (a & 32) | ((a & 31) >> 1) | ((a & 1) << 4)
}

func initialPermutation(inputData []byte) (int, int) {
	var l, r int
	l |= bitNum(inputData, 57, 31)
	l |= bitNum(inputData, 49, 30)
	l |= bitNum(inputData, 41, 29)
	l |= bitNum(inputData, 33, 28)
	l |= bitNum(inputData, 25, 27)
	l |= bitNum(inputData, 17, 26)
	l |= bitNum(inputData, 9, 25)
	l |= bitNum(inputData, 1, 24)
	l |= bitNum(inputData, 59, 23)
	l |= bitNum(inputData, 51, 22)
	l |= bitNum(inputData, 43, 21)
	l |= bitNum(inputData, 35, 20)
	l |= bitNum(inputData, 27, 19)
	l |= bitNum(inputData, 19, 18)
	l |= bitNum(inputData, 11, 17)
	l |= bitNum(inputData, 3, 16)
	l |= bitNum(inputData, 61, 15)
	l |= bitNum(inputData, 53, 14)
	l |= bitNum(inputData, 45, 13)
	l |= bitNum(inputData, 37, 12)
	l |= bitNum(inputData, 29, 11)
	l |= bitNum(inputData, 21, 10)
	l |= bitNum(inputData, 13, 9)
	l |= bitNum(inputData, 5, 8)
	l |= bitNum(inputData, 63, 7)
	l |= bitNum(inputData, 55, 6)
	l |= bitNum(inputData, 47, 5)
	l |= bitNum(inputData, 39, 4)
	l |= bitNum(inputData, 31, 3)
	l |= bitNum(inputData, 23, 2)
	l |= bitNum(inputData, 15, 1)
	l |= bitNum(inputData, 7, 0)

	r |= bitNum(inputData, 56, 31)
	r |= bitNum(inputData, 48, 30)
	r |= bitNum(inputData, 40, 29)
	r |= bitNum(inputData, 32, 28)
	r |= bitNum(inputData, 24, 27)
	r |= bitNum(inputData, 16, 26)
	r |= bitNum(inputData, 8, 25)
	r |= bitNum(inputData, 0, 24)
	r |= bitNum(inputData, 58, 23)
	r |= bitNum(inputData, 50, 22)
	r |= bitNum(inputData, 42, 21)
	r |= bitNum(inputData, 34, 20)
	r |= bitNum(inputData, 26, 19)
	r |= bitNum(inputData, 18, 18)
	r |= bitNum(inputData, 10, 17)
	r |= bitNum(inputData, 2, 16)
	r |= bitNum(inputData, 60, 15)
	r |= bitNum(inputData, 52, 14)
	r |= bitNum(inputData, 44, 13)
	r |= bitNum(inputData, 36, 12)
	r |= bitNum(inputData, 28, 11)
	r |= bitNum(inputData, 20, 10)
	r |= bitNum(inputData, 12, 9)
	r |= bitNum(inputData, 4, 8)
	r |= bitNum(inputData, 62, 7)
	r |= bitNum(inputData, 54, 6)
	r |= bitNum(inputData, 46, 5)
	r |= bitNum(inputData, 38, 4)
	r |= bitNum(inputData, 30, 3)
	r |= bitNum(inputData, 22, 2)
	r |= bitNum(inputData, 14, 1)
	r |= bitNum(inputData, 6, 0)

	return l, r
}

func inversePermutation(s0 int, s1 int) []byte {
	data := make([]byte, 8)

	data[3] = byte(bitNumIntr(s1, 7, 7) | bitNumIntr(s0, 7, 6) | bitNumIntr(s1, 15, 5) |
		bitNumIntr(s0, 15, 4) | bitNumIntr(s1, 23, 3) | bitNumIntr(s0, 23, 2) |
		bitNumIntr(s1, 31, 1) | bitNumIntr(s0, 31, 0))

	data[2] = byte(bitNumIntr(s1, 6, 7) | bitNumIntr(s0, 6, 6) | bitNumIntr(s1, 14, 5) |
		bitNumIntr(s0, 14, 4) | bitNumIntr(s1, 22, 3) | bitNumIntr(s0, 22, 2) |
		bitNumIntr(s1, 30, 1) | bitNumIntr(s0, 30, 0))

	data[1] = byte(bitNumIntr(s1, 5, 7) | bitNumIntr(s0, 5, 6) | bitNumIntr(s1, 13, 5) |
		bitNumIntr(s0, 13, 4) | bitNumIntr(s1, 21, 3) | bitNumIntr(s0, 21, 2) |
		bitNumIntr(s1, 29, 1) | bitNumIntr(s0, 29, 0))

	data[0] = byte(bitNumIntr(s1, 4, 7) | bitNumIntr(s0, 4, 6) | bitNumIntr(s1, 12, 5) |
		bitNumIntr(s0, 12, 4) | bitNumIntr(s1, 20, 3) | bitNumIntr(s0, 20, 2) |
		bitNumIntr(s1, 28, 1) | bitNumIntr(s0, 28, 0))

	data[7] = byte(bitNumIntr(s1, 3, 7) | bitNumIntr(s0, 3, 6) | bitNumIntr(s1, 11, 5) |
		bitNumIntr(s0, 11, 4) | bitNumIntr(s1, 19, 3) | bitNumIntr(s0, 19, 2) |
		bitNumIntr(s1, 27, 1) | bitNumIntr(s0, 27, 0))

	data[6] = byte(bitNumIntr(s1, 2, 7) | bitNumIntr(s0, 2, 6) | bitNumIntr(s1, 10, 5) |
		bitNumIntr(s0, 10, 4) | bitNumIntr(s1, 18, 3) | bitNumIntr(s0, 18, 2) |
		bitNumIntr(s1, 26, 1) | bitNumIntr(s0, 26, 0))

	data[5] = byte(bitNumIntr(s1, 1, 7) | bitNumIntr(s0, 1, 6) | bitNumIntr(s1, 9, 5) |
		bitNumIntr(s0, 9, 4) | bitNumIntr(s1, 17, 3) | bitNumIntr(s0, 17, 2) |
		bitNumIntr(s1, 25, 1) | bitNumIntr(s0, 25, 0))

	data[4] = byte(bitNumIntr(s1, 0, 7) | bitNumIntr(s0, 0, 6) | bitNumIntr(s1, 8, 5) |
		bitNumIntr(s0, 8, 4) | bitNumIntr(s1, 16, 3) | bitNumIntr(s0, 16, 2) |
		bitNumIntr(s1, 24, 1) | bitNumIntr(s0, 24, 0))

	return data
}

func fDES(state int, key []int) int {
	t1 := (bitNumIntl(state, 31, 0) | ((state & 0xf0000000) >> 1) | bitNumIntl(state, 4, 5) |
		bitNumIntl(state, 3, 6) | ((state & 0x0f000000) >> 3) | bitNumIntl(state, 8, 11) |
		bitNumIntl(state, 7, 12) | ((state & 0x00f00000) >> 5) | bitNumIntl(state, 12, 17) |
		bitNumIntl(state, 11, 18) | ((state & 0x000f0000) >> 7) | bitNumIntl(state, 16, 23))

	t2 := (bitNumIntl(state, 15, 0) | ((state & 0x0000f000) << 15) | bitNumIntl(state, 20, 5) |
		bitNumIntl(state, 19, 6) | ((state & 0x00000f00) << 13) | bitNumIntl(state, 24, 11) |
		bitNumIntl(state, 23, 12) | ((state & 0x000000f0) << 11) | bitNumIntl(state, 28, 17) |
		bitNumIntl(state, 27, 18) | ((state & 0x0000000f) << 9) | bitNumIntl(state, 0, 23))

	lrgstate := [6]int{
		(t1 >> 24) & 0x000000ff,
		(t1 >> 16) & 0x000000ff,
		(t1 >> 8) & 0x000000ff,
		(t2 >> 24) & 0x000000ff,
		(t2 >> 16) & 0x000000ff,
		(t2 >> 8) & 0x000000ff,
	}

	for i := 0; i < 6; i++ {
		lrgstate[i] ^= key[i]
	}

	newState := 0
	newState |= sbox[0][sboxBit(lrgstate[0]>>2)] << 28
	newState |= sbox[1][sboxBit(((lrgstate[0]&0x03)<<4)|(lrgstate[1]>>4))] << 24
	newState |= sbox[2][sboxBit(((lrgstate[1]&0x0f)<<2)|(lrgstate[2]>>6))] << 20
	newState |= sbox[3][sboxBit(lrgstate[2]&0x3f)] << 16
	newState |= sbox[4][sboxBit(lrgstate[3]>>2)] << 12
	newState |= sbox[5][sboxBit(((lrgstate[3]&0x03)<<4)|(lrgstate[4]>>4))] << 8
	newState |= sbox[6][sboxBit(((lrgstate[4]&0x0f)<<2)|(lrgstate[5]>>6))] << 4
	newState |= sbox[7][sboxBit(lrgstate[5]&0x3f)]

	return (bitNumIntl(newState, 15, 0) | bitNumIntl(newState, 6, 1) | bitNumIntl(newState, 19, 2) |
		bitNumIntl(newState, 20, 3) | bitNumIntl(newState, 28, 4) | bitNumIntl(newState, 11, 5) |
		bitNumIntl(newState, 27, 6) | bitNumIntl(newState, 16, 7) | bitNumIntl(newState, 0, 8) |
		bitNumIntl(newState, 14, 9) | bitNumIntl(newState, 22, 10) | bitNumIntl(newState, 25, 11) |
		bitNumIntl(newState, 4, 12) | bitNumIntl(newState, 17, 13) | bitNumIntl(newState, 30, 14) |
		bitNumIntl(newState, 9, 15) | bitNumIntl(newState, 1, 16) | bitNumIntl(newState, 7, 17) |
		bitNumIntl(newState, 23, 18) | bitNumIntl(newState, 13, 19) | bitNumIntl(newState, 31, 20) |
		bitNumIntl(newState, 26, 21) | bitNumIntl(newState, 2, 22) | bitNumIntl(newState, 8, 23) |
		bitNumIntl(newState, 18, 24) | bitNumIntl(newState, 12, 25) | bitNumIntl(newState, 29, 26) |
		bitNumIntl(newState, 5, 27) | bitNumIntl(newState, 21, 28) | bitNumIntl(newState, 10, 29) |
		bitNumIntl(newState, 3, 30) | bitNumIntl(newState, 24, 31))
}

func cryptDES(inputData []byte, key [][]int) []byte {
	s0, s1 := initialPermutation(inputData)

	for idx := 0; idx < 15; idx++ {
		prevS1 := s1
		s1 = fDES(s1, key[idx]) ^ s0
		s0 = prevS1
	}
	s0 = fDES(s1, key[15]) ^ s0

	return inversePermutation(s0, s1)
}

func keySchedule(key []byte, mode int) [][]int {
	schedule := make([][]int, 16)
	for i := 0; i < 16; i++ {
		schedule[i] = make([]int, 6)
	}

	keyRndShift := []int{1, 1, 2, 2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 2, 1}
	keyPermC := []int{56, 48, 40, 32, 24, 16, 8, 0, 57, 49, 41, 33, 25, 17, 9, 1, 58, 50, 42, 34, 26, 18, 10, 2, 59, 51, 43, 35}
	keyPermD := []int{62, 54, 46, 38, 30, 22, 14, 6, 61, 53, 45, 37, 29, 21, 13, 5, 60, 52, 44, 36, 28, 20, 12, 4, 27, 19, 11, 3}
	keyCompression := []int{13, 16, 10, 23, 0, 4, 2, 27, 14, 5, 20, 9, 22, 18, 11, 3, 25, 7, 15, 6, 26, 19, 12, 1, 40, 51, 30, 36, 46, 54, 29, 39, 50, 44, 32, 47, 43, 48, 38, 55, 33, 52, 45, 41, 49, 35, 28, 31}

	c := 0
	d := 0
	for i := 0; i < 28; i++ {
		c |= bitNum(key, keyPermC[i], 31-i)
		d |= bitNum(key, keyPermD[i], 31-i)
	}

	for i := 0; i < 16; i++ {
		c = ((c << keyRndShift[i]) | (c >> (28 - keyRndShift[i]))) & 0xfffffff0
		d = ((d << keyRndShift[i]) | (d >> (28 - keyRndShift[i]))) & 0xfffffff0

		togen := 15 - i
		if mode == 1 { // ENCRYPT
			togen = i
		}

		for j := 0; j < 6; j++ {
			schedule[togen][j] = 0
		}

		for j := 0; j < 24; j++ {
			schedule[togen][j/8] |= bitNumIntr(c, keyCompression[j], 7-(j%8))
		}

		for j := 24; j < 48; j++ {
			schedule[togen][j/8] |= bitNumIntr(d, keyCompression[j]-27, 7-(j%8))
		}
	}

	return schedule
}

func tripledesKeySetup(key []byte, mode int) [][][]int {
	if mode == 1 { // ENCRYPT
		return [][][]int{
			keySchedule(key[0:8], 1),
			keySchedule(key[8:16], 0),
			keySchedule(key[16:24], 1),
		}
	}
	return [][][]int{
		keySchedule(key[16:24], 0),
		keySchedule(key[8:16], 1),
		keySchedule(key[0:8], 0),
	}
}

func tripledesCrypt(data []byte, key [][]int) []byte {
	result := make([]byte, 0, len(data))
	for i := 0; i < len(data); i += 8 {
		block := make([]byte, 8)
		copy(block, data[i:])
		if len(block) < 8 {
			block = append(block, make([]byte, 8-len(block))...)
		}
		decrypted := cryptDES(block, key)
		result = append(result, decrypted...)
	}
	return result
}

func tripledes3Crypt(data []byte, key [][][]int) []byte {
	result := data
	for i := 0; i < 3; i++ {
		result = tripledesCrypt(result, key[i])
	}
	return result
}

func qrcDecrypt(encryptedQrc string) (string, error) {
	if encryptedQrc == "" {
		return "", nil
	}
	ciphertext, err := hex.DecodeString(encryptedQrc)
	if err != nil {
		return "", err
	}
	decrypted := tripledes3Crypt(ciphertext, desSchedule)
	r, err := zlib.NewReader(bytes.NewReader(decrypted))
	if err != nil {
		return "", err
	}
	defer r.Close()
	output, err := io.ReadAll(r)
	return string(output), err
}

// ==================== QQ音乐客户端 ====================

type SongInfo struct {
	ID       int64
	Title    string
	Singer   string
	Duration int // 毫秒
}

type QMCloud struct {
	client *http.Client
	comm   map[string]interface{}
}

func newQMCloud() *QMCloud {
	return &QMCloud{
		client: &http.Client{Timeout: 8 * time.Second},
		comm: map[string]interface{}{
			"ct":        11,
			"cv":        "1003006",
			"v":         "1003006",
			"os_ver":    "15",
			"phonetype": "24122RKC7C",
			"rom":       "Redmi/miro/miro:15/AE3A.240806.005/OS2.0.10X",
			"tmeAppID":  "qqmusiclight",
			"nettype":   "NETWORK_WIFI",
			"udid":      "0",
		},
	}
}

func (q *QMCloud) request(method, module string, param map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"comm": q.comm,
		"request": map[string]interface{}{
			"method": method,
			"module": module,
			"param":  param,
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	resp, err := q.client.Post("https://u.y.qq.com/cgi-bin/musicu.fcg", "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: %s", resp.Status)
	}
	body, _ := io.ReadAll(resp.Body)
	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if req, ok := res["request"].(map[string]interface{}); ok {
		if data, ok := req["data"].(map[string]interface{}); ok {
			return data, nil
		}
	}
	return res, nil
}

func (q *QMCloud) GetLyricsForSong(info SongInfo) (string, error) {
	if info.ID == 0 {
		return "", fmt.Errorf("歌曲ID为空")
	}
	param := map[string]interface{}{
		"albumName":  base64.StdEncoding.EncodeToString([]byte("")),
		"crypt":      1,
		"ct":         19,
		"cv":         2111,
		"interval":   info.Duration / 1000,
		"lrc_t":      0,
		"qrc":        1,
		"qrc_t":      0,
		"roma":       1,
		"roma_t":     0,
		"singerName": base64.StdEncoding.EncodeToString([]byte(info.Singer)),
		"songID":     info.ID,
		"songName":   base64.StdEncoding.EncodeToString([]byte(info.Title)),
		"trans":      1,
		"trans_t":    0,
		"type":       0,
	}
	resp, err := q.request("GetPlayLyricInfo", "music.musichallSong.PlayLyricInfo", param)
	if err != nil {
		return "", err
	}
	for _, field := range []string{"orig", "lyric"} {
		if v, ok := resp[field].(string); ok && strings.TrimSpace(v) != "" {
			return decryptLyric(v), nil
		}
	}
	if d, ok := resp["data"].(map[string]interface{}); ok {
		if v, ok := d["lyric"].(string); ok && strings.TrimSpace(v) != "" {
			return decryptLyric(v), nil
		}
	}
	return "", fmt.Errorf("未获取到歌词文本")
}

func decryptLyric(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if result, err := qrcDecrypt(s); err == nil {
		return result
	}
	return s
}

// ==================== QRC 转 LRC ====================

var (
	lineRe = regexp.MustCompile(`^\[(\d+),(\d+)\](.*)$`)
	wordRe = regexp.MustCompile(`([^\(]*)\((\d+),(\d+)\)`)
)

func qrc2lrc(qrcXML string) string {
	start := strings.Index(qrcXML, `LyricContent="`)
	if start < 0 {
		return qrcXML
	}
	start += len(`LyricContent="`)
	end := strings.Index(qrcXML[start:], `"/>`)
	if end < 0 {
		return qrcXML
	}
	content := qrcXML[start : start+end]
	lines := strings.Split(content, "\n")

	var out strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := lineRe.FindStringSubmatch(line)
		if parts == nil {
			continue
		}
		lineStart, _ := strconv.Atoi(parts[1])
		lineContent := parts[3]

		words := wordRe.FindAllStringSubmatch(lineContent, -1)
		if words == nil {
			out.WriteString(fmt.Sprintf("[%s]%s\n", msToTimeStr(lineStart), lineContent))
			continue
		}
		for i, w := range words {
			text := w[1]
			offset, _ := strconv.Atoi(w[2])
			dur, _ := strconv.Atoi(w[3])
			out.WriteString(fmt.Sprintf("[%s]%s", msToTimeStr(offset), text))
			if i == len(words)-1 {
				out.WriteString(fmt.Sprintf("[%s]", msToTimeStr(offset+dur)))
			}
		}
		out.WriteString("\n")
	}
	return out.String()
}

func msToTimeStr(ms int) string {
	return fmt.Sprintf("%02d:%02d.%03d", ms/60000, (ms%60000)/1000, ms%1000)
}

// ==================== 搜索歌曲 ====================

func searchSongs(client *http.Client, comm map[string]interface{}, artist, title string) ([]SongInfo, error) {
	query := strings.TrimSpace(artist + " " + title)
	if query == "" {
		return nil, fmt.Errorf("无效查询词")
	}
	payload := map[string]interface{}{
		"comm": comm,
		"request": map[string]interface{}{
			"method": "DoSearchForQQMusicLite",
			"module": "music.search.SearchCgiService",
			"param": map[string]interface{}{
				"query":        query,
				"search_type":  0,
				"num_per_page": 20,
				"page_num":     1,
			},
		},
	}
	body, _ := json.Marshal(payload)
	resp, err := client.Post("https://u.y.qq.com/cgi-bin/musicu.fcg", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("搜索失败: %s", resp.Status)
	}
	data, _ := io.ReadAll(resp.Body)
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}
	var results []SongInfo
	if req, ok := jsonData["request"].(map[string]interface{}); ok {
		if respData, ok := req["data"].(map[string]interface{}); ok {
			if body, ok := respData["body"].(map[string]interface{}); ok {
				if items, ok := body["item_song"].([]interface{}); ok {
					for _, item := range items {
						if m, ok := item.(map[string]interface{}); ok {
							s := SongInfo{Title: getString(m["title"])}
							if id, ok := toInt64(m["id"]); ok {
								s.ID = id
							}
							if singers, ok := m["singer"].([]interface{}); ok {
								var names []string
								for _, si := range singers {
									if sim, ok := si.(map[string]interface{}); ok {
										if name := getString(sim["name"]); name != "" {
											names = append(names, name)
										}
									}
								}
								s.Singer = strings.Join(names, ", ")
							}
							if dur, ok := toInt(m["interval"]); ok {
								s.Duration = dur * 1000
							}
							results = append(results, s)
						}
					}
				}
			}
		}
	}
	return results, nil
}

func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toInt(v interface{}) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	default:
		return 0, false
	}
}

func toInt64(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int64:
		return t, true
	case float64:
		return int64(t), true
	default:
		return 0, false
	}
}

// ==================== main ====================

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run ./tests/lyrics_qm.go \"歌手\" \"歌名\" [时长秒]")
		os.Exit(2)
	}
	artist, title := os.Args[1], os.Args[2]
	durationSec := 0
	if len(os.Args) == 4 {
		durationSec, _ = strconv.Atoi(os.Args[3])
	}

	cloud := newQMCloud()
	songs, err := searchSongs(cloud.client, cloud.comm, artist, title)
	if err != nil {
		fmt.Fprintln(os.Stderr, "搜索失败:", err)
		os.Exit(1)
	}
	if len(songs) == 0 {
		fmt.Println("未找到歌曲")
		return
	}
	fmt.Printf("找到 %d 首歌曲\n", len(songs))
	for i, s := range songs {
		fmt.Printf("[%d] ID:%d %s - %s (%ds)\n", i+1, s.ID, s.Title, s.Singer, s.Duration/1000)
	}

	selected := songs[0]
	if durationSec > 0 {
		bestIdx, bestDiff := 0, 999999
		for i, s := range songs {
			diff := abs(s.Duration/1000 - durationSec)
			if diff < bestDiff {
				bestDiff, bestIdx = diff, i
			}
		}
		selected = songs[bestIdx]
		fmt.Printf("选择歌曲: %s - %s (%ds)\n", selected.Title, selected.Singer, selected.Duration/1000)
	}

	lyric, err := cloud.GetLyricsForSong(selected)
	if err != nil {
		fmt.Fprintln(os.Stderr, "获取歌词失败:", err)
		os.Exit(1)
	}
	fmt.Println("\nLRC歌词:\n" + qrc2lrc(lyric))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}