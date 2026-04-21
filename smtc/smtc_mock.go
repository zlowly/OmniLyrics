package smtc

import (
	"fmt"
	"time"
)

// 常量定义模拟播放的时间和循环参数
const (
	// SongDurationMs 表示模拟歌曲的时长，3:48 分钟
	SongDurationMs = (3*60 + 48) * 1000
	// PauseDurationMs 表示歌曲结束后的暂停时长，5 秒 = 5000 毫秒
	PauseDurationMs = 5 * 1000
	// CycleDurationMs 是一个完整的播放周期，包含歌曲播放和暂停时间
	CycleDurationMs = SongDurationMs + PauseDurationMs
)

// startTime 记录程序启动时的时刻，用于计算相对时间
var startTime = time.Now()

// Mock 是 SMTC 接口的模拟实现，用于非 Windows 平台或开发测试。
// 通过计算当前时间与启动时间的差值来模拟播放过程，无需内部状态管理。
type Mock struct{}

// NewMock 创建新的 Mock 实例。
// @return *Mock 返回 Mock 指针
func NewMock() *Mock {
	fmt.Println("[SMTC] Mock backend initialized")
	return &Mock{}
}

// GetData 返回模拟的媒体播放数据。
// 播放过程：播放 4 分钟 -> 暂停 5 秒 -> 循环重新播放。
// 使用无状态设计，通过 time.Since(startTime) 计算当前位置。
// @return SMTCData 包含模拟的媒体信息；error 返回 nil
func (m *Mock) GetData() (SMTCData, error) {
	// 计算程序启动后经过的毫秒数
	elapsedMs := time.Since(startTime).Milliseconds()
	// 取模得到当前在周期中的位置
	cyclePosition := elapsedMs % CycleDurationMs

	// 如果位置超出歌曲时长，表示处于暂停期间
	if cyclePosition >= SongDurationMs {
		return SMTCData{
			Status:     "Stopped",
			Title:      "Demo Song",
			Artist:     "Demo Artist",
			AlbumTitle: "Demo Album",
			PositionMs: SongDurationMs,
			DurationMs: SongDurationMs,
			HasSession: true,
		}, nil
	}

	// 正常播放期间
	return SMTCData{
		Status:     "Playing",
		Title:      "Demo Song",
		Artist:     "Demo Artist",
		AlbumTitle: "Demo Album",
		PositionMs: cyclePosition,
		DurationMs: SongDurationMs,
		HasSession: true,
	}, nil
}
