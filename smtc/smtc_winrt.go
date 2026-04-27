//go:build windows
// +build windows

package smtc

import (
    "log"
    "sync"
    "time"
    "unsafe"

    "github.com/go-ole/go-ole"
    "github.com/saltosystems/winrt-go/windows/media/control"
)

// WinRTWinRTDebugEnabled 控制 SMTC WinRT 调试日志的输出
// 设置为 true 时输出详细诊断信息，false 时仅输出关键步骤和错误
var WinRTDebugEnabled = false

// WinRT 是 SMTC 接口的 Windows WinRT 实现。
// 该实现通过 Windows Runtime API 获取系统媒体传输控制（SMTC）的实时数据。
// 仅在 Windows 平台上编译和使用。
type WinRT struct {
    // lastData 缓存上一次获取的媒体数据，用于可能的派生操作
    lastData SMTCData
    // initMutex 保护初始化状态，防止 sync.Once 失败后无法重试
    initMutex sync.Mutex
    // initOnce 保证 COM 和 SMTC 会话管理器初始化仅进行一次
    initOnce sync.Once
    // initErr 保存初始化过程中的错误
    initErr error
    // mgr 持久化的会话管理器引用，供后续 GetData 调用复用
    mgr *control.GlobalSystemMediaTransportControlsSessionManager
    // initAttempts 记录初始化尝试次数，用于诊断
    initAttempts int
    // lastInitTime 记录上次初始化时间
    lastInitTime time.Time
}

// NewWinRT 创建一个新的 WinRT 实例。
// @return *WinRT 返回指向 WinRT 结构体的指针
func NewWinRT() *WinRT {
    log.Println("[SMTC] WinRT backend initialized (winrt-go)")
    return &WinRT{}
}

// Reset 重置 WinRT 实例，允许重新初始化
// 用于初始化失败后重试
func (w *WinRT) Reset() {
    w.initMutex.Lock()
    defer w.initMutex.Unlock()

    if WinRTDebugEnabled {
        log.Printf("[SMTC WinRT] Reset requested, was initialized: mgr=%v, initErr=%v, attempts=%d", w.mgr != nil, w.initErr, w.initAttempts)
    }

    // 释放 SMTC 管理器
    if w.mgr != nil {
        w.mgr = nil
    }

    // 重置状态，允许重新初始化
    w.initOnce = sync.Once{}
    w.initErr = nil
}

// ensureInit 在第一次调用 GetData 时进行一次性初始化
// 包含 COM 初始化以及 SMTC 会话管理器的获取与缓存
func (w *WinRT) ensureInit() {
    w.initMutex.Lock()
    w.initAttempts++
    w.lastInitTime = time.Now()
    attempt := w.initAttempts
    w.initMutex.Unlock()

    if WinRTDebugEnabled {
        log.Printf("[SMTC WinRT] ensureInit called, attempt=%d, current state: mgr=%v, initErr=%v", attempt, w.mgr != nil, w.initErr)
    }

    w.initOnce.Do(func() {
        if WinRTDebugEnabled {
            log.Printf("[SMTC WinRT] Starting initialization (attempt %d)...", attempt)
        }

        w.initErr = ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
        if w.initErr != nil {
            errMsg := w.initErr.Error()
            if errMsg == "RPC_E_CHANGED_MODE" {
                if WinRTDebugEnabled {
                    log.Printf("[SMTC WinRT] COM already initialized (MULTITHREADED conflict), assuming already initialized")
                }
                w.initErr = nil
            } else {
                w.initErr = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
                if w.initErr != nil {
                    errMsg := w.initErr.Error()
                    if errMsg == "RPC_E_CHANGED_MODE" {
                        if WinRTDebugEnabled {
                            log.Printf("[SMTC WinRT] COM already initialized (APARTMENTTHREADED conflict), assuming already initialized")
                        }
                        w.initErr = nil
                    } else {
                        log.Printf("[SMTC WinRT] COM init failed: %v", w.initErr)
                        return
                    }
                }
            }
        }
        log.Println("[SMTC WinRT] COM initialized OK")

        asyncOp, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
        if err != nil {
            log.Printf("[SMTC WinRT] Request SMTC manager failed: %v", err)
            w.initErr = err
            return
        }
        if WinRTDebugEnabled {
            log.Println("[SMTC WinRT] SMTC manager request sent, waiting...")
        }

        time.Sleep(100 * time.Millisecond)
        resultPtr, err := asyncOp.GetResults()
        if err != nil {
            log.Printf("[SMTC WinRT] Get results failed: %v", err)
            w.initErr = err
            return
        }
        log.Println("[SMTC WinRT] SMTC manager initialized OK")
        w.mgr = (*control.GlobalSystemMediaTransportControlsSessionManager)(unsafe.Pointer(resultPtr))
    })
}

// GetData 通过 Windows Runtime API 获取当前媒体的播放数据。
// 该方法调用系统媒体传输控制 API，获取当前播放会话的详细信息。
// @return SMTCData 返回包含当前媒体信息的结构体；error 返回可能发生的错误
func (w *WinRT) GetData() (SMTCData, error) {
    // 初始化
    w.ensureInit()

    // 如果初始化失败，尝试重置并重试（最多一次）
    if w.initErr != nil {
        log.Printf("[SMTC WinRT] Initial attempt failed: %v, attempting reset...", w.initErr)

        w.Reset()
        w.ensureInit()

        // 如果重试后仍然失败，记录详细错误
        if w.initErr != nil {
            log.Printf("[SMTC WinRT] Retry also failed: %v", w.initErr)
            return SMTCData{Status: "Error", HasSession: false, Title: w.initErr.Error()}, nil
        }
        log.Println("[SMTC WinRT] Retry initialization succeeded")
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

    // 尝试获取媒体属性（标题、艺术家、专辑名），增加重试机制
    var mediaPropsPtr unsafe.Pointer
    propsAsync, err := session.TryGetMediaPropertiesAsync()
    if err == nil && propsAsync != nil {
        for retry := 0; retry < 3; retry++ {
            time.Sleep(100 * time.Millisecond)
            ptr, err := propsAsync.GetResults()
            if err == nil && ptr != nil {
                mediaPropsPtr = ptr
                break
            }
        }
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
