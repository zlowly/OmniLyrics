//go:build windows
// +build windows

package smtc

import "log"

type Hybrid struct {
	winrt *WinRT
	kugou *KugouCatcher
}

func NewHybrid() *Hybrid {
	return &Hybrid{
		winrt: NewWinRT(),
		kugou: NewKugouCatcher(),
	}
}

func (h *Hybrid) GetData() (SMTCData, error) {
	data, err := h.winrt.GetData()
	if err != nil {
		log.Printf("[SMTC Hybrid] WinRT GetData failed: %v", err)
		return data, err
	}

	if data.AppName != "kugou" || !data.HasSession {
		return data, nil
	}

	posMs, durMs, err := h.kugou.GetPosition()
	if err != nil {
		return data, nil
	}

	data.PositionMs = posMs * 10
	data.DurationMs = durMs * 10
	return data, nil
}

// Reset 重置 Hybrid 后端，包括 WinRT 和 KugouCatcher
func (h *Hybrid) Reset() {
	log.Println("[SMTC Hybrid] Reset requested")
	h.winrt.Reset()
	h.kugou.Release()
}

// SetWinRTDebug 设置 WinRT 调试模式
func (h *Hybrid) SetWinRTDebug(enabled bool) {
	WinRTDebugEnabled = enabled
}

// SetKugouCatcherDebug 设置酷狗抓取器调试模式
func (h *Hybrid) SetKugouCatcherDebug(enabled bool) {
	KugouCatcherDebugEnabled = enabled
}