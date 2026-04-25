//go:build windows
// +build windows

package main

import (
	"fmt"

	"github.com/omnilyrics/bridge/smtc"
)

// NewSMTC 创建一个适用于 Windows 平台的 SMTC 实例。
// 使用 Hybrid 后端：优先使用原生 WinRT 获取媒体信息，
// 当检测到酷狗播放器时，使用 UI Automation 捕获获取精确进度。
// 如果配置了 --mock 或 config.json 中 mock=true，则使用 Mock 后端。
// @return smtc.SMTC 返回 Windows 平台的 SMTC 接口实现
func NewSMTC() smtc.SMTC {
	if GetMock() {
		fmt.Println("[SMTC] Using Mock backend (forced by config)")
		return smtc.NewMock()
	}
	fmt.Println("[SMTC] Using Hybrid backend (WinRT + Kugou Capture)")
	return smtc.NewHybrid()
}
