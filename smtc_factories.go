//go:build windows
// +build windows

package main

import (
	"fmt"

	"github.com/omnilyrics/bridge/smtc"
)

// NewSMTC 创建一个适用于 Windows 平台的 SMTC 实例。
// 如果配置了 --mock 或 config.json 中 mock=true，则使用 Mock 后端；
// 否则使用 WinRT 后端获取真实的系统媒体传输控制数据。
// @return smtc.SMTC 返回 Windows 平台的 SMTC 接口实现
func NewSMTC() smtc.SMTC {
	if GetMock() {
		fmt.Println("[SMTC] Using Mock backend (forced by config)")
		return smtc.NewMock()
	}
	fmt.Println("[SMTC] Using WinRT backend")
	return smtc.NewWinRT()
}
