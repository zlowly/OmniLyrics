# --- 0. 环境准备 ---
[void][Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media.Control, ContentType=WindowsRuntime]
Add-Type -AssemblyName System.Runtime.WindowsRuntime
Add-Type -AssemblyName System.Web

# 核心 Await 函数
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { 
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1' 
})[0]

Function Await($WinRtTask, $ResultType) {
    try {
        $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
        $netTask = $asTask.Invoke($null, @($WinRtTask))
        if ($netTask.Wait(2000)) { return $netTask.Result }
    } catch { }
    return $null
}

# --- 1. 初始化数据结构 ---
# Payload 存储所有原始锚点数据，供子线程计算进度
$syncHash = [hashtable]::Synchronized(@{
    Payload = @{ 
        Title = "Waiting"; Artist = ""; Status = "Stopped"
        PositionMs = 0; DurationMs = 0; LastUpdatedTimeTicks = 0 
    }
    Running = $true
})

# --- 2. 缓存目录管理 ---
$CacheDir = Join-Path $PSScriptRoot "Cache"
$ConfigDir = Join-Path $PSScriptRoot "Config"
if (-not (Test-Path -LiteralPath $CacheDir)) {
    New-Item -ItemType Directory -Path $CacheDir | Out-Null
}
if (-not (Test-Path -LiteralPath $ConfigDir)) {
    New-Item -ItemType Directory -Path $ConfigDir | Out-Null
}

function Get-CacheFilePath($title, $artist) {
    $safeName = "$($artist)_$($title)".Replace('\','').Replace('/','').Replace(':','_').Replace('*','').Replace('?','').Replace('"','_').Replace('<','_').Replace('>','_').Replace('|','_')
    return Join-Path $CacheDir "$safeName.lrc"
}

# --- 3. Web Server 逻辑 (子线程只读数据，不调用 WinRT 方法) ---
$serverLogic = {
    param($sync, $cacheDir, $configDir, $baseDir)
    try {
        Write-Host "[Server] Starting HTTP server on http://localhost:8080/" -ForegroundColor Cyan
    } catch { }

    function Get-CacheFilePath($title, $artist) {
        $safeName = "$($artist)_$($title)".Replace('\','').Replace('/','').Replace(':','_').Replace('*','').Replace('?','').Replace('"','_').Replace('<','_').Replace('>','_').Replace('|','_')
        return Join-Path $cacheDir "$safeName.lrc"
    }

    function Get-ConfigPath {
        return Join-Path $configDir "renderer.json"
    }

    function Get-WebPath($reqPath) {
        if ($reqPath -eq "/" -or $reqPath -eq "/index.html") {
            return Join-Path $baseDir "web\index.html"
        }
        return Join-Path $baseDir "web$reqPath"
    }

    $listener = New-Object System.Net.HttpListener
    $listener.Prefixes.Add("http://localhost:8080/")
    try {
        $listener.Start()
    } catch {
        Write-Host "[Server] ERROR: Cannot bind to port 8080 - $_" -ForegroundColor Red
        return
    }
    
    Write-Host "[Server] Listening on http://localhost:8080/" -ForegroundColor Green
    
    try {
        while ($sync.Running) {
            $context = $listener.GetContext()
            $res = $context.Response
            $res.Headers.Add("Access-Control-Allow-Origin", "*")
            $res.Headers.Add("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
            $res.Headers.Add("Access-Control-Allow-Headers", "Content-Type")
            $req = $context.Request
            
            # 处理 CORS 预检请求
            if ($req.HttpMethod -eq "OPTIONS") {
                $res.StatusCode = 200
                $res.Close()
                continue
            }
            
            # 路由：/status - 获取播放状态
            if ($req.Url.AbsolutePath -eq "/status") {
                $data = $sync.Payload
                $realPos = $data.PositionMs

                if ($data.Status -eq "Playing" -and $data.LastUpdatedTimeTicks -gt 0) {
                    $nowTicks = [DateTimeOffset]::Now.UtcTicks
                    $elapsedMs = ($nowTicks - $data.LastUpdatedTimeTicks) / 10000
                    $realPos = $data.PositionMs + $elapsedMs
                    if ($realPos -gt $data.DurationMs) { $realPos = $data.DurationMs }
                }

                $output = @{
                    title        = $data.Title
                    artist       = $data.Artist
                    status       = $data.Status
                    position     = [Math]::Floor($realPos)
                    duration     = $data.DurationMs
                }

                $bytes = [System.Text.Encoding]::UTF8.GetBytes(($output | ConvertTo-Json -Compress))
                $res.ContentType = "application/json"
                $res.OutputStream.Write($bytes, 0, $bytes.Length)
                $res.Close()
                continue
            }

            # 路由：/check_cache - 查询缓存
            if ($req.Url.AbsolutePath -eq "/check_cache") {
                # 从 RawUrl 手动解析以正确处理 UTF-8 编码
                $rawUrl = $req.Url.AbsolutePath + "?" + $req.Url.Query
                $title = $null
                $artist = $null
                if ($rawUrl -match "[?&]title=([^&]+)") {
                    $title = [System.Web.HttpUtility]::UrlDecode($matches[1], [System.Text.Encoding]::UTF8)
                }
                if ($rawUrl -match "[?&]artist=([^&]+)") {
                    $artist = [System.Web.HttpUtility]::UrlDecode($matches[1], [System.Text.Encoding]::UTF8)
                }
                $cachePath = Get-CacheFilePath $title $artist
                $result = @{ found = $false; content = "" }
                if ($title -and (Test-Path -LiteralPath $cachePath)) {
                    $result.found = $true
                    $result.content = [string](Get-Content -LiteralPath $cachePath -Raw -Encoding UTF8)
                }
                $bytes = [System.Text.Encoding]::UTF8.GetBytes(($result | ConvertTo-Json -Compress))
                $res.ContentType = "application/json"
                $res.OutputStream.Write($bytes, 0, $bytes.Length)
                $res.Close()
                continue
            }

            # 路由：/update_cache - 更新缓存
            if ($req.Url.AbsolutePath -eq "/update_cache" -and $req.HttpMethod -eq "POST") {
                $body = [System.IO.StreamReader]::new($req.InputStream).ReadToEnd()
                try {
                    $json = $body | ConvertFrom-Json
                    $cachePath = Get-CacheFilePath $json.title $json.artist
                    [System.IO.File]::WriteAllText($cachePath, $json.lrc, [System.Text.Encoding]::UTF8)
                    $res.StatusCode = 200
                } catch {
                    $res.StatusCode = 500
                }
                $res.Close()
                continue
            }

            # 路由：/config - 配置管理
            if ($req.Url.AbsolutePath -eq "/config") {
                $configPath = Get-ConfigPath
                if ($req.HttpMethod -eq "GET") {
                    if (Test-Path -LiteralPath $configPath) {
                        $content = Get-Content -LiteralPath $configPath -Raw -Encoding UTF8
                        $bytes = [System.Text.Encoding]::UTF8.GetBytes($content)
                        $res.ContentType = "application/json"
                        $res.OutputStream.Write($bytes, 0, $bytes.Length)
                    } else {
$defaultConfig = @{
                        mode = "karaoke"
                        colors = @{
                            text = "#ffffff"
                            bg = "#000000"
                            glowRange = 1
                            outlineWidth = 1
                            outlineColor = "#ffffff"
                        }
                        font = @{
                            size = "2.4rem"
                            family = "system-ui, -apple-system, Arial"
                        }
                        bg = @{
                            color = "#000000"
                        }
                        modeParams = @{
                            karaoke = @{
                                wordAnimation = $true
                                animationDuration = 0.3
                                currentScale = 1.05
                            }
                            scroll = @{
                                showNext = $true
                                nextOpacity = 0.6
                                scrollDuration = 0.4
                            }
                            blur = @{
                                visibleLines = 9
                                lineSpacing = 1.5
                                opacityDecay = 0.15
                                blurIncrement = 0.5
                                scaleDecay = 0.1
                                blurMax = 6
                                scrollSpeed = "linear"
                                scrollDuration = 0.5
                            }
                            }
                        }
                        $bytes = [System.Text.Encoding]::UTF8.GetBytes(($defaultConfig | ConvertTo-Json -Compress))
                        $res.ContentType = "application/json"
                        $res.OutputStream.Write($bytes, 0, $bytes.Length)
                    }
                    $res.Close()
                    continue
                }
                if ($req.HttpMethod -eq "POST") {
                    $body = [System.IO.StreamReader]::new($req.InputStream).ReadToEnd()
                    try {
                        [System.IO.File]::WriteAllText($configPath, $body, [System.Text.Encoding]::UTF8)
                        $res.StatusCode = 200
                    } catch {
                        $res.StatusCode = 500
                    }
                    $res.Close()
                    continue
                }
            }

            # 路由：静态文件服务（仅限 /web 目录，安全隔离）
            $reqPath = $req.Url.AbsolutePath
            
            # 安全检查：禁止路径遍历
            if ($reqPath -match "\.\.") { $res.StatusCode = 403; $res.Close(); continue }
            
            # 使用函数获取路径
            $staticPath = Get-WebPath $reqPath

            # 检查文件是否存在且为文件（非目录）
            if ($staticPath -and (Test-Path -LiteralPath $staticPath) -and -not (Test-Path -LiteralPath $staticPath -PathType Container)) {
                $ext = [System.IO.Path]::GetExtension($staticPath).ToLower()
                $mimeTypes = @{
                    ".html" = "text/html; charset=utf-8"
                    ".htm" = "text/html; charset=utf-8"
                    ".js" = "application/javascript; charset=utf-8"
                    ".css" = "text/css; charset=utf-8"
                    ".json" = "application/json; charset=utf-8"
                    ".png" = "image/png"
                    ".jpg" = "image/jpeg"
                    ".jpeg" = "image/jpeg"
                    ".gif" = "image/gif"
                    ".svg" = "image/svg+xml"
                    ".woff" = "font/woff"
                    ".woff2" = "font/woff2"
                    ".ttf" = "font/ttf"
                    ".eot" = "application/vnd.ms-fontobject"
                }
                $contentType = $mimeTypes[$ext]
                if (-not $contentType) { $contentType = "text/plain; charset=utf-8" }
                $content = Get-Content -LiteralPath $staticPath -Raw -Encoding UTF8
                $bytes = [System.Text.Encoding]::UTF8.GetBytes($content)
                $res.ContentType = $contentType
                $res.OutputStream.Write($bytes, 0, $bytes.Length)
                $res.Close()
                continue
            }

            # 路由：/shutdown - 优雅关闭
            if ($req.Url.AbsolutePath -eq "/shutdown") {
                $sync.Running = $false
                $res.StatusCode = 200
                $res.Close()
                continue
            }

            # 默认返回 404
            $res.StatusCode = 404
            $res.Close()
        }
    } finally { $listener.Stop() }
}

# --- 3. 启动后台线程 ---
$baseDir = $PSScriptRoot
$rs = [runspacefactory]::CreateRunspace(); $rs.Open()
$rs.SessionStateProxy.SetVariable('syncHash', $syncHash)
$powershell = [powershell]::Create().AddScript($serverLogic).AddArgument($syncHash).AddArgument($CacheDir).AddArgument($ConfigDir).AddArgument($baseDir)
$powershell.Runspace = $rs
$handle = $powershell.BeginInvoke()

# --- 4. 主线程：唯一负责调用 WinRT/SMTC 的地方 ---
Write-Host "[OmniLyrics] 服务启动中..." -ForegroundColor Green
Start-Sleep -Milliseconds 500

try {
    # 在主线程初始化 Manager
    $mgrTask = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()
    $Manager = Await $mgrTask ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager])

    while ($syncHash.Running) {
        if ($null -ne $Manager) {
            # 主线程安全地获取当前会话
            $session = $Manager.GetCurrentSession()
            if ($null -ne $session) {
                $props = Await ($session.TryGetMediaPropertiesAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties])
                $playback = $session.GetPlaybackInfo()
                $timeline = $session.GetTimelineProperties()

                if ($null -ne $props -and $props.Title -ne "") {
                    # 将所有原始数据打包进 Payload
                    $syncHash.Payload = @{
                        Title                = $props.Title
                        Artist               = $props.Artist
                        Status               = $playback.PlaybackStatus.ToString()
                        PositionMs           = $timeline.Position.TotalMilliseconds
                        DurationMs           = $timeline.EndTime.TotalMilliseconds
                        LastUpdatedTimeTicks = $timeline.LastUpdatedTime.UtcTicks
                    }
                    Write-Host "[Sync] $($props.Title) @ $($timeline.Position.TotalSeconds)s" -ForegroundColor Gray
                }
            }
        }
        
        if ([console]::KeyAvailable -and ([console]::ReadKey($true).Key -eq "Escape")) { break }
        Start-Sleep -Milliseconds 800
    }
} finally {
    $syncHash.Running = $false
    try { (New-Object System.Net.WebClient).DownloadString("http://localhost:8080/shutdown") } catch {}
    $rs.Close()
    Write-Host "服务已安全退出。" -ForegroundColor Cyan
}