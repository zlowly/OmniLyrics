# OmniLyrics 架构文档

## 1. 系统概述

OmniLyrics 是一个为 OBS 直播场景设计的桌面歌词引擎，采用前后端分离架构：

- **后端**：Go 实现，负责媒体状态监听、歌词获取与缓存
- **前端**：原生 JavaScript + GSAP 动画库，负责歌词渲染与展示

## 2. 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      前端 (Browser)                        │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐ │
│  │ index.html │  │ motion.js   │  │ renderers/*     │ │
│  │ settings   │  │ (运动引擎)   │  │ (渲染器模块)    │ │
│  └─────────────┘  └──────────────┘  └──────────────────┘ │
│        ↓                ↓                                   │
│  ┌──────────────────────────────────────────────┐          │
│  │ lyrics/scheduler.js - 歌词源调度器           │          │
│  │ lyrics/providers/ - 歌词源模块              │          │
│  └──────────────────────────────────────────────┘          │
│                            ↓                               │
│  ┌─────────────────────────────────────────┐               │
│  │ config.js - 配置管理                      │               │
│  └─────────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────┘
                               │
                     HTTP API (port 8080/8081)
                               │
┌───────────────────────────────┴───────────────────────────┐
│                     后端 (Go)                             │
│  ┌──────────────┐  ┌────────────┐  ┌───────────────┐    │
│  │ /status     │  │ /lyrics   │  │ /config/*    │    │
│  │ /smtc       │  │ /check_   │  │              │    │
│  │ /hold       │  │ cache     │  │ /fonts       │    │
│  └──────────────┘  └────────────┘  └───────────────┘    │
│         ↑              ↑                                    │
│  ┌─────┴─────────────────────────────────┐              │
│  │ SMTC 层 (系统媒体传输控制)             │              │
│  │ ├─ WinRT (Windows SMTC)               │              │
│  │ ├─ KugouCatcher (酷狗进度抓取)        │              │
│  │ └─ Mock (非Windows/调试)              │              │
│  └────────────────────────────────────────┘              │
│         ↑                                                   │
│  ┌─────┴─────────────────────────────────┐              │
│  │ 歌词获取层                            │              │
│  │ ├─ fetcher.go (调度器)                │              │
│  │ ├─ cache.go (本地缓存)                │              │
│  │ └─ sources/* (歌词源实现)             │              │
│  └────────────────────────────────────────┘              │
└──────────────────────────────────────────────────────────┘
```

## 3. 模块详细说明

### 3.1 后端模块

#### 3.1.1 主程序 (main.go)

职责：
- 初始化配置系统
- 创建 SMTC 后端实例
- 注册 HTTP 路由
- 启动 HTTP 服务器
- 处理优雅关闭

关键流程：
```
main()
  ├─ initConfig()        // 初始化配置
  ├─ smtc.NewSMTC()     // 创建 SMTC 后端
  ├─ lyrics.InitFetcher() // 初始化歌词获取器
  ├─ 注册路由             // /status, /lyrics, /config 等
  └─ ListenAndServe()   // 启动服务
```

#### 3.1.2 SMTC 层 (smtc/)

SMTC（System Media Transport Controls）层负责获取系统媒体播放状态。

**接口定义** (`smtc/smtc.go`)：
```go
type SMTC interface {
    GetData() (SMTCData, error)
    Reset()
    SetWinRTDebug(enabled bool)
    SetKugouCatcherDebug(enabled bool)
}
```

**数据结构** (`smtc/smtc.go`)：
```go
type SMTCData struct {
    Status     string // "Playing", "Paused", "Stopped", "NoSession", "Error"
    Title      string
    Artist     string
    AlbumTitle string
    PositionMs int64
    DurationMs int64
    HasSession bool
    AppName    string
}
```

**实现方式**：

| 实现 | 文件 | 平台 | 说明 |
|------|------|------|------|
| WinRT | smtc_winrt.go | Windows | 使用 winrt-go 访问 Windows SMTC API |
| KugouCatcher | kugou_catcher.go | Windows | 通过 UI Automation 抓取酷狗音乐进度 |
| Hybrid | smtc_hybrid.go | Windows | 组合 WinRT + KugouCatcher，针对酷狗特殊处理 |
| Mock | smtc_mock.go | 跨平台 | 模拟播放状态，用于开发测试 |

**工厂函数**：
- `smtc/factory_windows.go`：Windows 下创建 Hybrid 实例
- `smtc/factory_unix.go`：非 Windows 下创建 Mock 实例

#### 3.1.3 歌词获取层 (lyrics/)

**核心组件**：

| 文件 | 职责 |
|------|------|
| fetcher.go | 歌词获取调度器，协调缓存检查和多源搜索 |
| cache.go | 本地歌词缓存管理（文件存储） |
| config.go | 歌词源配置加载和热重载 |

**获取流程** (`lyrics/fetcher.go:Fetch`)：
```
Fetch(title, artist, duration, appName)
  ├─ CheckCache()        // 检查本地缓存
  │   └─ 命中 → 返回缓存歌词
  ├─ filterSourcesByApp() // 根据 appName 过滤歌词源
  └─ 按优先级搜索
      ├─ lrclib.Search()
      ├─ qqmusic.Search()
      └─ kgmusic.Search()
```

**歌词源接口** (`lyrics/sources/interface.go`)：
```go
type LyricsSource interface {
    Name() string
    Search(ctx context.Context, title, artist string, duration int) (lyrics string, err error)
}
```

**已实现歌词源**：

| 源 | 文件 | 说明 |
|----|------|------|
| lrclib | sources/lrclib.go | 公开歌词 API，支持逐字时间戳 |
| qqmusic | sources/qqmusic.go | QQ 音乐，支持QRC格式 |
| kgmusic | sources/kgmusic.go | 酷狗音乐，支持KRC格式 |

#### 3.1.4 配置系统 (config.go)

配置优先级（从高到低）：
1. 命令行参数
2. 配置文件 (通过 `-c` 指定的文件或 `config.json`)
3. `config_default.json`（嵌入到二进制文件的默认值）

**默认值来源**：所有默认值已集中在 `config_default.json` 文件中，通过 `//go:embed` 嵌入到二进制文件，不再使用代码中的硬编码默认值。

**配置项**：

| 参数 | 命令行 | 配置文件 | 默认值来源 | 说明 |
|------|--------|----------|------------|------|
| 端口 | -p | port | config_default.json | HTTP 服务端口 |
| 日志级别 | -l | log.level | config_default.json | debug/info/warn/error |
| 日志文件 | --log-file | log.file | config_default.json | 日志输出文件 |
| 缓存目录 | --cache-dir | cache-dir | config_default.json | 歌词缓存目录 |
| 配置目录 | --config-dir | config-dir | config_default.json | 配置文件目录 |
| Mock 模式 | --mock | mock | config_default.json | 强制使用 Mock SMTC（调试用) |

### 3.2 前端模块

#### 3.2.1 运动引擎 (motion.js)

`MotionEngine` 类负责：
- 定时轮询后端 `/status` 获取播放状态
- 检测歌曲变化，触发歌词加载
- 计算当前播放位置对应的歌词帧数据
- 调用渲染器进行画面更新

**核心方法**：
- `start()` - 启动轮询循环
- `stop()` - 停止轮询
- `fetchStatus()` - 获取播放状态
- `fetchLyrics()` - 获取歌词
- `updateFrame()` - 计算帧数据并渲染

#### 3.2.2 渲染器 (renderers/)

所有渲染器继承自 `base.js` 基类。

| 渲染器 | 文件 | 说明 |
|--------|------|------|
| BaseRenderer | base.js | 基类，定义通用接口 |
| KaraokeRenderer | karaoke.js | 单行卡拉OK，逐字高亮 |
| ScrollRenderer | scroll.js | 双行滚动 |
| BlurRenderer | blur.js | 多行渐变模糊 |

**渲染器接口**：
```javascript
class BaseRenderer {
    constructor(config) { }
    render(frameData) { }
    setConfig(config) { }
    destroy() { }
}
```

#### 3.2.3 歌词调度器 (lyrics/)

| 文件 | 说明 |
|------|------|
| index.js | 调度器入口，按优先级搜索歌词 |
| providers/base.js | 歌词源基类 |
| providers/lrclib.js | lrclib 源实现 |
| providers/qqmusic.js | QQ音乐源实现 |

## 4. 数据流

### 4.1 播放状态获取流程

```
前端 motion.js
    │
    │ HTTP GET /status (每 500ms)
    ▼
后端 handleStatus()
    │
    │ s.GetData()
    ▼
SMTC 后端 (WinRT/KugouCatcher/Mock)
    │
    │ 返回 SMTCData
    ▼
handleStatus() 处理并返回 JSON
    │
    ▼
前端收到 { title, artist, status, position, duration }
```

### 4.2 歌词获取流程

```
前端检测到歌曲变化
    │
    │ HTTP GET /lyrics?title=...&artist=...
    ▼
后端 handleLyrics()
    │
    │ lyrics.FetchLyrics()
    ▼
lyrics.Fetcher.Fetch()
    │
    ├─ 检查本地缓存 (Cache/*.lrc)
    │   └─ 命中 → 返回
    │
    └─ 未命中 → 按优先级搜索
        │
        ├─ lrclib.Search()
        ├─ qqmusic.Search()
        └─ kgmusic.Search()
            │
            └─ 找到 → 写入缓存 → 返回
```

### 4.3 配置更新流程

```
前端 settings.html
    │
    │ HTTP GET /config → 读取当前配置
    │ HTTP POST /config → 保存新配置
    ▼
后端 handleConfig()
    │
    ├─ GET → 读取 Config/renderer.json 或返回默认配置
    └─ POST → 写入 Config/renderer.json
```

## 5. 关键设计决策

### 5.1 为什么使用 Hybrid 模式处理酷狗音乐？

Windows SMTC API 无法获取酷狗音乐的播放进度（酷狗未正确实现 SMTC 接口）。解决方案：
- 使用 UI Automation 直接抓取酷狗窗口中的进度条控件
- `KugouCatcher` 通过正则表达式定位酷狗主窗口，然后查找名为"进度"的滑块控件

### 5.2 为什么歌词获取每次都重新加载配置？

实现热重载功能：
- 每次请求都调用 `LoadConfig()` 重新读取配置文件
- 用户修改 `Config/lyrics.json` 后无需重启服务

### 5.3 为什么使用 CORS 中间件？

前端可能从不同端口或域访问 API（如开发时的 live server），CORS 中间件允许跨域请求。

### 5.4 前端为什么使用原生 JS 而非框架？

- 项目面向 OBS 浏览器源，资源受限
- 原生 JS 无依赖，加载更快
- GSAP 是专门的动画库，性能优秀

## 6. 目录结构详解

```
OmniLyrics/
├── main.go                    # 入口：启动服务、注册路由
├── handlers.go               # HTTP 处理器：/status, /lyrics 等
├── config.go                 # 配置系统：加载、解析、日志
├── smtc/                     # SMTC 层
│   ├── smtc.go              # 接口定义和 SMTCData 结构
│   ├── smtc_winrt.go        # Windows SMTC 实现 (winrt-go)
│   ├── kugou_catcher.go     # 酷狗进度抓取 (UI Automation)
│   ├── smtc_hybrid.go       # 混合模式：WinRT + KugouCatcher
│   ├── smtc_mock.go         # Mock 实现（模拟播放）
│   ├── factory_windows.go   # Windows 工厂函数
│   └── factory_unix.go      # Unix 工厂函数
├── lyrics/                   # 歌词获取层
│   ├── fetcher.go           # 调度器：缓存检查 + 多源搜索
│   ├── cache.go             # 缓存管理：读写 .lrc 文件
│   ├── config.go            # 配置加载：lyrics.json
│   └── sources/             # 歌词源实现
│       ├── interface.go     # LyricsSource 接口
│       ├── lrclib.go        # lrclib 源
│       ├── qqmusic.go       # QQ音乐源
│       └── kgmusic.go       # 酷狗音乐源
├── fonts/                    # 系统字体获取
│   ├── fonts.go            # 接口定义
│   ├── fonts_windows.go    # Windows 实现
│   └── fonts_linux.go      # Linux 实现
├── web/                      # 前端资源
│   ├── index.html          # 主页面（歌词展示）
│   ├── settings.html       # 设置页面
│   ├── libs/
│   │   └── gsap.min.js    # GSAP 动画库
│   └── scripts/
│       ├── motion.js       # 运动引擎：轮询、歌词加载
│       ├── config.js       # 配置管理
│       ├── renderers.js    # 渲染器入口
│       └── renderers/
│           ├── index.js    # 渲染器注册
│           ├── base.js     # 基类
│           ├── karaoke.js  # 单行卡拉OK
│           ├── scroll.js   # 双行滚动
│           └── blur.js     # 多行渐变模糊
├── Config/                   # 配置目录（运行时生成）
│   ├── renderer.json       # 渲染器配置
│   └── lyrics.json         # 歌词源配置
├── Cache/                    # 缓存目录（运行时生成）
│   └── *.lrc               # 歌词缓存文件
├── docs/                     # 文档
│   ├── SPEC.md             # 需求与规格文档
│   ├── ARCHITECTURE.md     # 本文档
│   ├── CONFIG.md           # 配置详解
│   ├── PROVIDER.md         # 歌词源开发指南
│   └── TROUBLESHOOTING.md  # 故障排除
├── config.json              # 后端配置文件（可选）
├── Makefile                 # 构建脚本
├── go.mod                   # Go 模块定义
└── README.md                # 项目说明
```

## 7. 时序图

### 7.1 启动时序

```
用户
 │
 │ 运行 ./omnilyrics-bridge
 ▼
main()
 │
 ├─ initConfig()
 │   ├─ initFlags()      // 解析命令行参数
 │   └─ loadConfig()     // 加载配置文件
 │
 ├─ smtc.NewSMTC(mock)
 │   ├─ [Windows] NewHybrid()
 │   │   ├─ NewWinRT()
 │   │   └─ NewKugouCatcher()
 │   └─ [Unix] NewMock()
 │
 ├─ lyrics.InitFetcher()
 │
 ├─ 注册 HTTP 路由
 │   ├─ /status → makeStatusHandler(smtc)
 │   ├─ /lyrics → handleLyrics
 │   ├─ /config → handleConfigWrapper(configDir)
 │   └─ ...
 │
 └─ ListenAndServe()
```

### 7.2 歌词展示时序

```
motion.js (前端)
 │
 │ setInterval 250ms
 │
 ├─ fetchStatus() ──── HTTP GET /status ────►
 │                                               handleStatus()
 │                                               ├─ smtc.GetData()
 │                                               └─ 返回 JSON
 │
 ├─ 检测歌曲变化?
 │   └─ 是 → fetchLyrics()
 │           ├─ HTTP GET /lyrics?title=...&artist=...
 │           └─► handleLyrics()
 │               ├─ lyrics.FetchLyrics()
 │               │   ├─ CheckCache()
 │               │   └─ [未命中] 搜索歌词源
 │               └─ 返回歌词
 │
 └─ updateFrame()
     ├─ 计算当前歌词行
     └─ renderer.render(frameData)
```
