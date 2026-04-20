# OmniLyrics

桌面歌词引擎 (Desktop Lyrics Engine) - 为 OBS 直播场景优化的桌面歌词显示工具。

## 功能特性

- **系统媒体同步**：自动监听 Windows 系统媒体播放状态，实时同步歌曲播放进度
- **多源歌词获取**：优先使用本地缓存歌词，无缓存时自动从网络 (lrclib.net) 搜索匹配
- **多种展示模式**：
  - 单行卡拉OK - 逐字高亮发光效果
  - 双行滚动 - 当前行在上、下一行在下的交替滚动
  - 多行渐变模糊 - 一屏多行，亮度/模糊/尺寸渐变效果
- **OBS 优化**：透明背景、支持多层叠加
- **可配置**：支持自定义颜色、字体、动画参数

## 环境要求

- Windows 10/11
- golang
- 支持系统媒体传输控制 (SMTC) 的播放器（如 Spotify, Apple Music, QQ音乐等）

## 快速开始

### 1. 运行

```
.\bridge.exe
```

### 2. 打开歌词页面

浏览器访问：http://localhost:8080/

### 3. OBS 配置

1. 添加浏览器源
2. URL 设置为 `http://localhost:8080/`
3. 宽度：1920，高度：1080（根据需要调整）
4. 勾选"控制 audio"选项（根据需要）
5. 自定义 CSS 中设置透明：

```css
body {
    background: transparent !important;
}
```

## 配置

访问设置页面：http://localhost:8080/settings.html

### 通用设置

- 字体颜色
- 字体大小
- 字体
- 背景色
- 背景透明

### 模式参数

根据选择的展示模式，显示对应的参数配置。

## 文件结构

```
OmniLyrics/
├── bridge.exe          # Go 后端服务 (可执行文件)
├── Bridge.ps1         # PowerShell 后端 (旧版本保留)
├── main.go           # Go 后端源码
├── handlers.go       # HTTP 处理器
├── go.mod           # Go 模块定义
├── smtc/           # SMTC 实现
│   ├── smtc.go         # 接口定义
│   ├── smtc_winrt.go   # Windows WinRT 实现
│   └── smtc_mock.go    # 跨平台 Mock
├── web/             # 前端资源
│   ├── index.html
│   ├── settings.html
│   └── scripts/
├── Cache/            # 歌词缓存
└── Config/          # 配置存储
```

## API 接口

### 后端接口

| 接口 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/status` | GET | 获取播放状态 (简单) |
| `/smtc` | GET | 获取播放状态 (完整，含专辑) |
| `/check_cache` | GET | 查询歌词缓存 |
| `/update_cache` | POST | 更新歌词缓存 |
| `/config` | GET/POST | 配置管理 |
| `/shutdown` | GET/POST | 关闭服务 |

### 数据格式

**/status 返回**：
```json
{
    "title": "歌曲名",
    "artist": "艺术家",
    "status": "Playing|Stopped",
    "position": 51260,
    "duration": 211000
}
```

## 技术栈

- 前端：原生 JavaScript + GSAP 动画
- 后端：Go + winrt-go (Windows SMTC)
- 架构：心跳轮询 + 多源竞态调度

## 常见问题

### Q: 歌词不显示
A:
1. 检查后端是否正常运行
2. 检查播放器是否支持 SMTC
3. 检查浏览器 Console 日志

### Q: 歌词与歌曲时间对不上
A: 检查本地缓存歌词是否为正确版本，可删除 Cache/ 目录下的 .lrc 文件重新搜索

### Q: OBS 背景不透明
A: 在浏览器源的自定义 CSS 中添加：
```css
body { background: transparent !important; }
#app { background: transparent !important; }
```

## 开发

### 调试

1. 启动后端：`.\bridge.exe` 或 `go run main.go`
2. 打开浏览器开发者工具
3. 查看 Console 输出

### 构建

```bash
go build -o bridge.exe
```

### 热重载

修改 JS 文件后刷新浏览器即可，无需重启后端。

## 许可证

MIT License
