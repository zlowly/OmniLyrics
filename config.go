package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Level 定义日志级别，值越小级别越低。
// 使用 iota 自动递增：Debug=0, Info=1, Warn=2, Error=3
type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

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

// logLevel 当前日志级别，默认为 Info
var logLevel Level = Info

// initFlags 初始化命令行参数。
// 使用 pflag 替代标准库的 flag，支持更丰富的命令行参数风格。
func initFlags() {
	// 端口参数
	pflag.StringP("port", "p", "8081", "HTTP server port")
	// 日志参数
	pflag.StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")
	pflag.String("log-file", "", "Log file path (empty for stdout)")
	// 目录参数
	pflag.String("cache-dir", "", "Cache directory path (default: ./Cache)")
	pflag.String("config-dir", "", "Config directory path (default: ./Config)")
	// 配置文件参数
	pflag.StringP("config", "c", "", "Config file path (default: config.json in executable directory)")
	// SMTC 模式参数
	pflag.Bool("mock", false, "Force use mock SMTC backend")

	// 将 pflag 绑定到 viper
	viper.BindPFlags(pflag.CommandLine)

	// 解析命令行参数
	pflag.Parse()
}

// loadConfig 加载配置。
// 优先级（从高到低）：命令行参数 > 配置文件 > 默认值
// 配置文件默认命名为 config.json，放在基础目录（当前工作目录或可执行文件目录）。
func loadConfig() (*Config, error) {
	// 设置配置文件目录为基础目录（当前工作目录优先）
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

	// 如果命令行指定了配置文件路径，则额外读取该文件
	if configPath := viper.GetString("config"); configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	}

	// 解析配置到结构体
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 命令行参数优先级最高，如果被设置则覆盖配置文件的值
	if pflag.CommandLine.Changed("port") {
		cfg.Port = viper.GetString("port")
	}
	if pflag.CommandLine.Changed("log-level") {
		cfg.Log.Level = viper.GetString("log-level")
	}
	if pflag.CommandLine.Changed("log-file") {
		cfg.Log.File = viper.GetString("log-file")
	}
	if pflag.CommandLine.Changed("cache-dir") {
		cfg.CacheDir = viper.GetString("cache-dir")
	}
	if pflag.CommandLine.Changed("config-dir") {
		cfg.ConfigDir = viper.GetString("config-dir")
	}
	if pflag.CommandLine.Changed("mock") {
		cfg.Mock = viper.GetBool("mock")
	}

	// 处理相对路径：相对于当前工作目录
	// 如果配置值为空，使用默认值
	if cfg.CacheDir == "" {
		cfg.CacheDir = "Cache"
	}
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = "Config"
	}

	// 如果是相对路径，转换为绝对路径（相对于当前工作目录）
	if !filepath.IsAbs(cfg.CacheDir) {
		cfg.CacheDir = filepath.Join(getBaseDir(), cfg.CacheDir)
	}
	if !filepath.IsAbs(cfg.ConfigDir) {
		cfg.ConfigDir = filepath.Join(getBaseDir(), cfg.ConfigDir)
	}

	return &cfg, nil
}

// parseLogLevel 解析日志级别字符串。
// @param levelStr 日志级别字符串 (debug, info, warn, error)
// @return Level 对应的日志级别
// @return error 解析失败时返回错误
func parseLogLevel(levelStr string) (Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return Debug, nil
	case "info":
		return Info, nil
	case "warn", "warning":
		return Warn, nil
	case "error":
		return Error, nil
	default:
		return Info, fmt.Errorf("unknown log level: %s (supported: debug, info, warn, error)", levelStr)
	}
}

// shouldLog 检查给定级别是否应该被记录。
// @param level 要检查的级别
// @return bool 是否应该记录
func shouldLog(level Level) bool {
	return level >= logLevel
}

// logWithLevel 按级别记录日志。
// @param level 日志级别
// @param format 格式化字符串
// @param args 参数
func logWithLevel(level Level, format string, args ...interface{}) {
	if !shouldLog(level) {
		return
	}

	prefix := ""
	switch level {
	case Debug:
		prefix = "[Debug] "
	case Info:
		prefix = "[Info] "
	case Warn:
		prefix = "[Warn] "
	case Error:
		prefix = "[Error] "
	}

	log.Printf(prefix+format, args...)
}

// setupLogger 配置日志系统。
// 根据配置设置日志级别和输出目标。
func setupLogger(cfg *Config) error {
	// 解析日志级别
	level, err := parseLogLevel(cfg.Log.Level)
	if err != nil {
		return err
	}
	logLevel = level

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

	logWithLevel(Info, "Log level: %s", cfg.Log.Level)
	if cfg.Log.File != "" {
		logWithLevel(Info, "Log file: %s", cfg.Log.File)
	} else {
		logWithLevel(Info, "Log output: stdout")
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
			log.Printf("[Info] Using default log level (info), reason: %v", err)
		} else {
			log.Printf("[Info] Using default log settings, reason: %v", err)
		}
	}

	return nil
}

// GetPort 返回配置的端口。
// @return string 端口号
func GetPort() string {
	if config == nil {
		return "8081"
	}
	return config.Port
}

// GetCacheDir 返回配置的缓存目录。
// @return string 缓存目录路径
func GetCacheDir() string {
	if config == nil {
		return "Cache"
	}
	return config.CacheDir
}

// GetConfigDir 返回配置的配置目录。
// @return string 配置目录路径
func GetConfigDir() string {
	if config == nil {
		return "Config"
	}
	return config.ConfigDir
}

// GetMock 返回是否强制使用 Mock SMTC 后端。
// @return bool 是否强制使用 mock
func GetMock() bool {
	if config == nil {
		return false
	}
	return config.Mock
}

// GetLogLevel 返回日志级别字符串。
// @return string 日志级别 ("debug", "info", "warn", "error")
func GetLogLevel() string {
	if config == nil {
		return "info"
	}
	return config.Log.Level
}
