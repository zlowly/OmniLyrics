//go:build !windows
// +build !windows

package smtc

// NewSMTC 创建一个适用于非 Windows 平台的 SMTC 实例。
// 在非 Windows 平台使用 Mock 后端提供模拟的测试数据。
// 参数：
//   - mock: 是否强制使用 Mock 后端（非 Windows 平台始终为 true）
//
// 返回：
//   - SMTC: 返回非 Windows 平台的 SMTC 接口实现
func NewSMTC(mock bool) SMTC {
	_ = mock // 非 Windows 平台始终使用 Mock
	println("[SMTC] Using Mock backend (non-Windows)")
	return NewMock()
}
