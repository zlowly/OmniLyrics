# OmniLyrics 开发指南

## 项目概述

Go 后端 + 原生 JavaScript 前端，为 OBS 直播场景提供桌面歌词引擎。

详细架构见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## 快速启动

### 开发运行

```bash
# 所有平台都使用
go run .

# 或者使用 Makefile
make run
```

**⚠️ 重要**：使用 `go run .` 而不是 `go run main.go`，因为 build tag 文件（factory_windows.go/factory_unix.go）不会自动包含。

### 服务端口

- **默认端口**：9090（可在 config.json 或 --port 参数中修改）
- **Web 访问**：http://localhost:9090/
- **设置页面**：http://localhost:9090/settings.html

## 配置系统

使用 viper + pflag 管理配置，配置默认值集中在 `config_default.json` 文件中，**优先级从高到低**：

1. 命令行参数（--port, --log-level 等）
2. config.json 文件（默认在执行文件同目录）
3. config_default.json（嵌入到二进制文件）

### 常用参数

```bash
# 指定端口
go run . --port 8080

# 指定日志级别（debug/info/warn/error）
go run . --log-level debug

# 启用 Mock 后端（非 Windows 开发测试）
go run . --mock

# 自定义缓存目录
go run . --cache-dir /path/to/cache

# 加载特定配置文件
go run . --config /path/to/config.json
```

## 构建命令

### 自动构建（推荐）

```bash
make build-linux    # Linux 构建
make build-windows  # Windows 构建（需要 CGO_ENABLED=1）
```

### 手动构建

```bash
# Linux
go build -o omnilyrics-bridge .

# Windows（必须指定 CGO_ENABLED=1 和 build tag）
GOOS=windows CGO_ENABLED=1 go build -tags windows -o omnilyrics-bridge.exe .
```

## 平台检测与 SMTC 后端

SMTC 层通过 build tag 文件自动选择平台实现：

- **Windows**：`smtc/factory_windows.go` → WinRT + KugouCatcher（酷狗进度抓取）
- **Linux/Mac**：`smtc/factory_unix.go` → Mock 后端（4 分钟歌曲循环播放：240s + 5s 暂停）

在非 Windows 环境强制使用 Mock 后端：

```bash
go run . --mock
```

## 关键依赖

- `go-ole`：直接依赖（修改后运行 `go mod tidy` 修复 indirect 警告）
- `winrt-go`：Windows 专属，仅在 Windows 构建时需要
- `viper`：配置管理库
- `pflag`：命令行参数解析

## 歌词获取系统

### 架构

- **lyrics/fetcher.go**：调度器，管理多个歌词源
- **lyrics/cache.go**：本地缓存（存储在 Cache 目录）
- **lyrics/sources/interface.go**：歌词源接口定义
- **lyrics/sources/***：具体实现（lrclib.go、qqmusic.go、kgmusic.go）

### 添加新歌词源

在 `lyrics/sources/` 中创建新文件，实现 `Provider` 接口：

```go
type Provider interface {
    Search(title, artist string) (string, error)
    Name() string
}
```

## HTTP API 结构

所有响应启用 CORS（允许所有来源）。

主要端点：
- `GET /status` → 当前播放状态（SMTC 数据）
- `GET /lyrics` → 搜索歌词
- `GET /health` → 健康检查
- `POST /hold` → 冻结状态
- `GET/POST /config/*` → 配置管理
- `GET /fonts` → 字体列表

详见 handlers.go

## 编码与注释规范

1. **注释语言**：必须使用中文
2. **函数注释**：所有新增或修改的函数必须添加标准注释
   ```go
   // FunctionName 做了什么事情。
   // 详细说明参数和返回值的作用。
   func FunctionName(param string) error {
       // ...
   }
   ```
3. **行内注释**：在逻辑复杂的代码块上方添加注释，解释其背后的逻辑
4. **修改同步**：修改逻辑后必须同步更新对应的注释
5. **缩进保持**：保持代码原有的缩进和结构不变

## 测试与调试

### 开发模式

- **Mock 播放**：使用 `--mock` 参数在任何平台测试播放逻辑
- **调试日志**：使用 `--log-level debug` 查看详细日志
- **热重载**：修改 JS 文件后刷新浏览器即可，无需重启后端

### 手动测试

在 `tests/` 目录中有测试脚本：
- `Bridge.ps1`：PowerShell 启动脚本（Windows）
- `test_smfc.ps1`、`test_cache.ps1`：功能测试（Windows）
- `kugou_catch.go`、`lyrics_qm.go`、`lyrics_kg.go`：歌词源测试

### 缓存清理

删除 `Cache/` 目录下的 `.lrc` 文件即可强制重新搜索歌词。

## 交互要求

思考过程和回答使用中文

## 文档导航

| 文档 | 说明 |
|------|------|
| [README.md](README.md) | 项目概述、快速开始、支持的播放器、常见问题 |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 系统架构详解、模块设计、数据流 |
| [docs/SPEC.md](docs/SPEC.md) | 需求与功能规格 |
| [docs/CONFIG.md](docs/CONFIG.md) | 配置文件详细说明 |
| [docs/PROVIDER.md](docs/PROVIDER.md) | 歌词源开发指南 |
| [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | 故障排除 |
