//go:build windows
// +build windows

package smtc

import (
    "fmt"
    "sync"
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
    // ensureInit 保证 COM 和 SMTC 会话管理器初始化仅进行一次
    initOnce sync.Once
    // initErr 保存初始化过程中的错误
    initErr error
    // mgr 持久化的会话管理器引用，供后续 GetData 调用复用
    mgr *control.GlobalSystemMediaTransportControlsSessionManager
}

// NewWinRT 创建一个新的 WinRT 实例。
// @return *WinRT 返回指向 WinRT 结构体的指针
func NewWinRT() *WinRT {
    fmt.Println("[SMTC] WinRT backend initialized (winrt-go)")
    return &WinRT{}
}

// ensureInit 在第一次调用 GetData 时进行一次性初始化
// 包含 COM 初始化以及 SMTC 会话管理器的获取与缓存
func (w *WinRT) ensureInit() {
    w.initOnce.Do(func() {
        // 初始化 COM，使用单线程模型
        w.initErr = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
        // 如果初始化返回错误且非已初始化，无需继续
        if w.initErr != nil {
            return
        }
        // 异步获取全局系统媒体传输控制会话管理器
        asyncOp, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
        if err != nil {
            w.initErr = err
            return
        }
        // 等待异步结果，避免太长阻塞
        time.Sleep(100 * time.Millisecond)
        resultPtr, err := asyncOp.GetResults()
        if err != nil {
            w.initErr = err
            return
        }
        w.mgr = (*control.GlobalSystemMediaTransportControlsSessionManager)(unsafe.Pointer(resultPtr))
    })
}

// GetData 通过 Windows Runtime API 获取当前媒体的播放数据。
// 该方法调用系统媒体传输控制 API，获取当前播放会话的详细信息。
// @return SMTCData 返回包含当前媒体信息的结构体；error 返回可能发生的错误
func (w *WinRT) GetData() (SMTCData, error) {
    // 初始化并尽量复用初始化结果
    w.ensureInit()
    if w.initErr != nil {
        return SMTCData{Status: "Error", HasSession: false, Title: w.initErr.Error()}, nil
    }
    if w.mgr == nil {
        return SMTCData{Status: "NoSession", HasSession: false}, nil
    }

    // 获取当前活动的媒体会话
    session, err := w.mgr.GetCurrentSession()
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

    if appName, err := session.GetSourceAppUserModelId(); err == nil && appName != "" {
        data.AppName = appName
    }

    // 如果标题为空，使用默认占位符
    if data.Title == "" {
        data.Title = "Unknown"
    }

    // 获取时间线属性，包括播放位置和总时长
    timeline, err := session.GetTimelineProperties()
    if err == nil && timeline != nil {
        // Position 是以 100 纳秒（10^7 ticks）为单位，需要除以 10000 转换为毫秒
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
