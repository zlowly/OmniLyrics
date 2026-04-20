package smtc

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/media/control"
)

type WinRT struct {
	lastData SMTCData
}

func NewWinRT() *WinRT {
	return &WinRT{}
}

func (w *WinRT) GetData() (SMTCData, error) {
	if runtime.GOOS != "windows" {
		return SMTCData{Status: "Unavailable", HasSession: false}, nil
	}

	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	defer ole.CoUninitialize()

	asyncOp, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
	if err != nil {
		return SMTCData{Status: "Error", HasSession: false, Title: err.Error()}, nil
	}

	time.Sleep(100 * time.Millisecond)

	resultPtr, err := asyncOp.GetResults()
	if err != nil {
		return SMTCData{Status: "Error", HasSession: false, Title: err.Error()}, nil
	}

	mgrInstance := (*control.GlobalSystemMediaTransportControlsSessionManager)(unsafe.Pointer(resultPtr))
	session, err := mgrInstance.GetCurrentSession()
	if err != nil || session == nil {
		return SMTCData{Status: "NoSession", HasSession: false}, nil
	}

	data := SMTCData{
		HasSession: true,
		Status:     "Playing",
	}

	propsAsync, err := session.TryGetMediaPropertiesAsync()
	if err == nil && propsAsync != nil {
		time.Sleep(50 * time.Millisecond)
		mediaPropsPtr, _ := propsAsync.GetResults()
		if mediaPropsPtr != nil {
			mediaProps := (*control.GlobalSystemMediaTransportControlsSessionMediaProperties)(unsafe.Pointer(mediaPropsPtr))
			if title, err := mediaProps.GetTitle(); err == nil && title != "" {
				data.Title = title
			}
			if artist, err := mediaProps.GetArtist(); err == nil && artist != "" {
				data.Artist = artist
			}
			if album, err := mediaProps.GetAlbumTitle(); err == nil && album != "" {
				data.AlbumTitle = album
			}
		}
	}

	if data.Title == "" {
		if srcApp, err := session.GetSourceAppUserModelId(); err == nil && srcApp != "" {
			data.Title = srcApp
		}
	}

	timeline, err := session.GetTimelineProperties()
	if err == nil && timeline != nil {
		if pos, err := timeline.GetPosition(); err == nil {
			data.PositionMs = pos.Duration / 10000
		}
		if start, err := timeline.GetStartTime(); err == nil {
			if end, err := timeline.GetEndTime(); err == nil {
				data.DurationMs = (end.Duration - start.Duration) / 10000
			}
		}
	}

	playbackInfo, err := session.GetPlaybackInfo()
	if err == nil && playbackInfo != nil {
		status, _ := playbackInfo.GetPlaybackStatus()
		switch status {
		case control.GlobalSystemMediaTransportControlsSessionPlaybackStatus(1):
			data.Status = "Playing"
		case control.GlobalSystemMediaTransportControlsSessionPlaybackStatus(2):
			data.Status = "Paused"
		case control.GlobalSystemMediaTransportControlsSessionPlaybackStatus(0):
			data.Status = "Stopped"
		case control.GlobalSystemMediaTransportControlsSessionPlaybackStatus(3):
			data.Status = "Stopped"
		}
	}

	w.lastData = data
	return data, nil
}

func init() {
	fmt.Println("[SMTC] WinRT backend initialized (winrt-go)")
}