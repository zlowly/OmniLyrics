//go:build !windows
// +build !windows

package main

import (
	"fmt"

	"github.com/omnilyrics/bridge/smtc"
)

// NewSMTC 创建一个适用于非 Windows 平台的 SMTC 实例。
// 在非 Windows 平台使用 Mock 后端提供模拟的测试数据。
// @return smtc.SMTC 返回非 Windows 平台的 SMTC 接口实现
func NewSMTC() smtc.SMTC {
	fmt.Println("[SMTC] Using Mock backend (non-Windows)")
	return smtc.NewMock()
}