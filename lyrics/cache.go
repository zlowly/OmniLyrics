package lyrics

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// sanitizeFilename 将文件名中的非法字符替换为下划线。
// Windows 文件系统不允许以下字符：\ / : * ? " < > |
// 参数：
//   - name: 原始文件名
//
// 返回：
//   - string: 清理后的安全文件名
func sanitizeFilename(name string) string {
	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = reg.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if name == "" {
		return "_empty_"
	}
	return name
}

// CheckCache 检查歌词缓存是否存在。
// 通过 title 和 artist 组合查找对应的 .lrc 文件。
// 参数：
//   - cacheDir: 缓存目录路径
//   - title: 歌曲标题
//   - artist: 艺术家名称
//
// 返回：
//   - found: 是否找到缓存
//   - content: 歌词内容（如果找到）
//   - err: 错误信息
func CheckCache(cacheDir, title, artist string) (found bool, content string, err error) {
	if title == "" && artist == "" {
		return false, "", nil
	}

	safeName := sanitizeFilename(artist + "_" + title)
	filePath := filepath.Join(cacheDir, safeName+".lrc")

	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[Lyrics] 读取缓存文件失败: %v", err)
			return false, "", err
		}
		return true, string(data), nil
	}

	return false, "", nil
}

// UpdateCache 更新歌词缓存。
// 将歌词内容写入以 "艺术家_标题.lrc" 命名的文件中。
// 参数：
//   - cacheDir: 缓存目录路径
//   - title: 歌曲标题
//   - artist: 艺术家名称
//   - lrc: LRC 格式的歌词内容
//
// 返回：
//   - error: 错误信息（如果有）
func UpdateCache(cacheDir, title, artist, lrc string) error {
	if title == "" || artist == "" || lrc == "" {
		return nil
	}

	safeName := sanitizeFilename(artist + "_" + title)
	filePath := filepath.Join(cacheDir, safeName+".lrc")

	if err := os.WriteFile(filePath, []byte(lrc), 0644); err != nil {
		log.Printf("[Lyrics] 写入缓存文件失败: %v", err)
		return err
	}

	log.Printf("[Lyrics] 缓存已更新: %s", filePath)
	return nil
}
