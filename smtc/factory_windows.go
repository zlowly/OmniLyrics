//go:build windows
// +build windows

package smtc

// NewSMTC 创建一个适用于 Windows 平台的 SMTC 实例。
// 使用 Hybrid 后端：优先使用原生 WinRT 获取媒体信息，
// 当检测到酷狗播放器时，使用 UI Automation 捕获获取精确进度。
// 参数：
//   - mock: 是否强制使用 Mock 后端
//
// 返回：
//   - SMTC: 返回 Windows 平台的 SMTC 接口实现
func NewSMTC(mock bool) SMTC {
	if mock {
		println("[SMTC] Using Mock backend (forced by config)")
		return NewMock()
	}
	println("[SMTC] Using Hybrid backend (WinRT + Kugou Capture)")
	return NewHybrid()
}
