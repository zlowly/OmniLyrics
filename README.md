# OmniLyrics

桌面歌词引擎 - 为 OBS 直播场景优化的桌面歌词显示工具。

## 功能特性

- **系统媒体同步**：自动监听 Windows 系统媒体播放状态，实时同步歌曲播放进度
- **多源歌词搜索**：支持 lrclib、QQ 音乐，按优先级搜索，优先使用逐字时间戳
- **多种展示模式**：
  - 单行卡拉OK - 逐字高亮发光效果
  - 双行滚动 - 当前行在上、下一行在下的交替滚动
  - 多行渐变模糊 - 一屏多行，亮度/模糊/尺寸渐变效果
- **OBS 优化**：透明背景、支持多层叠加
- **可配置**：支持自定义颜色、字体、动画参数

## 快速开始

### 1. 运行

```powershell
.\bridge.exe
```

### 2. 打开歌词页面

浏览器访问：http://localhost:8080/

### 3. OBS 配置

1. 添加浏览器源
2. URL 设置为 `http://localhost:8080/`
3. 宽度：1920，高度：1080（根据需要调整）
4. 自定义 CSS 中设置透明：

```css
body {
    background: transparent !important;
}
```

## 配置

访问设置页面：http://localhost:8080/settings.html

- **歌词源设置**：配置搜索源优先级、超时、适用App
- **展示设置**：选择模式、颜色、字体、动画参数

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
### Q: 只有QQ播放器等少数播放器能滚动歌词
A: 目前仅技术上通过Windows的SMTC获取播放器的播放进度，而酷狗音乐没有提供播放进度，网易云不支持SMTC（可通过安装BetterNCM支持）

## 开发

### 启动
```powershell
.\bridge.exe
```
或
```powershell
go run main.go
```

### 构建

```powershell
go build -o bridge.exe
```

### 热重载

修改 JS 文件后刷新浏览器即可，无需重启后端。

## 许可证

GPL License

### 鸣谢

- GSAP - 由 GreenSock 开发。自 2025 年 5 月起，GSAP 已由 Webflow 赞助并全面开放免费使用。