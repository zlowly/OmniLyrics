package smtc

// SMTCData 表示系统媒体传输控制（SMTC）的当前状态数据。
// 该结构体包含当前播放媒体的所有相关信息，用于在应用程序中显示或处理。
type SMTCData struct {
	// Status 表示播放状态，可能的值包括 "Playing"、"Paused"、"Stopped"、"NoSession"、"Error"、"Unavailable"
	Status string `json:"status"`
	// Title 表示当前播放媒体（歌曲/视频）的标题
	Title string `json:"title"`
	// Artist 表示当前媒体的艺术家或创作者
	Artist string `json:"artist"`
	// AlbumTitle 表示专辑标题，如果不可用则忽略
	AlbumTitle string `json:"albumTitle,omitempty"`
	// PositionMs 表示当前播放位置，以毫秒为单位
	PositionMs int64 `json:"positionMs"`
	// DurationMs 表示媒体总时长，以毫秒为单位，如果不可用则忽略
	DurationMs int64 `json:"durationMs,omitempty"`
	// HasSession 表示是否存在活动的媒体会话
	HasSession bool `json:"hasSession"`
	// AppName 表示当前媒体源应用的名称（App User Model ID）
	AppName string `json:"appName,omitempty"`
}

// SMTC 是获取系统媒体传输控制数据的接口。
// 该接口定义了获取当前媒体播放状态的标准方法，允许使用不同的后端实现（WinRT、Mock等）。
type SMTC interface {
	// GetData 获取当前的媒体播放数据。
	// @return SMTCData 返回包含当前媒体信息的结构体；error 返回可能发生的错误
	GetData() (SMTCData, error)
	// Reset 重置后端，允许重新初始化。某些后端实现可能不需要此方法。
	Reset()
	// SetWinRTDebug 设置 WinRT 调试模式
	SetWinRTDebug(enabled bool)
	// SetKugouCatcherDebug 设置酷狗抓取器调试模式
	SetKugouCatcherDebug(enabled bool)
}
