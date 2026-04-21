//go:build windows
// +build windows

package fonts

import (
	"os/exec"
	"strings"
)

func getSystemFontsImpl() ([]FontInfo, error) {
	// PowerShell 命令获取系统字体
	cmd := exec.Command("powershell", "-Command", 
		"[System.Reflection.Assembly]::LoadWithPartialName('System.Drawing') | Out-Null; " +
		"(New-Object System.Drawing.Text.InstalledFontCollection).Families | " +
		"ForEach-Object { $_.Name }")

	output, err := cmd.Output()
	if err != nil {
		return getDefaultFonts(), err
	}

	fonts := parseFontList(string(output))
	if len(fonts) == 0 {
		return getDefaultFonts(), nil
	}
	return fonts, nil
}

func parseFontList(output string) []FontInfo {
	lines := strings.Split(output, "\n")
	fonts := make([]FontInfo, 0, len(lines))

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" || name == "Exception" || strings.Contains(name, "WARNING") {
			continue
		}
		fonts = append(fonts, FontInfo{Name: name, Family: name})
	}
	return fonts
}

func getDefaultFonts() []FontInfo {
	return []FontInfo{
		{Name: "Arial", Family: "Arial"},
		{Name: "Times New Roman", Family: "Times New Roman"},
		{Name: "Courier New", Family: "Courier New"},
		{Name: "Verdana", Family: "Verdana"},
		{Name: "Georgia", Family: "Georgia"},
		{Name: "Tahoma", Family: "Tahoma"},
		{Name: "Trebuchet MS", Family: "Trebuchet MS"},
		{Name: "Impact", Family: "Impact"},
		{Name: "Comic Sans MS", Family: "Comic Sans MS"},
		{Name: "Microsoft YaHei", Family: "Microsoft YaHei"},
		{Name: "SimSun", Family: "SimSun"},
		{Name: "SimHei", Family: "SimHei"},
	}
}