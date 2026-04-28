package sources

import "context"

// LyricsSource 定义歌词源接口，所有歌词源实现需遵循此接口。
// 该接口提供统一的歌词搜索方法，便于扩展新的歌词源。
type LyricsSource interface {
	// Name 返回歌词源的名称标识。
	Name() string

	// Search 根据歌曲信息搜索歌词。
	// 参数：
	//   - ctx: 上下文，用于取消和超时控制
	//   - title: 歌曲标题
	//   - artist: 艺术家名称
	//   - duration: 歌曲时长（秒）
	// 返回：
	//   - lyrics: LRC 格式的歌词文本
	//   - err: 错误信息
	Search(ctx context.Context, title, artist string, duration int) (lyrics string, err error)
}
