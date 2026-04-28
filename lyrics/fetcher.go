package lyrics

import (
	"context"
	"fmt"
	"time"

	"github.com/zlowly/OmniLyrics/logger"
	"github.com/zlowly/OmniLyrics/lyrics/sources"
)

// Fetcher 是歌词获取器，负责协调缓存检查和多源搜索。
type Fetcher struct {
	cacheDir string        // 缓存目录路径
	config   *LyricsConfig // 歌词源配置
}

// NewFetcher 创建新的歌词获取器实例。
// 参数：
//   - cacheDir: 缓存目录路径
//   - cfg: 歌词源配置
//
// 返回：
//   - *Fetcher: 歌词获取器实例
func NewFetcher(cacheDir string, cfg *LyricsConfig) *Fetcher {
	return &Fetcher{
		cacheDir: cacheDir,
		config:   cfg,
	}
}

// sourceFactories 定义歌词源工厂函数映射。
var sourceFactories = map[string]func() sources.LyricsSource{
	"lrclib":  func() sources.LyricsSource { return sources.NewLRCLibSource() },
	"qqmusic": func() sources.LyricsSource { return sources.NewQQMusicSource() },
	"kgmusic": func() sources.LyricsSource { return sources.NewKGMusicSource() },
}

// FetchRequest 定义歌词获取请求。
type FetchRequest struct {
	Title    string // 歌曲标题
	Artist   string // 艺术家名称
	Duration int    // 歌曲时长（秒）
	AppName  string // 播放器名称（用于智能匹配源）
}

// FetchResult 定义歌词获取结果。
type FetchResult struct {
	Found   bool   // 是否找到歌词
	Lyrics  string // LRC 格式的歌词文本
	Source  string // 歌词来源名称
	Cached  bool   // 是否来自缓存
	Error   string // 错误信息（如果有）
}

// Fetch 获取歌词，先检查缓存，未命中则搜索。
// 参数：
//   - ctx: 上下文，用于取消和超时控制
//   - req: 歌词获取请求
//
// 返回：
//   - *FetchResult: 获取结果
func (f *Fetcher) Fetch(ctx context.Context, req *FetchRequest) *FetchResult {
	logger.Infof("[Lyrics] 获取歌词: title=%s, artist=%s, appName=%s",
		req.Title, req.Artist, req.AppName)

	// 步骤1：检查缓存
	if found, content, err := CheckCache(f.cacheDir, req.Title, req.Artist, req.Duration); err == nil && found {
		logger.Infof("[Lyrics] 缓存命中")
		return &FetchResult{
			Found:  true,
			Lyrics: content,
			Source: "cache",
			Cached: true,
		}
	}

	// 步骤2：根据 appName 过滤和排序歌词源
	sources := f.filterSourcesByApp(req.AppName)
	if len(sources) == 0 {
		logger.Warnf("[Lyrics] 没有可用的歌词源")
		return &FetchResult{
			Found: false,
			Error: "没有可用的歌词源",
		}
	}

	// 步骤3：按优先级搜索歌词
	for _, src := range sources {
		logger.Infof("[Lyrics] 尝试从 %s 搜索...", src.Name())

		// 为每个源设置超时
		srcCtx, cancel := context.WithTimeout(ctx, time.Duration(f.config.Timeout)*time.Millisecond)
		defer cancel()

		lyrics, err := src.Search(srcCtx, req.Title, req.Artist, req.Duration)
		if err != nil {
			logger.Warnf("[Lyrics] %s 搜索失败: %v", src.Name(), err)
			continue
		}

		if lyrics != "" {
			logger.Infof("[Lyrics] %s 找到歌词", src.Name())

			// 保存到缓存
			if err := UpdateCache(f.cacheDir, req.Title, req.Artist, req.Duration, lyrics); err != nil {
				logger.Warnf("[Lyrics] 缓存保存失败: %v", err)
			}

			return &FetchResult{
				Found:  true,
				Lyrics: lyrics,
				Source: src.Name(),
				Cached: false,
			}
		}
	}

	logger.Warnf("[Lyrics] 所有歌词源均未找到")
	return &FetchResult{
		Found: false,
		Error: "未找到歌词",
	}
}

// filterSourcesByApp 根据播放器名称过滤和排序歌词源。
// 根据匹配规则返回对应歌词源，并按优先级排序。
func (f *Fetcher) filterSourcesByApp(appName string) []sources.LyricsSource {
	type scoredSource struct {
		source sources.LyricsSource
		score  int // 分数越低优先级越高
	}

	// 查找匹配的规则：优先精确匹配 AppName，否则找兜底规则
	var matchedRule *MatchRule
	for i := range f.config.Rules {
		rule := &f.config.Rules[i]
		if rule.AppName == "" {
			// 兜底规则（AppName为空），只在使用其他规则都不匹配时采用
			if matchedRule == nil {
				matchedRule = rule
			}
		} else if appName != "" && rule.AppName == appName {
			// 精确匹配
			matchedRule = rule
			break
		}
	}

	if matchedRule == nil {
		logger.Infof("[Lyrics] 无匹配规则，使用默认配置")
		return nil
	}

	logger.Infof("[Lyrics] 使用匹配规则: appName=%s", matchedRule.AppName)

	// 收集启用的源并按优先级排序
	var scored []scoredSource
	for _, srcConfig := range matchedRule.Sources {
		if !srcConfig.Enabled {
			continue
		}
		if factory, ok := sourceFactories[srcConfig.Name]; ok {
			scored = append(scored, scoredSource{
				source: factory(),
				score:  srcConfig.Priority,
			})
			logger.Infof("[Lyrics] 启用歌词源: %s (优先级: %d)", srcConfig.Name, srcConfig.Priority)
		}
	}

	// 按分数排序（低分优先），相同分数保持原始顺序（并行）
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score < scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	result := make([]sources.LyricsSource, len(scored))
	for i, s := range scored {
		result[i] = s.source
	}

	return result
}

// 全局路径变量（用于每次请求重新加载配置）
var (
	globalCacheDir  string
	globalConfigDir string
)

// InitFetcher 初始化歌词获取器全局路径。
// 参数：
//   - cacheDir: 缓存目录路径
//   - configDir: 配置目录路径
//
// 返回：
//   - error: 错误信息（如果有）
func InitFetcher(cacheDir, configDir string) error {
	globalCacheDir = cacheDir
	globalConfigDir = configDir
	logger.Infof("[Lyrics] 歌词获取器配置完成（每次请求将重新加载配置）")
	return nil
}

// FetchLyrics 全局函数：获取歌词（便捷方法）。
// 每次调用都会重新加载配置，实现热重载。
// 参数：
//   - ctx: 上下文
//   - title: 歌曲标题
//   - artist: 艺术家名称
//   - duration: 时长（秒）
//   - appName: 播放器名称
//
// 返回：
//   - *FetchResult: 获取结果
func FetchLyrics(ctx context.Context, title, artist string, duration int, appName string) *FetchResult {
	if globalCacheDir == "" || globalConfigDir == "" {
		return &FetchResult{
			Found: false,
			Error: "歌词获取器未初始化",
		}
	}

	// 每次都重新加载配置，实现热重载
	cfg, err := LoadConfig(globalConfigDir)
	if err != nil {
		return &FetchResult{
			Found: false,
			Error: fmt.Sprintf("加载配置失败: %v", err),
		}
	}

	// 创建新的 Fetcher（使用最新配置）
	fetcher := NewFetcher(globalCacheDir, cfg)

	req := &FetchRequest{
		Title:    title,
		Artist:   artist,
		Duration: duration,
		AppName:  appName,
	}

	return fetcher.Fetch(ctx, req)
}
