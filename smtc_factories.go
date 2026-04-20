package main

import (
	"fmt"
	"runtime"

	"github.com/omnilyrics/bridge/smtc"
)

func NewSMTC() smtc.SMTC {
	if runtime.GOOS == "windows" {
		fmt.Println("[SMTC] Using WinRT backend")
		return smtc.NewWinRT()
	}
	fmt.Println("[SMTC] Using Mock backend (non-Windows)")
	return smtc.NewMock()
}
