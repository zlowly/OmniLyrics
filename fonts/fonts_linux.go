//go:build !windows
// +build !windows

package fonts

import (
	"os"
	"os/exec"
	"strings"
)

func getSystemFontsImpl() ([]FontInfo, error) {
	// 尝试使用 fc-list 获取字体
	// 格式: fc-list : family
	cmd := exec.Command("fc-list", ":", "family")

	output, err := cmd.Output()
	if err != nil {
		// fc-list 不可用时返回默认字体
		return getDefaultFonts(), nil
	}

	fonts := parseFcListOutput(string(output))
	if len(fonts) == 0 {
		return getDefaultFonts(), nil
	}
	return fonts, nil
}

func parseFcListOutput(output string) []FontInfo {
	lines := strings.Split(output, "\n")
	seen := make(map[string]bool)
	fonts := make([]FontInfo, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// fc-list 输出格式: "Font Name:style1,style2 /path/to/font"
		// 只取冒号前面的字体名称
		parts := strings.Split(line, ":")
		if len(parts) > 0 {
			fontNames := strings.Split(parts[0], ",")
			for _, name := range fontNames {
				name = strings.TrimSpace(name)
				if name == "" || seen[name] {
					continue
				}
				seen[name] = true
				fonts = append(fonts, FontInfo{Name: name, Family: name})
			}
		}
	}
	return fonts
}

func getDefaultFonts() []FontInfo {
	return []FontInfo{
		{Name: "Arial", Family: "Arial"},
		{Name: "Helvetica", Family: "Helvetica"},
		{Name: "Times New Roman", Family: "Times New Roman"},
		{Name: "Georgia", Family: "Georgia"},
		{Name: "Verdana", Family: "Verdana"},
		{Name: "DejaVu Sans", Family: "DejaVu Sans"},
		{Name: "Liberation Sans", Family: "Liberation Sans"},
		{Name: "Noto Sans", Family: "Noto Sans"},
		{Name: "Noto Sans CJK SC", Family: "Noto Sans CJK SC"},
		{Name: "WenQuanYi Micro Hei", Family: "WenQuanYi Micro Hei"},
		{Name: "Droid Sans Fallback", Family: "Droid Sans Fallback"},
	}
}

// isFileExecutable 检查文件是否可执行
func isFileExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}