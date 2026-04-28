package lyrics

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RuleSource 定义规则中单个歌词源的配置。
type RuleSource struct {
	Name     string `json:"name"`     // 歌词源名称标识
	Enabled  bool   `json:"enabled"`  // 是否启用该源
	Priority int    `json:"priority"` // 优先级，数字越小优先级越高
}

// MatchRule 定义一条匹配规则。
type MatchRule struct {
	AppName string       `json:"appName"` // 匹配的App名称，空表示兜底规则
	Sources []RuleSource `json:"sources"` // 歌词源列表
}

// LyricsConfig 定义歌词源的整体配置。
type LyricsConfig struct {
	Timeout int         `json:"timeout"` // 单次搜索超时时间（毫秒）
	Retry   int         `json:"retry"`   // 重试次数
	Rules   []MatchRule `json:"rules"`   // 匹配规则列表
}

// defaultConfig 返回默认歌词配置。
func defaultConfig() *LyricsConfig {
	return &LyricsConfig{
		Timeout: 5000,
		Retry:   1,
		Rules: []MatchRule{
			{
				AppName: "",
				Sources: []RuleSource{
					{Name: "lrclib", Enabled: true, Priority: 1},
					{Name: "qqmusic", Enabled: true, Priority: 2},
					{Name: "kgmusic", Enabled: true, Priority: 3},
				},
			},
		},
	}
}

// LoadConfig 从配置文件加载歌词源配置。
// 如果配置文件不存在，返回默认配置。
// 参数：
//   - configDir: 配置目录路径
//
// 返回：
//   - *LyricsConfig: 歌词配置对象
//   - error: 错误信息（如果有）
func LoadConfig(configDir string) (*LyricsConfig, error) {
	configPath := filepath.Join(configDir, "lyrics.json")

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		var cfg LyricsConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	return defaultConfig(), nil
}

// SaveConfig 保存歌词源配置到配置文件。
// 参数：
//   - configDir: 配置目录路径
//   - cfg: 歌词配置对象
//
// 返回：
//   - error: 错误信息（如果有）
func SaveConfig(configDir string, cfg *LyricsConfig) error {
	configPath := filepath.Join(configDir, "lyrics.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
