# OmniLyrics 配置详解

本文档详细说明 OmniLyrics 的所有配置项，包括后端配置和前端配置。

## 1. 后端配置

后端配置通过 `config.json` 文件和命令行参数共同控制。

### 1.1 配置文件 (config.json)

配置文件位于程序运行目录，也可通过 `--config` 参数指定路径。

**示例配置**：
```json
{
    "port": "8081",
    "log": {
        "level": "info",
        "file": ""
    },
    "cache-dir": "Cache",
    "config-dir": "Config",
    "mock": false
}
```

### 1.2 配置项说明

#### port
- **类型**：string
- **默认值**："8081"
- **命令行**：`-p` 或 `--port`
- **说明**：HTTP 服务器监听端口

#### log.level
- **类型**：string
- **可选值**：debug, info, warn, error
- **默认值**："info"
- **命令行**：`-l` 或 `--log-level`
- **说明**：日志输出级别，级别越低输出越详细

| 级别 | 说明 |
|------|------|
| debug | 输出所有日志，包括详细调试信息 |
| info | 输出一般信息（默认） |
| warn | 仅输出警告和错误 |
| error | 仅输出错误 |

#### log.file
- **类型**：string
- **默认值**：""（输出到 stdout）
- **命令行**：`--log-file`
- **说明**：日志文件路径，为空时输出到标准输出

#### cache-dir
- **类型**：string
- **默认值**："Cache"
- **命令行**：`--cache-dir`
- **说明**：歌词缓存目录，存储下载的 `.lrc` 文件

#### config-dir
- **类型**：string
- **默认值**："Config"
- **命令行**：`--config-dir`
- **说明**：配置文件目录，存储 `renderer.json` 和 `lyrics.json`

#### mock
- **类型**：bool
- **默认值**：false
- **命令行**：`--mock`
- **说明**：强制使用 Mock SMTC 后端，用于开发测试

### 1.3 命令行参数完整列表

```bash
./omnilyrics-bridge [选项]

选项：
  -p, --port string         HTTP 服务器端口 (默认: "8081")
  -l, --log-level string    日志级别 (debug/info/warn/error) (默认: "info")
      --log-file string     日志文件路径 (默认: 输出到 stdout)
      --cache-dir string    缓存目录 (默认: "./Cache")
      --config-dir string   配置目录 (默认: "./Config")
  -c, --config string      配置文件路径 (默认: "config.json")
      --mock                强制使用 Mock SMTC 后端
```

### 1.4 配置优先级

优先级从高到低：
1. 命令行参数
2. 配置文件 (通过 `-c` 指定的文件或 `config.json`)
3. 默认值

## 2. 渲染器配置

渲染器配置存储于 `<config-dir>/renderer.json`，可通过设置页面或 API 修改。

### 2.1 完整配置结构

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
        "family": "system-ui, -apple-system, Arial"
    },
    "bg": {
        "color": "#000000"
    },
    "modeParams": {
        "karaoke": {
            "wordAnimation": true,
            "animationDuration": 0.3,
            "currentScale": 1.05
        },
        "scroll": {
            "showNext": true,
            "nextOpacity": 0.6,
            "scrollDuration": 0.4
        },
        "blur": {
            "visibleLines": 9,
            "lineSpacing": 1.5,
            "opacityDecay": 0.15,
            "blurIncrement": 0.5,
            "scaleDecay": 0.1,
            "blurMax": 6,
            "scrollSpeed": "linear"
        }
    }
}
```

### 2.2 通用配置项

#### mode
- **类型**：string
- **可选值**：karaoke, scroll, blur
- **说明**：歌词展示模式

#### colors
- **text**：歌词文字颜色（十六进制）
- **bg**：歌词背景色（通常用于调试，实际 OBS 中为透明）
- **glowRange**：发光范围（像素）
- **outlineWidth**：描边宽度（像素）
- **outlineColor**：描边颜色

#### font
- **size**：字体大小（CSS 单位，如 `2.4rem`）
- **family**：字体族（CSS font-family 值）

#### bg
- **color**：背景颜色（OBS 中通常设置为透明）

### 2.3 模式特定参数

#### karaoke（单行卡拉OK）

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| wordAnimation | boolean | true | 是否启用逐字动画 |
| animationDuration | number | 0.3 | 动画时长（秒） |
| currentScale | number | 1.05 | 当前字缩放比例 |

#### scroll（双行滚动）

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| showNext | boolean | true | 是否显示下一行 |
| nextOpacity | number | 0.6 | 下一行透明度 (0-1) |
| scrollDuration | number | 0.4 | 滚动动画时长（秒） |

#### blur（多行渐变模糊）

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| visibleLines | number | 9 | 显示行数 |
| lineSpacing | number | 1.5 | 行距（倍数） |
| opacityDecay | number | 0.15 | 亮度衰减 (0-1) |
| blurIncrement | number | 0.5 | 模糊递增 (px) |
| scaleDecay | number | 0.1 | 缩小比例 (0-1) |
| blurMax | number | 6 | 最大模糊值 (px) |
| scrollSpeed | string | "linear" | 滚动速度曲线 |

### 2.4 通过 API 管理渲染器配置

```bash
# 获取当前配置
curl http://localhost:8081/config

# 更新配置
curl -X POST http://localhost:8081/config \
  -H "Content-Type: application/json" \
  -d '{"mode": "blur", "font": {"size": "3rem"}}'
```

## 3. 歌词源配置

歌词源配置存储于 `<config-dir>/lyrics.json`，控制歌词搜索行为。

### 3.1 完整配置结构

```json
{
    "timeout": 5000,
    "retry": 1,
    "sources": [
        {
            "name": "lrclib",
            "enabled": true,
            "priority": 1,
            "apps": ["*"]
        },
        {
            "name": "qqmusic",
            "enabled": true,
            "priority": 2,
            "apps": ["QQMusic.exe", "*"]
        },
        {
            "name": "kgmusic",
            "enabled": true,
            "priority": 3,
            "apps": ["kugou", "*"]
        }
    ]
}
```

### 3.2 配置项说明

#### timeout
- **类型**：number
- **默认值**：5000
- **单位**：毫秒
- **说明**：单个歌词源搜索超时时间

#### retry
- **类型**：number
- **默认值**：1
- **说明**：失败重试次数

#### sources
歌词源列表，按优先级排序。

**源配置字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 源名称（lrclib/qqmusic/kgmusic） |
| enabled | boolean | 是否启用该源 |
| priority | number | 优先级（数字越小优先级越高） |
| apps | string[] | 适用的播放器列表，`*` 表示全部 |

### 3.3 apps 字段说明

`apps` 字段用于控制该歌词源在哪些播放器下生效：

- `["*"]` - 对所有播放器生效
- `["QQMusic.exe", "*"]` - 优先匹配 QQ 音乐，其他播放器也生效
- `["kugou"]` - 仅对酷狗音乐生效

**匹配逻辑**：
1. 后端获取当前播放器名称（`AppName`）
2. 如果 `appName` 在源的 `apps` 列表中，该源优先级提升
3. 如果 `apps` 包含 `*`，该源作为通用源

### 3.4 通过 API 管理歌词源配置

```bash
# 获取当前配置
curl http://localhost:8081/config/lyrics

# 更新配置
curl -X POST http://localhost:8081/config/lyrics \
  -H "Content-Type: application/json" \
  -d '{
    "timeout": 3000,
    "sources": [
      {"name": "lrclib", "enabled": true, "priority": 1, "apps": ["*"]}
    ]
  }'
```

### 3.5 热重载

歌词源配置支持热重载，修改 `lyrics.json` 后无需重启服务，下次歌词搜索时会自动加载新配置。

## 4. 配置示例

### 4.1 开发环境配置

```json
// config.json
{
    "port": "8081",
    "log": {
        "level": "debug",
        "file": ""
    },
    "cache-dir": "./Cache",
    "config-dir": "./Config",
    "mock": true
}
```

### 4.2 生产环境配置

```json
// config.json
{
    "port": "8080",
    "log": {
        "level": "info",
        "file": "/var/log/omnilyrics.log"
    },
    "cache-dir": "/var/lib/omnilyrics/cache",
    "config-dir": "/etc/omnilyrics"
}
```

### 4.3 仅使用 lrclib 歌词源

```json
// Config/lyrics.json
{
    "timeout": 5000,
    "retry": 2,
    "sources": [
        {
            "name": "lrclib",
            "enabled": true,
            "priority": 1,
            "apps": ["*"]
        }
    ]
}
```

### 4.4 OBS 透明背景配置

```json
// Config/renderer.json
{
    "mode": "karaoke",
    "colors": {
        "text": "#00ffaa",
        "bg": "transparent",
        "glowRange": 2,
        "outlineWidth": 1,
        "outlineColor": "#ffffff"
    },
    "font": {
        "size": "2.4rem",
        "family": "Microsoft YaHei, Arial"
    }
}
```

## 5. 配置目录结构

```
项目根目录/
├── config.json              # 后端配置（可选）
├── Config/                  # 配置目录
│   ├── renderer.json       # 渲染器配置
│   └── lyrics.json         # 歌词源配置
└── Cache/                   # 缓存目录
    └── 艺术家_歌曲名.lrc   # 歌词缓存文件
```

## 6. 注意事项

1. **路径处理**：相对路径基于程序运行目录（非可执行文件目录）
2. **配置文件编码**：请使用 UTF-8 编码保存 JSON 配置文件
3. **权限问题**：确保程序对缓存和配置目录有读写权限
4. **热重载限制**：仅歌词源配置支持热重载，后端配置需要重启生效
