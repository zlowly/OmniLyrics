# 歌词源开发指南

本文档说明如何为 OmniLyrics 添加新的歌词源。

## 1. 歌词源接口

所有歌词源需实现 `lyrics/sources/interface.go` 中定义的接口：

```go
type LyricsSource interface {
    // Name 返回歌词源的名称标识
    Name() string

    // Search 根据歌曲信息查询歌词
    // 参数：
    //   - ctx: 上下文，用于取消和超时控制
    //   - title: 歌曲标题
    //   - artist: 艺术家名称
    //   - duration: 歌曲时长（秒）
    // 返回：
    //   - lyrics: LRC 格式的歌词文本（包含逐字时间戳更佳）
    //   - err: 错误信息
    Search(ctx context.Context, title, artist string, duration int) (lyrics string, err error)
}
```

## 2. 添加新歌词源步骤

### 2.1 创建歌词源文件

在 `lyrics/sources/` 目录下创建新文件，如 `mysource.go`：

```go
package sources

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "time"
)

// MysourceSource 是自定义歌词源实现
type MysourceSource struct {
    // 可添加配置字段
}

// NewMysourceSource 创建新的歌词源实例
func NewMysourceSource() *MysourceSource {
    return &MysourceSource{}
}

// Name 返回歌词源名称
func (s *MysourceSource) Name() string {
    return "mysource"
}

// Search 搜索歌词
func (s *MysourceSource) Search(ctx context.Context, title, artist string, duration int) (string, error) {
    // 1. 构建搜索请求
    searchURL := fmt.Sprintf("https://api.mysource.com/search?title=%s&artist=%s", title, artist)

    // 2. 创建 HTTP 请求（带 context 支持取消和超时）
    req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
    if err != nil {
        return "", fmt.Errorf("创建请求失败: %w", err)
    }

    // 3. 发送请求
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("请求失败: %w", err)
    }
    defer resp.Body.Close()

    // 4. 检查响应状态
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("API 返回错误状态码: %d", resp.StatusCode)
    }

    // 5. 解析响应，获取歌词 ID
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("读取响应失败: %w", err)
    }

    // 假设解析得到歌词 ID
    lyricID := parseLyricID(body)

    // 6. 获取歌词内容
    lyricURL := fmt.Sprintf("https://api.mysource.com/lyric/%s", lyricID)
    req, _ = http.NewRequestWithContext(ctx, "GET", lyricURL, nil)
    resp, err = client.Do(req)
    if err != nil {
        return "", fmt.Errorf("获取歌词失败: %w", err)
    }
    defer resp.Body.Close()

    lyricBody, _ := io.ReadAll(resp.Body)

    // 7. 转换为 LRC 格式
    lrc := convertToLRC(lyricBody)

    return lrc, nil
}

// parseLyricID 解析响应获取歌词 ID（需根据实际 API 实现）
func parseLyricID(body []byte) string {
    // 实现解析逻辑
    return ""
}

// convertToLRC 将 API 返回格式转换为 LRC 格式
func convertToLRC(data []byte) string {
    // 实现转换逻辑
    // 如果有逐字时间戳，格式为：[mm:ss.xx]字
    return ""
}
```

### 2.2 注册歌词源

在 `lyrics/fetcher.go` 的 `registerSources` 方法中注册新歌词源：

```go
func (f *Fetcher) registerSources() {
    sourceMap := map[string]func() sources.LyricsSource{
        "lrclib":  func() sources.LyricsSource { return sources.NewLRCLibSource() },
        "qqmusic": func() sources.LyricsSource { return sources.NewQQMusicSource() },
        "kgmusic": func() sources.LyricsSource { return sources.NewKGMusicSource() },
        "mysource": func() sources.LyricsSource { return sources.NewMysourceSource() }, // 新增
    }
    // ...
}
```

### 2.3 添加配置

用户可在 `Config/lyrics.json` 中配置新歌词源：

```json
{
    "timeout": 5000,
    "retry": 1,
    "sources": [
        {"name": "lrclib", "enabled": true, "priority": 1, "apps": ["*"]},
        {"name": "mysource", "enabled": true, "priority": 2, "apps": ["*"]}
    ]
}
```

## 3. LRC 格式规范

歌词源应返回标准 LRC 格式，支持逐字时间戳更佳。

### 3.1 基本 LRC 格式

```
[00:00.00]歌曲标题
[00:05.00]歌手名称
[00:10.00]第一句歌词
[00:15.50]第二句歌词
```

### 3.2 逐字时间戳格式

```
[00:10.00][00:10.00]第[00:10.10]一[00:10.20]句[00:10.30]歌[00:10.40]词
```

或（更紧凑的格式）：

```
[00:10.00][00:10.00][00:10.10]第[00:10.20]一[00:10.30]句[00:10.40]歌[00:10.50]词
```

前端 `motion.js` 的 `parseLRCInternal` 函数会解析这两种格式。

### 3.3 时间格式说明

| 格式 | 示例 | 说明 |
|------|------|------|
| `[mm:ss.xx]` | `[03:45.50]` | 分:秒.百分秒（2-3位毫秒） |
| `[mm:ss.xxx]` | `[03:45.500]` | 分:秒.毫秒（3位） |

## 4. 现有歌词源参考

### 4.1 lrclib 源 (sources/lrclib.go)

- **特点**：公开 API，无需认证，支持逐字时间戳
- **搜索接口**：`GET https://lrclib.net/api/search?q={query}`
- **歌词接口**：`GET https://lrclib.net/api/get/{id}`
- **返回格式**：JSON，包含 `syncedLyrics` 字段（LRC 格式）

### 4.2 QQ音乐源 (sources/qqmusic.go)

- **特点**：需解密 QRC 格式，支持逐字歌词
- **搜索接口**：需要构造请求获取歌曲 mid
- **歌词接口**：获取 QRC 格式歌词
- **解密**：使用 `decrypter.go` 中的解密函数

### 4.3 酷狗音乐源 (sources/kgmusic.go)

- **特点**：支持逐字歌词，需要解析特定格式
- **搜索**：通过关键词搜索
- **歌词获取**：获取 KRC 格式并转换

## 5. 错误处理

### 5.1 超时控制

使用传入的 `ctx` 参数支持超时和取消：

```go
func (s *MysourceSource) Search(ctx context.Context, title, artist string, duration int) (string, error) {
    // 创建带超时的上下文（如果调用方未设置）
    if _, ok := ctx.Deadline(); !ok {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
    }

    // 使用 ctx 创建请求
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    // ...
}
```

### 5.2 常见错误返回

| 场景 | 返回值 |
|------|--------|
| 未找到歌词 | `return "", nil` （歌词为空字符串，无错误） |
| 网络请求失败 | `return "", fmt.Errorf("请求失败: %w", err)` |
| API 返回错误 | `return "", fmt.Errorf("API 错误: %s", msg)` |
| 超时 | `return "", ctx.Err()` |

**注意**：未找到歌词时不要返回错误，只需返回空字符串。错误仅用于真正的异常情况。

## 6. 歌词源测试

### 6.1 编写测试

在 `lyrics/sources/` 目录下创建 `mysource_test.go`：

```go
package sources

import (
    "context"
    "testing"
    "time"
)

func TestMysourceSource_Search(t *testing.T) {
    source := NewMysourceSource()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    lyrics, err := source.Search(ctx, "歌曲标题", "艺术家", 180)
    if err != nil {
        t.Fatalf("搜索失败: %v", err)
    }

    if lyrics == "" {
        t.Log("未找到歌词（正常）")
    } else {
        t.Logf("找到歌词，长度: %d", len(lyrics))
    }
}
```

### 6.2 运行测试

```bash
# 测试单个源
go test -v ./lyrics/sources -run TestMysourceSource

# 测试所有歌词源
go test -v ./lyrics/...
```

## 7. 前端歌词源扩展（可选）

如果需要同时扩展前端歌词源（某些场景前端直接获取歌词），参考 `web/scripts/lyrics/providers/` 下的实现。

### 7.1 前端歌词源接口

```javascript
class BaseProvider {
    constructor() {
        this.name = 'base';
    }

    // 搜索歌词，返回 Promise<LRC 字符串>
    async search(title, artist, duration) {
        throw new Error('Not implemented');
    }
}
```

### 7.2 添加前端歌词源

1. 在 `web/scripts/lyrics/providers/` 创建新文件
2. 继承 `BaseProvider`
3. 在 `web/scripts/lyrics/index.js` 中注册

## 8. 调试技巧

### 8.1 启用调试日志

启动后端时设置日志级别为 debug：

```bash
./omnilyrics-bridge -l debug
```

### 8.2 查看歌词搜索日志

```
[Lyrics] 获取歌词: title=歌曲, artist=艺术家, appName=QQMusic.exe
[Lyrics] 注册歌词源: lrclib (优先级: 1)
[Lyrics] 尝试从 lrclib 搜索...
[Lyrics] lrclib 找到歌词
```

### 8.3 手动测试歌词源

创建测试文件 `tests/lyrics_mysource.go`：

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/zlowly/OmniLyrics/lyrics/sources"
)

func main() {
    src := sources.NewMysourceSource()
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    lyrics, err := src.Search(ctx, "告白气球", "周杰伦", 210)
    if err != nil {
        fmt.Printf("错误: %v\n", err)
        return
    }

    fmt.Println(lyrics)
}
```

运行：
```bash
go run tests/lyrics_mysource.go
```

## 9. 注意事项

1. **遵循接口**：确保实现 `LyricsSource` 接口的所有方法
2. **超时处理**：使用 `ctx` 参数，支持调用方的超时控制
3. **LRC 格式**：返回的歌词应是标准 LRC 格式
4. **逐字时间戳**：如果源支持，尽量返回带逐字时间戳的歌词
5. **错误语义**：未找到歌词返回空字符串而非错误
6. **配置热重载**：修改 `lyrics.json` 后无需重启即可生效
