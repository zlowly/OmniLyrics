//go:build windows
// +build windows

package smtc

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/media/control"
)

// WinRT 是 SMTC 接口的 Windows WinRT 实现。
// 该实现通过 Windows Runtime API 获取系统媒体传输控制（SMTC）的实时数据。
// 仅在 Windows 平台上编译和使用。
type WinRT struct {
	// lastData 缓存上一次获取的媒体数据，用于可能的派生操作
	lastData SMTCData
}

// NewWinRT 创建一个新的 WinRT 实例。
// @return *WinRT 返回指向 WinRT 结构体的指针
func NewWinRT() *WinRT {
	return &WinRT{}
}

// GetData 通过 Windows Runtime API 获取当前媒体的播放数据。
// 该方法调用系统媒体传输控制 API，获取当前播放会话的详细信息。
// @return SMTCData 返回包含当前媒体信息的结构体；error 返回可能发生的错误
func (w *WinRT) GetData() (SMTCData, error) {
	// 初始化 COM 库，COINIT_APARTMENTTHREADED 表示每个线程有自己的 Apartment
	// COM 是 Windows 组件对象模型，用于与系统 API 交互
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	defer ole.CoUninitialize()

	// 请求获取系统媒体传输控制的会话管理器
	// 这是一个异步操作，需要等待完成后才能获取结果
	asyncOp, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
	if err != nil {
		return SMTCData{Status: "Error", HasSession: false, Title: err.Error()}, nil
	}

	// 等待异步操作完成（超时 100ms），Windows Runtime 异步调用需要显式等待
	time.Sleep(100 * time.Millisecond)

	// 获取异步操作的结果，获取会话管理器实例
	resultPtr, err := asyncOp.GetResults()
	if err != nil {
		return SMTCData{Status: "Error", HasSession: false, Title: err.Error()}, nil
	}

	// 将返回的指针转换为会话管理器类型
	mgrInstance := (*control.GlobalSystemMediaTransportControlsSessionManager)(unsafe.Pointer(resultPtr))

	// 获取当前活动的媒体会话，如果没有会话则返回空状态
	session, err := mgrInstance.GetCurrentSession()
	if err != nil || session == nil {
		return SMTCData{Status: "NoSession", HasSession: false}, nil
	}

	// 初始化默认数据，会话存在且正在播放
	data := SMTCData{
		HasSession: true,
		Status:     "Playing",
	}

	// 尝试获取媒体属性（标题、艺术家、专辑名）
	// 这些属性可能不可用，因此使用条件检查
	propsAsync, err := session.TryGetMediaPropertiesAsync()
	if err == nil && propsAsync != nil {
		time.Sleep(50 * time.Millisecond)
		mediaPropsPtr, _ := propsAsync.GetResults()
		if mediaPropsPtr != nil {
			// 转换为媒体属性类型并逐个获取属性值
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

	// 如果标题为空，使用源应用程序名称作为备选
	// 这确保始终有可显示的内容
	if data.Title == "" {
		if srcApp, err := session.GetSourceAppUserModelId(); err == nil && srcApp != "" {
			data.Title = srcApp
		}
	}

	// 获取时间线属性，包括播放位置和总时长
	timeline, err := session.GetTimelineProperties()
	if err == nil && timeline != nil {
		// Position 是以 100 纳秒���10^4 ticks）为单位，需要除以 10000 转换为毫秒
		if pos, err := timeline.GetPosition(); err == nil {
			data.PositionMs = pos.Duration / 10000
		}
		// 时长由结束时间减去开始时间计算得出，同样需要转换为毫秒
		if start, err := timeline.GetStartTime(); err == nil {
			if end, err := timeline.GetEndTime(); err == nil {
				data.DurationMs = (end.Duration - start.Duration) / 10000
			}
		}
	}

	// 获取播放信息，包括播放状态（播放/暂停/停止）
	playbackInfo, err := session.GetPlaybackInfo()
	if err == nil && playbackInfo != nil {
		status, _ := playbackInfo.GetPlaybackStatus()
		// 将 Windows Runtime 的枚举值映射为字符串状态
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

// init 在包初始化时输出提示信息，表示 WinRT 后端已就绪
func init() {
	fmt.Println("[SMTC] WinRT backend initialized (winrt-go)")
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