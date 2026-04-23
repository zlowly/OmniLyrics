package lyrics

import (
	"compress/zlib"
	"crypto/des"
	"encoding/hex"
	"io"
	"bytes"
)

// QRCKey 是 QQ 音乐歌词解密的固定密钥。
const QRCKey = "!@#)(*$%123ZXC!@!@#)(NHL"

// DecryptQRC 解密 QRC 加密的歌词。
// QRC 格式：Hex 编码 → 3DES-EDE 解密 → zlib 解压 → UTF-8 文本。
// @param encryptedHex 十六进制编码的加密歌词
// @return 解密后的歌词文本，error 解密过程中的错误
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
// 3DES 使用两个密钥 K1 和 K2，解密流程为: D(K1) → E(K2) → D(K1)
// @param data 待解密数据
// @param key 24 字节密钥
// @return 解密后的数据
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

var errInvalidKeyLength = errInvalidKeyLengthErr{}

type errInvalidKeyLengthErr struct{}

func (e errInvalidKeyLengthErr) Error() string {
	return "invalid key length: must be 24 bytes"
}