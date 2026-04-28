package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
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

// 包级变量：当前日志级别
var logLevel Level = Info

// SetLevel 设置当前日志级别。
func SetLevel(level Level) {
	logLevel = level
}

// SetLevelFromString 根据字符串设置日志级别。
func SetLevelFromString(levelStr string) error {
	switch strings.ToLower(levelStr) {
	case "debug":
		logLevel = Debug
	case "info":
		logLevel = Info
	case "warn", "warning":
		logLevel = Warn
	case "error":
		logLevel = Error
	default:
		return fmt.Errorf("unknown log level: %s (supported: debug, info, warn, error)", levelStr)
	}
	return nil
}

// shouldLog 检查给定级别是否应该被记录。
func shouldLog(level Level) bool {
	return level >= logLevel
}

// Debugf 记录 Debug 级别的日志。
func Debugf(format string, args ...interface{}) {
	if !shouldLog(Debug) {
		return
	}
	log.Printf("[Debug] "+format, args...)
}

// Infof 记录 Info 级别的日志。
func Infof(format string, args ...interface{}) {
	if !shouldLog(Info) {
		return
	}
	log.Printf("[Info] "+format, args...)
}

// Warnf 记录 Warn 级别的日志。
func Warnf(format string, args ...interface{}) {
	if !shouldLog(Warn) {
		return
	}
	log.Printf("[Warn] "+format, args...)
}

// Errorf 记录 Error 级别的日志。
func Errorf(format string, args ...interface{}) {
	if !shouldLog(Error) {
		return
	}
	log.Printf("[Error] "+format, args...)
}

// Fatalf 记录 Error 级别的日志并退出程序。
func Fatalf(format string, args ...interface{}) {
	log.Printf("[Fatal] "+format, args...)
	os.Exit(1)
}
