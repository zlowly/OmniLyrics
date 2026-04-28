# 故障排除指南

本文档汇总了 OmniLyrics 使用中的常见问题、技术限制和解决方案。

## 1. 歌词不显示

### 1.1 检查后端是否运行

```bash
# 检查进程
ps aux | grep omnilyrics

# 检查端口（默认 8081）
curl http://localhost:8081/health
```

预期返回：
```json
{"status": "OK"}
```

### 1.2 检查播放器支持

OmniLyrics 通过 Windows SMTC API 获取播放状态。以下播放器支持情况：

| 播放器 | SMTC 支持 | 进度获取 | 说明 |
|--------|-----------|----------|------|
| Windows Media Player | ✅ | ✅ | 原生支持 |
| Groove 音乐 | ✅ | ✅ | 原生支持 |
| QQ 音乐 | ✅ | ✅ | 原生支持 |
| 网易云音乐 | ❌ | ❌ | 不支持 SMTC，需安装 BetterNCM 插件 |
| 酷狗音乐 | ⚠️ | ⚠️ | 使用 UI Automation 抓取进度 |
| Spotify | ✅ | ✅ | 原生支持 |
| Foobar2000 | ✅ | ✅ | 需安装 SMTC 组件 |

### 1.3 检查浏览器控制台

1. 打开歌词页面 http://localhost:8081/
2. 按 F12 打开开发者工具
3. 查看 Console 标签页的错误信息

常见错误：
- `Failed to fetch` - 后端未运行或端口错误
- `Lyrics not found` - 未找到匹配歌词

### 1.4 检查日志

启动后端时添加调试日志：

```bash
./omnilyrics-bridge -l debug
```

查看关键日志：
```
[Lyrics] 获取歌词: title=..., artist=...
[Lyrics] 缓存命中 / 尝试从 xxx 搜索...
```

## 2. 歌词与歌曲时间对不上

### 2.1 清除缓存重新搜索

```bash
# 删除缓存目录
rm -rf Cache/*.lrc

# 或仅删除特定歌曲
rm "Cache/艺术家_歌曲名.lrc"
```

### 2.2 检查歌词源

某些歌词源的歌词可能存在时间偏移。可在设置页面调整歌词源优先级。

### 2.3 酷狗音乐进度问题

酷狗音乐不使用标准 SMTC 接口，OmniLyrics 通过 UI Automation 抓取进度条。

**已知限制**：
- 仅支持酷狗主窗口的进度条
- 如果酷狗界面布局改变可能失效
- 最小化到托盘时可能无法获取

启用调试查看详情：
```bash
./omnilyrics-bridge -l debug
```

查找日志：
```
[Kugou] findSlider success / readSlider hit
```

## 3. OBS 背景不透明

### 3.1 设置浏览器源 CSS

在 OBS 浏览器源设置中，在"自定义 CSS"中添加：

```css
body {
    background: transparent !important;
}
#app {
    background: transparent !important;
}
```

### 3.2 检查渲染器配置

确保 `Config/renderer.json` 中没有设置不透明的背景色。

## 4. 播放器兼容性问题

### 4.1 网易云音乐

网易云音乐未实现 Windows SMTC 接口，无法直接获取播放状态。

**解决方案**：安装 BetterNCM 插件，它提供了 SMTC 支持。

### 4.2 酷狗音乐

酷狗音乐实现了部分 SMTC 功能，但不提供准确的播放进度。

**解决方案**：OmniLyrics 内置 `KugouCatcher`，通过 UI Automation 直接读取进度条。

工作原理：
1. 查找窗口标题匹配"酷狗音乐"的主窗口
2. 查找名为"进度"的滑块控件
3. 读取滑块的当前值和最大值

**如果失效**：
- 检查酷狗是否正在播放
- 检查酷狗窗口是否打开（不能仅显示桌面歌词）
- 查看调试日志确认查找过程

### 4.3 QQ 音乐 QRC 解密

QQ 音乐返回的是加密的 QRC 格式歌词，后端使用 `decrypter.go` 进行解密。

如果解密失败：
- 检查 `decrypter.go` 中的解密逻辑
- 查看日志中的解密错误信息

## 5. 性能问题

### 5.1 帧率不稳定

- 检查 OBS 是否开启了浏览器源硬件加速
- 减少 `visibleLines`（模糊模式）或关闭复杂动画
- 使用性能更好的渲染模式（karaoke 最轻量）

### 5.2 内存泄漏

正常情况下不应有内存泄漏。如果怀疑：
1. 启用调试日志观察请求频率
2. 检查是否有大量缓存文件

```bash
# 查看缓存文件数量
ls -l Cache/ | wc -l
```

## 6. 网络连接问题

### 6.1 歌词搜索失败

检查网络连接和歌词源可用性：

```bash
# 测试 lrclib
curl "https://lrclib.net/api/search?q=test"

# 测试 QQ 音乐（需要正确的请求格式）
```

### 6.2 超时设置

如果网络较慢，可增加超时时间：

修改 `Config/lyrics.json`：
```json
{
    "timeout": 10000,
    "retry": 2
}
```

## 7. 配置问题

### 7.1 配置文件不生效

- 检查配置文件路径：默认在程序运行目录的 `Config/` 下
- 检查 JSON 格式是否正确（可用 JSON 校验工具）
- 歌词源配置支持热重载，渲染器配置需刷新页面

### 7.2 端口被占用

```bash
# 检查端口占用
lsof -i :8081

# 使用其他端口
./omnilyrics-bridge -p 8082
```

## 8. 已知技术限制

### 8.1 Windows 平台限制

- SMTC API 仅 Windows 10/11 支持
- 部分播放器未实现 SMTC（如网易云）
- 酷狗进度抓取依赖窗口 UI 结构，可能随版本失效

### 8.2 歌词源限制

| 源 | 限制 |
|----|------|
| lrclib | 公开 API，可能有速率限制 |
| QQ音乐 | 需要正确的请求签名，可能随版本变化 |
| 酷狗音乐 | 需要网页接口可用 |

### 8.3 前端限制

- OBS 浏览器源基于 Chromium，支持 ES6+
- GSAP 动画库已内嵌，无需网络加载
- 不支持旧版 OBS（需 Chromium 70+）

## 9. 调试技巧

### 9.1 启用全部调试日志

```bash
./omnilyrics-bridge -l debug
```

### 9.2 单独测试歌词源

创建测试文件：

```go
// tests/manual_test.go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/zlowly/OmniLyrics/lyrics/sources"
)

func main() {
    src := sources.NewLRCLibSource()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
go run tests/manual_test.go
```

### 9.3 查看当前播放状态

```bash
# 简化状态
curl http://localhost:8081/status | jq

# 完整 SMTC 数据
curl http://localhost:8081/smtc | jq
```

### 9.4 模拟播放（Mock 模式）

在 Linux 或非 Windows 环境测试：

```bash
./omnilyrics-bridge --mock -l debug
```

Mock 模式会模拟 4 分钟歌曲循环播放（240秒播放 + 5秒暂停）。

## 10. 常见问题快速参考

| 问题 | 快速检查 | 解决方案 |
|------|----------|----------|
| 歌词不显示 | `curl /health` | 检查后端运行、播放器支持 |
| 时间对不上 | 删除缓存 | `rm Cache/*.lrc` |
| OBS 不透明 | 检查 CSS | 添加 `background: transparent` |
| 网易云无效 | 检查 SMTC | 安装 BetterNCM |
| 酷狗进度错 | 查看日志 | 确保酷狗主窗口打开 |
| 端口被占 | `lsof -i :8081` | 更换端口 `-p 8082` |
| 搜索失败 | 检查网络 | 增加超时时间 |
