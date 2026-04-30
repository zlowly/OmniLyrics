package sources

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zlowly/OmniLyrics/logger"
)

// QRCKey 是 QQ 音乐歌词解密的固定密钥。
const QRCKey = "!@#)(*$%123ZXC!@!@#)(NHL"

// 以下实现QQ音乐使用的自定义3DES-EDE解密算法
// 参考Python实现: https://github.com/chenmozhijin/LDDC/blob/main/LDDC/core/decryptor/tripledes.py

// sbox 是Python代码中的S盒定义
var sbox = [8][64]int{
	// sbox1
	{14, 4, 13, 1, 2, 15, 11, 8, 3, 10, 6, 12, 5, 9, 0, 7,
		0, 15, 7, 4, 14, 2, 13, 1, 10, 6, 12, 11, 9, 5, 3, 8,
		4, 1, 14, 8, 13, 6, 2, 11, 15, 12, 9, 7, 3, 10, 5, 0,
		15, 12, 8, 2, 4, 9, 1, 7, 5, 11, 3, 14, 10, 0, 6, 13},
	// sbox2
	{15, 1, 8, 14, 6, 11, 3, 4, 9, 7, 2, 13, 12, 0, 5, 10,
		3, 13, 4, 7, 15, 2, 8, 15, 12, 0, 1, 10, 6, 9, 11, 5,
		0, 14, 7, 11, 10, 4, 13, 1, 5, 8, 12, 6, 9, 3, 2, 15,
		13, 8, 10, 1, 3, 15, 4, 2, 11, 6, 7, 12, 0, 5, 14, 9},
	// sbox3
	{10, 0, 9, 14, 6, 3, 15, 5, 1, 13, 12, 7, 11, 4, 2, 8,
		13, 7, 0, 9, 3, 4, 6, 10, 2, 8, 5, 14, 12, 11, 15, 1,
		13, 6, 4, 9, 8, 15, 3, 0, 11, 1, 2, 12, 5, 10, 14, 7,
		1, 10, 13, 0, 6, 9, 8, 7, 4, 15, 14, 3, 11, 5, 2, 12},
	// sbox4
	{7, 13, 14, 3, 0, 6, 9, 10, 1, 2, 8, 5, 11, 12, 4, 15,
		13, 8, 11, 5, 6, 15, 0, 3, 4, 7, 2, 12, 1, 10, 14, 9,
		10, 6, 9, 0, 12, 11, 7, 13, 15, 1, 3, 14, 5, 2, 8, 4,
		3, 15, 0, 6, 10, 10, 13, 8, 9, 4, 5, 11, 12, 7, 2, 14},
	// sbox5
	{2, 12, 4, 1, 7, 10, 11, 6, 8, 5, 3, 15, 13, 0, 14, 9,
		14, 11, 2, 12, 4, 7, 13, 1, 5, 0, 15, 10, 3, 9, 8, 6,
		4, 2, 1, 11, 10, 13, 7, 8, 15, 9, 12, 5, 6, 3, 0, 14,
		11, 8, 12, 7, 1, 14, 2, 13, 6, 15, 0, 9, 10, 4, 5, 3},
	// sbox6
	{12, 1, 10, 15, 9, 2, 6, 8, 0, 13, 3, 4, 14, 7, 5, 11,
		10, 15, 4, 2, 7, 12, 9, 5, 6, 1, 13, 14, 0, 11, 3, 8,
		9, 14, 15, 5, 2, 8, 12, 3, 7, 0, 4, 10, 1, 13, 11, 6,
		4, 3, 2, 12, 9, 5, 15, 10, 11, 14, 1, 7, 6, 0, 8, 13},
	// sbox7
	{4, 11, 2, 14, 15, 0, 8, 13, 3, 12, 9, 7, 5, 10, 6, 1,
		13, 0, 11, 7, 4, 9, 1, 10, 14, 3, 5, 12, 2, 15, 8, 6,
		1, 4, 11, 13, 12, 3, 7, 14, 10, 15, 6, 8, 0, 5, 9, 2,
		6, 11, 13, 8, 1, 4, 10, 7, 9, 5, 0, 15, 14, 2, 3, 12},
	// sbox8
	{13, 2, 8, 4, 6, 15, 11, 1, 10, 9, 3, 14, 5, 0, 12, 7,
		1, 15, 13, 8, 10, 3, 7, 4, 12, 5, 6, 11, 0, 14, 9, 2,
		7, 11, 4, 1, 9, 12, 14, 2, 0, 6, 10, 13, 15, 3, 5, 8,
		2, 1, 14, 7, 4, 10, 8, 13, 15, 12, 9, 0, 3, 5, 6, 11},
}

// keyPermC 是密钥初始置换表C
var keyPermC = []int{
	56, 48, 40, 32, 24, 16, 8, 0, 57, 49, 41, 33, 25, 17, 9, 1,
	58, 50, 42, 34, 26, 18, 10, 2, 59, 51, 43, 35, 62, 54, 46, 38,
	30, 22, 14, 6, 61, 53, 45, 37, 29, 21, 13, 5, 60, 52, 44, 36,
	28, 20, 12, 4, 63, 55, 47, 39, 31, 23, 15, 7,
}

// keyPermD 是密钥初始置换表D
var keyPermD = []int{
	62, 54, 46, 38, 30, 22, 14, 6, 61, 53, 45, 37, 29, 21, 13, 5,
	60, 52, 44, 36, 28, 20, 12, 4, 27, 19, 11, 3, 59, 51, 43, 35,
	40, 32, 24, 16, 8, 0, 57, 49, 41, 33, 25, 17, 9, 1, 58, 50,
	42, 34, 26, 18, 10, 2, 63, 55, 47, 39, 31, 23, 15, 7,
}

// keyCompression 是密钥压缩置换表
var keyCompression = []int{
	13, 16, 10, 23, 0, 4, 2, 27, 14, 5, 20, 9, 22, 18, 11, 3,
	25, 7, 15, 6, 26, 19, 12, 1, 40, 51, 30, 36, 46, 54, 29, 39,
	50, 44, 32, 47, 43, 48, 38, 55, 33, 52, 45, 41, 49, 35, 28, 31,
}

// keyRndShift 是密钥轮移位表
var keyRndShift = []int{1, 1, 2, 2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 2, 1}

// initialPerm 是初始置换表
var initialPerm = []int{
	57, 49, 41, 33, 25, 17, 9, 1, 59, 51, 43, 35, 27, 19, 11, 3,
	61, 53, 45, 37, 29, 21, 13, 5, 63, 55, 47, 39, 31, 23, 15, 7,
	56, 48, 40, 32, 24, 16, 8, 0, 58, 50, 42, 34, 26, 18, 10, 2,
	60, 52, 44, 36, 28, 20, 12, 4, 62, 54, 46, 38, 30, 22, 14, 6,
}

// inversePerm 是逆置换表
var inversePerm = []int{
	39, 7, 47, 15, 55, 23, 63, 31, 38, 6, 46, 14, 54, 22, 62, 30,
	37, 5, 45, 13, 53, 21, 61, 29, 36, 4, 44, 12, 52, 20, 60, 28,
	35, 3, 43, 11, 51, 19, 59, 27, 34, 2, 42, 10, 50, 18, 58, 26,
	33, 1, 41, 9, 49, 17, 57, 25, 32, 0, 40, 8, 48, 16, 56, 24,
}

// bitnum 从字节串中提取指定位置的位
func bitnum(a []byte, b, c int) int {
	byteIdx := (b/32)*4 + 3 - (b%32)/8
	bit := int((a[byteIdx] >> (7 - b%8)) & 1)
	return bit << c
}

// bitnumIntr 从整数中提取指定位置的位
func bitnumIntr(a int, b, c int) int {
	return ((a >> (31 - b)) & 1) << c
}

// bitnumIntl 从整数中提取指定位置的位并左移
func bitnumIntl(a int, b, c int) int {
	return ((a << b) & 0x80000000) >> c
}

// sboxBit 对输入进行S盒位运算
func sboxBit(a int) int {
	return (a & 32) | ((a & 31) >> 1) | ((a & 1) << 4)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// initialPermutation 初始置换，使用 DES 标准初始置换表。
// 直接参考 tests/lyrics_qm.go 的实现（1-indexed 位置）。
func initialPermutation(input []byte) (int, int) {
	var l, r int
	// 左半部分 L (前32位输出)
	l |= bitnum(input, 57, 31)
	l |= bitnum(input, 49, 30)
	l |= bitnum(input, 41, 29)
	l |= bitnum(input, 33, 28)
	l |= bitnum(input, 25, 27)
	l |= bitnum(input, 17, 26)
	l |= bitnum(input, 9, 25)
	l |= bitnum(input, 1, 24)
	l |= bitnum(input, 59, 23)
	l |= bitnum(input, 51, 22)
	l |= bitnum(input, 43, 21)
	l |= bitnum(input, 35, 20)
	l |= bitnum(input, 27, 19)
	l |= bitnum(input, 19, 18)
	l |= bitnum(input, 11, 17)
	l |= bitnum(input, 3, 16)
	l |= bitnum(input, 61, 15)
	l |= bitnum(input, 53, 14)
	l |= bitnum(input, 45, 13)
	l |= bitnum(input, 37, 12)
	l |= bitnum(input, 29, 11)
	l |= bitnum(input, 21, 10)
	l |= bitnum(input, 13, 9)
	l |= bitnum(input, 5, 8)
	l |= bitnum(input, 63, 7)
	l |= bitnum(input, 55, 6)
	l |= bitnum(input, 47, 5)
	l |= bitnum(input, 39, 4)
	l |= bitnum(input, 31, 3)
	l |= bitnum(input, 23, 2)
	l |= bitnum(input, 15, 1)
	l |= bitnum(input, 7, 0)

	// 右半部分 R (后32位输出)
	r |= bitnum(input, 56, 31)
	r |= bitnum(input, 48, 30)
	r |= bitnum(input, 40, 29)
	r |= bitnum(input, 32, 28)
	r |= bitnum(input, 24, 27)
	r |= bitnum(input, 16, 26)
	r |= bitnum(input, 8, 25)
	r |= bitnum(input, 0, 24)
	r |= bitnum(input, 58, 23)
	r |= bitnum(input, 50, 22)
	r |= bitnum(input, 42, 21)
	r |= bitnum(input, 34, 20)
	r |= bitnum(input, 26, 19)
	r |= bitnum(input, 18, 18)
	r |= bitnum(input, 10, 17)
	r |= bitnum(input, 2, 16)
	r |= bitnum(input, 60, 15)
	r |= bitnum(input, 52, 14)
	r |= bitnum(input, 44, 13)
	r |= bitnum(input, 36, 12)
	r |= bitnum(input, 28, 11)
	r |= bitnum(input, 20, 10)
	r |= bitnum(input, 12, 9)
	r |= bitnum(input, 4, 8)
	r |= bitnum(input, 62, 7)
	r |= bitnum(input, 54, 6)
	r |= bitnum(input, 46, 5)
	r |= bitnum(input, 38, 4)
	r |= bitnum(input, 30, 3)
	r |= bitnum(input, 22, 2)
	r |= bitnum(input, 14, 1)
	r |= bitnum(input, 6, 0)

	return l, r
}

// inversePermutation 逆置换
func inversePermutation(s0, s1 int) []byte {
	data := make([]byte, 8)
	data[3] = byte(bitnumIntr(s1, 7, 7) | bitnumIntr(s0, 7, 6) | bitnumIntr(s1, 15, 5) |
		bitnumIntr(s0, 15, 4) | bitnumIntr(s1, 23, 3) | bitnumIntr(s0, 23, 2) |
		bitnumIntr(s1, 31, 1) | bitnumIntr(s0, 31, 0))

	data[2] = byte(bitnumIntr(s1, 6, 7) | bitnumIntr(s0, 6, 6) | bitnumIntr(s1, 14, 5) |
		bitnumIntr(s0, 14, 4) | bitnumIntr(s1, 22, 3) | bitnumIntr(s0, 22, 2) |
		bitnumIntr(s1, 30, 1) | bitnumIntr(s0, 30, 0))

	data[1] = byte(bitnumIntr(s1, 5, 7) | bitnumIntr(s0, 5, 6) | bitnumIntr(s1, 13, 5) |
		bitnumIntr(s0, 13, 4) | bitnumIntr(s1, 21, 3) | bitnumIntr(s0, 21, 2) |
		bitnumIntr(s1, 29, 1) | bitnumIntr(s0, 29, 0))

	data[0] = byte(bitnumIntr(s1, 4, 7) | bitnumIntr(s0, 4, 6) | bitnumIntr(s1, 12, 5) |
		bitnumIntr(s0, 12, 4) | bitnumIntr(s1, 20, 3) | bitnumIntr(s0, 20, 2) |
		bitnumIntr(s1, 28, 1) | bitnumIntr(s0, 28, 0))

	data[7] = byte(bitnumIntr(s1, 3, 7) | bitnumIntr(s0, 3, 6) | bitnumIntr(s1, 11, 5) |
		bitnumIntr(s0, 11, 4) | bitnumIntr(s1, 19, 3) | bitnumIntr(s0, 19, 2) |
		bitnumIntr(s1, 27, 1) | bitnumIntr(s0, 27, 0))

	data[6] = byte(bitnumIntr(s1, 2, 7) | bitnumIntr(s0, 2, 6) | bitnumIntr(s1, 10, 5) |
		bitnumIntr(s0, 10, 4) | bitnumIntr(s1, 18, 3) | bitnumIntr(s0, 18, 2) |
		bitnumIntr(s1, 26, 1) | bitnumIntr(s0, 26, 0))

	data[5] = byte(bitnumIntr(s1, 1, 7) | bitnumIntr(s0, 1, 6) | bitnumIntr(s1, 9, 5) |
		bitnumIntr(s0, 9, 4) | bitnumIntr(s1, 17, 3) | bitnumIntr(s0, 17, 2) |
		bitnumIntr(s1, 25, 1) | bitnumIntr(s0, 25, 0))

	data[4] = byte(bitnumIntr(s1, 0, 7) | bitnumIntr(s0, 0, 6) | bitnumIntr(s1, 8, 5) |
		bitnumIntr(s0, 8, 4) | bitnumIntr(s1, 16, 3) | bitnumIntr(s0, 16, 2) |
		bitnumIntr(s1, 24, 1) | bitnumIntr(s0, 24, 0))
	return data
}

// f 函数 (Feistel轮函数)
func f(state int, key []int) int {
	t1 := (bitnumIntl(state, 31, 0) | ((state & 0xf0000000) >> 1) | bitnumIntl(state, 4, 5) |
		bitnumIntl(state, 3, 6) | ((state & 0x0f000000) >> 3) | bitnumIntl(state, 8, 11) |
		bitnumIntl(state, 7, 12) | ((state & 0x00f00000) >> 5) | bitnumIntl(state, 12, 17) |
		bitnumIntl(state, 11, 18) | ((state & 0x000f0000) >> 7) | bitnumIntl(state, 16, 23))

	t2 := (bitnumIntl(state, 15, 0) | ((state & 0x0000f000) << 15) | bitnumIntl(state, 20, 5) |
		bitnumIntl(state, 19, 6) | ((state & 0x00000f00) << 13) | bitnumIntl(state, 24, 11) |
		bitnumIntl(state, 23, 12) | ((state & 0x000000f0) << 11) | bitnumIntl(state, 28, 17) |
		bitnumIntl(state, 27, 18) | ((state & 0x0000000f) << 9) | bitnumIntl(state, 0, 23))

	lrgstate := []int{
		(t1 >> 24) & 0x000000ff, (t1 >> 16) & 0x000000ff, (t1 >> 8) & 0x000000ff,
		(t2 >> 24) & 0x000000ff, (t2 >> 16) & 0x000000ff, (t2 >> 8) & 0x000000ff,
	}

	for i := 0; i < 6; i++ {
		lrgstate[i] ^= key[i]
	}

	state = ((sbox[0][sboxBit(lrgstate[0]>>2)] << 28) |
		(sbox[1][sboxBit(((lrgstate[0]&0x03)<<4)|(lrgstate[1]>>4))] << 24) |
		(sbox[2][sboxBit(((lrgstate[1]&0x0f)<<2)|(lrgstate[2]>>6))] << 20) |
		(sbox[3][sboxBit(lrgstate[2]&0x3f)] << 16) |
		(sbox[4][sboxBit(lrgstate[3]>>2)] << 12) |
		(sbox[5][sboxBit(((lrgstate[3]&0x03)<<4)|(lrgstate[4]>>4))] << 8) |
		(sbox[6][sboxBit(((lrgstate[4]&0x0f)<<2)|(lrgstate[5]>>6))] << 4) |
		sbox[7][sboxBit(lrgstate[5]&0x3f)])

	state = (bitnumIntl(state, 15, 0) | bitnumIntl(state, 6, 1) | bitnumIntl(state, 19, 2) |
		bitnumIntl(state, 20, 3) | bitnumIntl(state, 28, 4) | bitnumIntl(state, 11, 5) |
		bitnumIntl(state, 27, 6) | bitnumIntl(state, 16, 7) | bitnumIntl(state, 0, 8) |
		bitnumIntl(state, 14, 9) | bitnumIntl(state, 22, 10) | bitnumIntl(state, 25, 11) |
		bitnumIntl(state, 4, 12) | bitnumIntl(state, 17, 13) | bitnumIntl(state, 30, 14) |
		bitnumIntl(state, 9, 15) | bitnumIntl(state, 1, 16) | bitnumIntl(state, 7, 17) |
		bitnumIntl(state, 23, 18) | bitnumIntl(state, 13, 19) | bitnumIntl(state, 31, 20) |
		bitnumIntl(state, 26, 21) | bitnumIntl(state, 2, 22) | bitnumIntl(state, 8, 23) |
		bitnumIntl(state, 18, 24) | bitnumIntl(state, 12, 25) | bitnumIntl(state, 29, 26) |
		bitnumIntl(state, 5, 27) | bitnumIntl(state, 21, 28) | bitnumIntl(state, 10, 29) |
		bitnumIntl(state, 3, 30) | bitnumIntl(state, 24, 31))

	return state
}

// crypt DES加密/解密单个块
// key 包含16个48位子密钥，每个子密钥是6字节
func crypt(input []byte, key [16][]int) []byte {
	s0, s1 := initialPermutation(input)
	for idx := 0; idx < 15; idx++ {
		previousS1 := s1
		s1 = f(s1, key[idx]) ^ s0
		s0 = previousS1
	}
	s0 = f(s1, key[15]) ^ s0
	return inversePermutation(s0, s1)
}

// keySchedule 生成密钥调度表
// mode: 1=ENCRYPT, 0=DECRYPT
func keySchedule(key []byte, mode int) [16][]int {
	schedule := [16][]int{}
	for i := 0; i < 16; i++ {
		schedule[i] = make([]int, 6)
	}

	// 从原始密钥生成C和D
	var c, d int
	for i := 0; i < 28; i++ {
		c |= bitnum(key, keyPermC[i], 31-i)
	}
	for i := 0; i < 28; i++ {
		d |= bitnum(key, keyPermD[i], 31-i)
	}

	for i := 0; i < 16; i++ {
		// 循环左移
		shift := keyRndShift[i]
		c = ((c << shift) | (c >> (28 - shift))) & 0xfffffff0
		d = ((d << shift) | (d >> (28 - shift))) & 0xfffffff0

		// 确定生成的轮密钥索引
		toGen := i
		if mode == 0 { // DECRYPT - 反转顺序
			toGen = 15 - i
		}

		// 从C生成前24位
		for j := 0; j < 24; j++ {
			bitPos := keyCompression[j]
			schedule[toGen][j/8] |= ((c >> (31 - bitPos)) & 1) << (7 - (j % 8))
		}
		// 从D生成后24位
		for j := 24; j < 48; j++ {
			bitPos := keyCompression[j]
			schedule[toGen][j/8] |= ((d >> (31 - (bitPos - 27))) & 1) << (7 - ((j - 24) % 8))
		}
	}
	return schedule
}

// tripledesKeySetup 3DES密钥调度表生成
func tripledesKeySetup(key []byte, mode int) [3][16][]int {
	if mode == 1 { // ENCRYPT
		return [3][16][]int{
			keySchedule(key[0:8], 1),
			keySchedule(key[8:16], 0),
			keySchedule(key[16:24], 1),
		}
	}
	// DECRYPT
	return [3][16][]int{
		keySchedule(key[16:24], 0),
		keySchedule(key[8:16], 1),
		keySchedule(key[0:8], 0),
	}
}

// tripledesCrypt 3DES加密/解密单个块
func tripledesCrypt(data []byte, key [3][16][]int) []byte {
	result := data
	for i := 0; i < 3; i++ {
		result = crypt(result, key[i])
	}
	return result
}

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
	key := []byte(QRCKey)
	schedule := tripledesKeySetup(key, 0) // 0 = DECRYPT

	result := make([]byte, len(data))
	for i := 0; i < len(data); i += 8 {
		block := data[i : i+8]
		decrypted := tripledesCrypt(block, schedule)
		copy(result[i:], decrypted)
	}

	// 调试：打印解密后的前几个字节
	end := min(10, len(result))
	logger.Debugf("[QQMusic] 解密后数据长度: %d, 前10字节: %x", len(result), result[:end])

	// Step 3: zlib 解压
	uncompressed, err := zlibInflate(result)
	if err != nil {
		return "", fmt.Errorf("zlib解压失败: %v, 前10字节: %x", err, result[:min(10, len(result))])
	}

	// Step 4: 去除 UTF-8 BOM (如果有)
	uncompressed = bytes.TrimPrefix(uncompressed, []byte{0xEF, 0xBB, 0xBF})

	return string(uncompressed), nil
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

// songSearchResult 歌曲搜索结果。
type songSearchResult struct {
	SongID   int64  // 歌曲ID，用于获取歌词
	SongMid  string // 歌曲MID
	Duration int    // 歌曲时长（秒）
}

// Search 根据歌曲信息从 QQ音乐搜索歌词。
// 参数：
//   - ctx: 上下文，用于取消和超时控制
//   - title: 歌曲标题
//   - artist: 艺术家名称
//   - duration: 歌曲时长（秒，用于版本匹配）
//
// 返回：
//   - lyrics: LRC 格式的歌词文本
//   - err: 错误信息
func (s *QQMusicSource) Search(ctx context.Context, title, artist string, duration int) (string, error) {
	// 步骤1：搜索歌曲获取候选列表（包含时长信息）
	candidates := s.searchSongs(ctx, title, artist)
	if len(candidates) == 0 {
		return "", nil // 未找到歌曲
	}

	// 步骤2：根据时长选择最匹配的版本
	var result *songSearchResult
	if duration > 0 {
		result = selectBestMatch(candidates, duration)
	} else {
		// 保持向后兼容：如果没有提供时长，使用第一个结果
		result = candidates[0]
	}

	if result == nil {
		return "", nil // 未找到歌曲
	}

	logger.Infof("[QQMusic] 找到歌曲: songID=%d, songMid=%s, duration=%ds (目标: %ds)",
		result.SongID, result.SongMid, result.Duration, duration)

	// 步骤3：获取加密歌词（使用新的API）
	encrypted, err := s.getLyricsByID(ctx, result.SongID, title, artist, duration)
	if err != nil {
		return "", fmt.Errorf("获取歌词失败: %w", err)
	}
	if encrypted == "" {
		return "", nil // 未找到歌词
	}

	// 步骤4：解密 QRC 格式歌词
	lyricsText, err := DecryptQRC(encrypted)
	if err != nil {
		return "", fmt.Errorf("解密歌词失败: %w", err)
	}

	// 步骤5：将 QRC 格式转换为 LRC 格式
	lrcText := qrc2lrc(lyricsText)

	return lrcText, nil
}

// searchSongs 搜索歌曲并返回候选列表，包含时长信息用于版本匹配。
func (s *QQMusicSource) searchSongs(ctx context.Context, title, artist string) []*songSearchResult {
	query := artist + " " + title

	// 构造请求数据，使用 DoSearchForQQMusicLite 方法
	searchData := map[string]interface{}{
		"comm": map[string]interface{}{
			"ct":        11,
			"cv":        "1003006",
			"v":         "1003006",
			"os_ver":    "15",
			"phonetype": "24122RKC7C",
			"rom":       "Redmi/miro/miro:15/AE3A.240806.005/OS2.0.105.0.VOMCNXM:user/release-keys",
			"tmeAppID":  "qqmusiclight",
			"nettype":   "NETWORK_WIFI",
			"udid":      "0",
			"uid":       "0",
			"sid":       "",
			"userip":    "",
		},
		"request": map[string]interface{}{
			"method": "DoSearchForQQMusicLite",
			"module": "music.search.SearchCgiService",
			"param": map[string]interface{}{
				"search_id":    fmt.Sprintf("%d", rand.Int63()),
				"remoteplace":  "search.android.keyboard",
				"query":        query,
				"search_type":  0,
				"num_per_page": 10,
				"page_num":     1,
				"highlight":    0,
				"nqc_flag":     0,
				"page_id":      1,
				"grp":          1,
			},
		},
	}

	body, err := json.Marshal(searchData)
	if err != nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://u.y.qq.com/cgi-bin/musicu.fcg", bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "okhttp/3.14.9")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	logger.Debugf("[QQMusic] 搜索响应: %s", string(respBody))

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil
	}

	// 提取 request 数据
	reqData, ok := result["request"].(map[string]interface{})
	if !ok {
		logger.Warnf("[QQMusic] 响应结构无效: 缺少 request")
		return nil
	}

	data, ok := reqData["data"].(map[string]interface{})
	if !ok {
		logger.Warnf("[QQMusic] 响应结构无效: 缺少 data")
		return nil
	}

	bodyData, ok := data["body"].(map[string]interface{})
	if !ok {
		logger.Warnf("[QQMusic] 响应结构无效: 缺少 body")
		return nil
	}

	// 获取歌曲列表（新API使用 item_song）
	songs, ok := bodyData["item_song"].([]interface{})
	if !ok || len(songs) == 0 {
		return nil // 无搜索结果
	}

	var results []*songSearchResult
	// 限制处理前10个结果以提高效率
	maxResults := 10
	if len(songs) < maxResults {
		maxResults = len(songs)
	}

	for i := 0; i < maxResults; i++ {
		song, ok := songs[i].(map[string]interface{})
		if !ok {
			continue
		}

		// 提取 songID、songMid 和 duration（新API使用 id、mid 和 interval）
		var songID int64
		var songMid string
		var duration int // 歌曲时长（秒）

		if id, ok := song["id"]; ok {
			switch v := id.(type) {
			case float64:
				songID = int64(v)
			case int64:
				songID = v
			}
		}

		if mid, ok := song["mid"].(string); ok {
			songMid = mid
		}

		// 提取时长信息（interval字段，单位为秒）
		if interval, ok := song["interval"]; ok {
			switch v := interval.(type) {
			case float64:
				duration = int(v)
			case int64:
				duration = int(v)
			}
		}

		// 只添加有效的歌曲结果
		if songID != 0 && songMid != "" {
			results = append(results, &songSearchResult{
				SongID:   songID,
				SongMid:  songMid,
				Duration: duration,
			})
		}
	}

	return results
}

// getMapKeys 获取 map 的所有键（用于调试）。
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// selectBestMatch 根据目标时长选择最匹配的歌曲版本
// 如果目标时长 <= 0，返回第一个结果（保持原有行为）
// 否则选择时长最接近的版本（允许±2秒的误差范围）
func selectBestMatch(results []*songSearchResult, targetDuration int) *songSearchResult {
	if len(results) == 0 {
		return nil
	}

	// 如果目标时长无效，返回第一个结果
	if targetDuration <= 0 {
		return results[0]
	}

	// 查找时长最接近的结果
	var bestMatch *songSearchResult
	minDiff := int(^uint(0) >> 1) // 最大整数值

	for _, result := range results {
		// 计算时长差的绝对值
		diff := result.Duration - targetDuration
		if diff < 0 {
			diff = -diff
		}

		// 更新最佳匹配
		if diff < minDiff {
			minDiff = diff
			bestMatch = result

			// 如果完全匹配，直接返回
			if diff == 0 {
				break
			}
		}
	}

	// 如果在容忍范围内（±2秒），返回最佳匹配
	// 否则返回第一个结果（保持向后兼容）
	if minDiff <= 2 {
		return bestMatch
	}

	// 超出容忍范围，返回第一个结果
	return results[0]
}

// getLyricsByID 根据歌曲ID获取加密歌词（使用新API）。
func (s *QQMusicSource) getLyricsByID(ctx context.Context, songID int64, title, artist string, duration int) (string, error) {
	// 使用 GetPlayLyricInfo API 获取歌词
	requestData := map[string]interface{}{
		"comm": map[string]interface{}{
			"ct":        11,
			"cv":        "1003006",
			"v":         "1003006",
			"os_ver":    "15",
			"phonetype": "24122RKC7C",
			"rom":       "Redmi/miro/miro:15/AE3A.240806.005/OS2.0.105.0.VOMCNXM:user/release-keys",
			"tmeAppID":  "qqmusiclight",
			"nettype":   "NETWORK_WIFI",
			"udid":      "0",
			"uid":       "0",
			"sid":       "",
			"userip":    "",
		},
		"request": map[string]interface{}{
			"method": "GetPlayLyricInfo",
			"module": "music.musichallSong.PlayLyricInfo",
			"param": map[string]interface{}{
				"albumName":  base64.StdEncoding.EncodeToString([]byte("")),
				"crypt":      1,
				"ct":         19,
				"cv":         2111,
				"interval":   duration,
				"lrc_t":      0,
				"qrc":        1,
				"qrc_t":      0,
				"roma":       1,
				"roma_t":     0,
				"singerName": base64.StdEncoding.EncodeToString([]byte(artist)),
				"songID":     songID,
				"songName":   base64.StdEncoding.EncodeToString([]byte(title)),
				"trans":      1,
				"trans_t":    0,
				"type":       0,
			},
		},
	}

	body, err := json.Marshal(requestData)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://u.y.qq.com/cgi-bin/musicu.fcg", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "okhttp/3.14.9")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	logger.Debugf("[QQMusic] 歌词响应: %s", string(respBody))

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	// 提取歌词数据
	reqData, ok := result["request"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("歌词响应结构无效: 缺少 request")
	}

	data, ok := reqData["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("歌词响应结构无效: 缺少 data")
	}

	// 获取加密歌词（qrc 字段）
	lyric, ok := data["lyric"].(string)
	if !ok || lyric == "" {
		return "", nil // 未找到歌词
	}

	return lyric, nil
}

// qrc2lrc 将 QRC 格式歌词转换为逐字 LRC 格式。
// QRC 格式：XML 结构，歌词内容在 LyricContent 属性中，格式为 [行起始ms,行持续ms]字(偏移,持续)...
// 返回：逐字 LRC 格式，每个字都有时间戳，如 [00:00.000]晴[00:00.016]天
func qrc2lrc(qrcXML string) string {
	startMarker := `LyricContent="`
	endMarker := `"/>`

	startIdx := strings.Index(qrcXML, startMarker)
	if startIdx < 0 {
		return qrcXML
	}

	startIdx += len(startMarker)
	endIdx := strings.Index(qrcXML[startIdx:], endMarker)
	if endIdx < 0 {
		return qrcXML
	}

	content := qrcXML[startIdx : startIdx+endIdx]
	lines := strings.Split(content, "\n")

	lineRe := regexp.MustCompile(`^\[(\d+),(\d+)\](.*)$`)
	wordRe := regexp.MustCompile(`([^\(]*)\((\d+),(\d+)\)`)

	var result strings.Builder

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		lineMatch := lineRe.FindStringSubmatch(line)
		if lineMatch == nil {
			continue
		}

		wordMatches := wordRe.FindAllStringSubmatch(lineMatch[3], -1)
		if wordMatches == nil {
			continue
		}

		for i, wm := range wordMatches {
			wordContent := wm[1]
			wordOffset, _ := strconv.Atoi(wm[2])
			wordDuration, _ := strconv.Atoi(wm[3])
			result.WriteString(fmt.Sprintf("[%s]%s", msToTimeStr(wordOffset), wordContent))
			if i == len(wordMatches)-1 {
				result.WriteString(fmt.Sprintf("[%s]", msToTimeStr(wordOffset+wordDuration)))
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// msToTimeStr 将毫秒转换为 LRC 时间格式 [mm:ss.xxx]
func msToTimeStr(ms int) string {
	minutes := ms / 60000
	seconds := (ms % 60000) / 1000
	milliseconds := ms % 1000
	return fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, milliseconds)
}
