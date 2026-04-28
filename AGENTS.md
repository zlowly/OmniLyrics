# OmniLyrics 开发指南

## 项目概述

Go 后端 + 原生前端，为 OBS 直播场景提供桌面歌词引擎。

## 构建命令

```bash
# Linux 构建
make build-linux

# Windows 构建（需要 CGO）
make build-windows

# 或手动
go build -o omnilyrics-bridge main.go           # Linux
GOOS=windows CGO_ENABLED=1 go build -tags windows -o omnily-bridge.exe main.go  # Windows
```

## 关键架构
见docs/SPEC.md

### 依赖注意

- `go-ole` 是直接依赖，运行 `go mod tidy` 修复 indirect 警告
- Windows 构建才需要 winrt-go + CGO

## 开发注意

1. Mock 模拟 4 分钟歌曲循环播放（240s + 5s 暂停）
2. HTTP 服务默认端口 9090
3. 前端静态文件位于 `web/` 目录

## 运行注意

- Linux 下使用 `go run .`（不是 `go run main.go`），因为 build tag 文件不会随 main.go 自动包含
- Windows 下使用 `go run .` 或直接运行编译好的二进制文件

## 编码与注释规范
1. 必须为所有新增或修改的函数添加标准注释。
2. 注释语言：中文。
3. 修改逻辑后，必须同步更新对应的注释内容。
4. 在逻辑复杂的代码块上方添加行内注释，解释其背后的逻辑。
5. 保持代码原有的缩进和结构不变。
