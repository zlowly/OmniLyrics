# OmniLyrics 需求与设计规格文档

## 1. 项目概述

### 1.1 项目名称
OmniLyrics

### 1.2 项目类型
桌面歌词引擎 (Desktop Lyrics Engine)

### 1.3 核心功能
通过监听系统媒体播放状态，自动获取并显示匹配的歌词，支持多种展示模式，专为 OBS 直播场景优化。

### 1.4 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | 原生 JavaScript + HTML/CSS |
| 动画 | GSAP (由 Webflow 赞助，全面免费) |
| 后端 | Go (golang) |
| Windows API | winrt-go (SMTC 系统媒体传输控制) |
| 数据格式 | JSON |

---

## 2. 系统架构

### 2.1 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      前端 (Browser)                        │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │ index.html │  │ motion.js   │  │ renderers/*     │   │
│  │ settings  │  │ (调度引擎) │  │ (渲染器模块)    │   │
│  └─────────────┘  └──────────────┘  └──────────────────┘   │
│        ↓                ↓                                   │
│  ┌──────────────────────────────────────────────┐      │
│  │ lyrics/scheduler.js - 歌词源调度器           │      │
│  │ lyrics/providers/ - 歌词源模块            │      │
│  └──────────────────────────────────────────────┘      │
│                            ↓                           │
│  ┌─────────────────────────────────────────┐           │
│  │ config.js - 配置管理                      │           │
│  └─────────────────────────────────────────┘           │
└───────────────────────────────────────────────────────────────┘
                              │
                    HTTP:8080/* API
                              │
┌───────────────────────────────┴───────────────────────────┐
│                     后端 (Go)                        │
│  ┌──────────────┐  ┌────────────┐  ┌───────────────┐ │
│  │ /status    │  │ /decrypt  │  │ /config/*   │ │
│  │ /smtc     │  │ (QRC解密) │  │            │ │
│  └──────────────┘  └────────────┘  └───────────────┘ │
│         ↑              ↑                               │
│  ┌─────┴─────────────────────────────────┐         │
│  │ WinRT: SMTC (系统媒体传输控制)        │         │
│  │ Mock: 模拟实现 (非Windows)          │         │
│  └───────────────────────────────��───────────┘         │
└─────────────────────────────────────────────────────┘
```

### 2.2 模块说明

| 模块 | 职责 |
|------|------|
| motion.js | 心跳监测、帧数据计算 |
| config.js | 配置读取/保存 |
| renderers/base.js | 渲染器基类 |
| renderers/karaoke.js | 单行卡拉OK |
| renderers/scroll.js | 双行滚动 |
| renderers/blur.js | 多行渐变模糊 |
| lyrics/index.js | 歌词调度器（优先级/并行搜索） |
| lyrics/providers/*.js | 歌词源实现 |
| main.go | Go 后端入口 |
| handlers.go | HTTP 接口处理 |
| smtc/*.go | SMTC 实现 |

---

## 3. API 接口

### 3.1 HTTP 接口

| 端点 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/status` | GET | 简化播放状态 (title, artist, status, position, duration) |
| `/smtc` | GET | 完整播放状态 (含 albumTitle, appName, positionMs, durationMs) |
| `/smtc` | GET | 播放状态 (完整 SMTCData) |
| `/check_cache` | GET | 查询歌词缓存 (title, artist) |
| `/update_cache` | POST | 更新歌词缓存 (JSON Body) |
| `/config` | GET/POST | 渲染器配置 |
| `/config/lyrics` | GET/POST | 歌词源配置 |
| `/decrypt` | POST | QRC 歌词解密 |
| `/fonts` | GET | 系统字体列表 |
| `/shutdown` | GET/POST | 关闭服务 |

### 3.2 数据格式

**/status 返回**：
```json
{
  "title": "歌曲名",
  "artist": "艺术家",
  "status": "Playing",
  "position": 51260,
  "duration": 211000
}
```

**/smtc 返回**：
```json
{
  "status": "Playing",
  "title": "歌曲名",
  "artist": "艺术家",
  "albumTitle": "专辑名",
  "positionMs": 51260,
  "durationMs": 211000,
  "hasSession": true,
  "appName": "QQMusic.exe"
}
```

**/config/lyrics 返回**：
```json
{
  "timeout": 5000,
  "retry": 1,
  "sources": [
    { "name": "lrclib", "enabled": true, "priority": 1, "apps": ["*"] },
    { "name": "qqmusic", "enabled": true, "priority": 2, "apps": ["QQMusic.exe", "*"] }
  ]
}
```

**/decrypt 请求**：
```json
{"encrypted": "4A5B6C..."}
```

**/decrypt 返回**：
```json
{"lyrics": "[00:00.00]原词", "error": ""}
```

---

## 4. 歌词源调度

### 4.1 搜索流程

```
1. 获取当前播放信息 (/status)
2. 检测歌曲变化 → 检查本地缓存 (/check_cache)
3. 无本地缓存 → 歌词调度器搜索
4. 按优先级顺序搜索，相同优先级并行
5. 优先返回有逐字时间戳的结果
6. 结果写入本地缓存 (/update_cache)
```

### 4.2 调度配置

| 字段 | 类型 | 描述 |
|------|------|------|
| timeout | number | 单源超时(毫秒) |
| retry | number | 失败重试次数 |
| sources | array | 歌词源列表 |

### 4.3 源配置

| 字段 | 类型 | 描述 |
|------|------|------|
| name | string | 源名称 (lrclib, qqmusic) |
| enabled | boolean | 是否启用 |
| priority | number | 优先级 (数字越小越高) |
| apps | array | 适用App列表，* 表示全部 |

---

## 5. 展示模式规格

### 5.1 单行卡拉OK (karaoke)

逐字高亮发光效果，当前字有缩放动画。

**参数**：
| 参数 | 默认值 | 描述 |
|------|--------|------|
| wordAnimation | true | 逐字动画开关 |
| animationDuration | 0.3 | 动画时长(秒) |
| currentScale | 1.05 | 当前字缩放比例 |

### 5.2 双行滚动 (scroll)

当前行在上、下一行在下交替滚动。

**参数**：
| 参数 | 默认值 | 描述 |
|------|--------|------|
| showNext | true | 显示下一行 |
| nextOpacity | 0.6 | 下一行透明度 |
| scrollDuration | 0.4 | 滚动时长(秒) |

### 5.3 多行渐变模糊 (blur)

一屏多行，亮度/模糊/尺寸递减。

**参数**：
| 参数 | 默认值 | 描述 |
|------|--------|------|
| visibleLines | 9 | 显示行数 |
| lineSpacing | 1.5 | 行距 |
| opacityDecay | 0.15 | 亮度衰减 |
| blurIncrement | 0.5 | 模糊递增 |
| scaleDecay | 0.1 | 缩小比例 |
| blurMax | 6 | 最大模糊值 |

---

## 6. 配置规格

### 6.1 渲染器配置

```json
{
  "mode": "karaoke",
  "colors": {
    "text": "#ffffff",
    "bg": "#000000",
    "glowRange": 1,
    "outlineWidth": 1,
    "outlineColor": "#ffffff"
  },
  "font": {
    "size": "2.4rem",
    "family": "Arial, Microsoft YaHei"
  },
  "modeParams": {
    "karaoke": { "wordAnimation": true, "animationDuration": 0.3, "currentScale": 1.05 },
    "scroll": { "showNext": true, "nextOpacity": 0.6, "scrollDuration": 0.4 },
    "blur": { "visibleLines": 9, "lineSpacing": 1.5, "opacityDecay": 0.15, "blurIncrement": 0.5, "scaleDecay": 0.1, "blurMax": 6 }
  }
}
```

### 6.2 歌词源配置

```json
{
  "timeout": 5000,
  "retry": 1,
  "sources": [
    { "name": "lrclib", "enabled": true, "priority": 1, "apps": ["*"] },
    { "name": "qqmusic", "enabled": true, "priority": 2, "apps": ["QQMusic.exe", "*"] }
  ]
}
```

---

## 7. 文件结构

```
OmniLyrics/
├── main.go                    # Go 后端入口
├── handlers.go               # HTTP 处理器
├── config.go               # 配置系统
├── decrypter.go            # QRC 解密
├── handler.go             # /decrypt 接口
├── smtc/
│   ├── smtc.go           # 接口定义
│   ├── smtc_winrt.go     # Windows 实现
│   └── smtc_mock.go     # Mock 实现
├── smtc_factories*.go    # 工厂函数
├── fonts/
│   ├── fonts.go          # 接口
│   ├── fonts_windows.go # Windows 实现
│   └── fonts_linux.go  # Linux 实现
├── go.mod              # Go 模块
├── go.sum
├── Makefile            # 构建脚本
├── README.md          # 项目说明
├── bridge.exe         # 编译后的可执行文件
│
├── Config/                 # 配置目录
│   ├── renderer.json       # 渲染器配置
│   └── lyrics.json        # 歌词源配置
│
├── Cache/                  # 歌词缓存目录
│   └── *.lrc           # 歌词文件
│
├── web/                   # 前端资源
│   ├── index.html        # 主页面
│   ├── settings.html   # 设置页面
│   └── scripts/
│       ├── config.js     # 配置管理
│       ├── motion.js    # 调度引擎
│       ├── renderers/
│       │   ├── index.js
│       │   ├── base.js
│       │   ├── karaoke.js
│       │   ├── scroll.js
│       │   └── blur.js
│       └── lyrics/
│           ├── index.js      # 调度器
│           └── providers/
│               ├── base.js
│               └── qqmusic.js
│
└── docs/
    └── SPEC.md          # 本文档
```

---

## 8. 验收标准

### 8.1 功能验收
- [x] 心跳正常获取播放状态 (/status)
- [x] 换歌时自动获取歌词
- [x] 本地缓存优先，无缓存时请求在线歌词
- [x] 三种展示模式可正常切换
- [x] 配置可保存和读取
- [x] 多歌词源按优先级搜索
- [x] QQ音乐歌词解密

### 8.2 视��验��
- [x] 单行卡拉OK：逐字高亮效果流畅
- [x] 双行滚动：上下行交替滚动无跳跃
- [x] 多行模糊：亮度/模糊/尺寸渐变平滑

### 8.3 性能验收
- [x] 帧率稳定 60fps
- [x] 内存无泄漏
- [x] 切换歌曲无卡顿