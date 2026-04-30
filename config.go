package main

import (
	"encoding/json"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/zlowly/OmniLyrics/logger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// 嵌入 config_default.json 到二进制文件中
//go:embed config_default.json
var defaultConfigFS embed.FS

// Config 定义应用程序的配置结构。
type Config struct {
	Port      string    `mapstructure:"port"`
	Log       LogConfig `mapstructure:"log"`
	CacheDir  string    `mapstructure:"cache-dir"`
	ConfigDir string    `mapstructure:"config-dir"`
	Mock      bool      `mapstructure:"mock"`
}

// LogConfig 定义日志相关的配置。
type LogConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// config 全局配置实例
var config *Config

// initFlags 初始化命令行参数。
// 使用 pflag 替代标准库的 flag，支持更丰富的命令行参数风格。
// 注意：默认值已通过 viper.SetDefault() 从 config_default.json 注入。
func initFlags() {
	// 定义命令行参数（无默认值，由 viper 从嵌入的 config_default.json 提供）
	pflag.StringP("port", "p", "", "HTTP server port")
	pflag.StringP("log-level", "l", "", "Log level (debug, info, warn, error)")
	pflag.String("log-file", "", "Log file path (empty for stdout)")
	pflag.String("cache-dir", "", "Cache directory path")
	pflag.String("config-dir", "", "Config directory path")
	pflag.StringP("config", "c", "", "Config file path")
	pflag.Bool("mock", false, "Force use mock SMTC backend")

	// 将 pflag 绑定到 viper
	viper.BindPFlags(pflag.CommandLine)

	// 解析命令行参数
	pflag.Parse()
}

// loadConfig 加载配置。
// 优先级（从高到低）：命令行参数 > -c 指定的配置文件 > config.json > config_default.json（嵌入的默认值）
// 配置文件默认命名为 config.json，放在基础目录（当前工作目录或可执行文件目录）。
func loadConfig() (*Config, error) {
	// 第一步：加载嵌入的 config_default.json 作为基础默认值
	defaultData, err := defaultConfigFS.ReadFile("config_default.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded default config: %w", err)
	}

	// 解析默认值 JSON
	var defaultValues map[string]interface{}
	if err := json.Unmarshal(defaultData, &defaultValues); err != nil {
		return nil, fmt.Errorf("failed to parse default config: %w", err)
	}

	// 将嵌入的默认值注入到 viper（这会成为最低优先级）
	for key, value := range defaultValues {
		viper.SetDefault(key, value)
	}

	// 第二步：设置配置文件目录为基础目录（当前工作目录优先）
	baseDir := getBaseDir()
	viper.AddConfigPath(baseDir)

	// 设置配置文件名（不含扩展名，viper 会自动识别 .json）
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	// 尝试读取配置文件（如果不存在则使用默认值）
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在不是错误，使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("[Warn] Failed to read config file: %v", err)
		}
	}

	// 第三步：如果命令行指定了配置文件路径，则额外读取该文件
	if configPath := viper.GetString("config"); configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	}

	// 第四步：解析配置到结构体
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 第五步：处理命令行参数对嵌套配置的覆盖
	// viper 无法自动将 "log-level" 映射到 "log.level"，需要手动处理
	if pflag.CommandLine.Changed("log-level") {
		cfg.Log.Level = viper.GetString("log-level")
	}
	if pflag.CommandLine.Changed("log-file") {
		cfg.Log.File = viper.GetString("log-file")
	}

	// 第六步：处理相对路径：相对于当前工作目录
	if !filepath.IsAbs(cfg.CacheDir) {
		cfg.CacheDir = filepath.Join(getBaseDir(), cfg.CacheDir)
	}
	if !filepath.IsAbs(cfg.ConfigDir) {
		cfg.ConfigDir = filepath.Join(getBaseDir(), cfg.ConfigDir)
	}

	return &cfg, nil
}

// setupLogger 配置日志系统。
// 根据配置设置日志级别和输出目标。
func setupLogger(cfg *Config) error {
	// 设置日志级别
	if err := logger.SetLevelFromString(cfg.Log.Level); err != nil {
		return err
	}

	// 设置日志输出目标
	var writer io.Writer
	if cfg.Log.File != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(cfg.Log.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// 打开日志文件（追加模式）
		f, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		writer = f
	} else {
		// 使用标准输出
		writer = os.Stdout
	}

	// 设置全局日志输出
	log.SetOutput(writer)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	logger.Infof("Log level: %s", cfg.Log.Level)
	if cfg.Log.File != "" {
		logger.Infof("Log file: %s", cfg.Log.File)
	} else {
		logger.Infof("Log output: stdout")
	}

	return nil
}

// initConfig 初始化配置系统，在 main 函数开始时调用。
func initConfig() error {
	// 先初始化命令行参数
	initFlags()

	// 加载配置
	var err error
	config, err = loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 配置日志系统
	if err := setupLogger(config); err != nil {
		// 日志配置失败时使用标准输出，但使用 info 级别提示（不是错误，仅提示）
		log.SetOutput(os.Stdout)
		log.SetFlags(log.Ldate | log.Ltime)
		// 检查是否是空级别导致的错误（即使用默认值的情况）
		if config.Log.Level == "" {
			logger.Infof("Using default log level (info), reason: %v", err)
		} else {
			logger.Infof("Using default log settings, reason: %v", err)
		}
	}

	return nil
}

// GetPort 返回配置的端口。
// @return string 端口号
func GetPort() string {
	return config.Port
}

// GetCacheDir 返回配置的缓存目录。
// @return string 缓存目录路径
func GetCacheDir() string {
	return config.CacheDir
}

// GetConfigDir 返回配置的配置目录。
// @return string 配置目录路径
func GetConfigDir() string {
	return config.ConfigDir
}

// GetMock 返回是否强制使用 Mock SMTC 后端。
// @return bool 是否强制使用 mock
func GetMock() bool {
	return config.Mock
}

// GetLogLevel 返回日志级别字符串。
// @return string 日志级别 ("debug", "info", "warn", "error")
func GetLogLevel() string {
	return config.Log.Level
}
