//go:build windows
// +build windows

package smtc

type Hybrid struct {
	winrt SMTC
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