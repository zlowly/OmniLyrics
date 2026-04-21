package fonts

// FontInfo 表示单个字体的信息。
type FontInfo struct {
	Name   string `json:"name"`   // 字体名称
	Family string `json:"family"` // 字体族（可选）
}

// GetSystemFonts 获取系统已安装的字体列表。
// Windows: 通过 PowerShell 命令获取字体
// Linux: 通过 fc-list 命令获取字体
// @return []FontInfo 字体信息数组
// @return error 获取失败时返回错误
func GetSystemFonts() ([]FontInfo, error) {
	return getSystemFontsImpl()
}