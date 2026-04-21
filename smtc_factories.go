//go:build windows
// +build windows

package main

import (
	"fmt"

	"github.com/omnilyrics/bridge/smtc"
)

// NewSMTC 创建一个适用于 Windows 平台的 SMTC 实例。
// 在 Windows 平台使用 WinRT 后端获取真实的系统媒体传输控制数据。
// @return smtc.SMTC 返回 Windows 平台的 SMTC 接口实现
func NewSMTC() smtc.SMTC {
	fmt.Println("[SMTC] Using WinRT backend")
	return smtc.NewWinRT()
}
