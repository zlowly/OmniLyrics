@echo off
REM OmniLyrics Bridge - Go Version
echo Starting OmniLyrics Bridge (Go)...

REM Check if bridge.exe exists
if not exist "bridge.exe" (
    echo bridge.exe not found. Building...
    call :build
)

REM Run the bridge
echo Starting server on http://localhost:8080/
bridge.exe
pause
exit /b

:build
echo.
echo Building bridge.exe...
go build -o bridge.exe
if errorlevel 1 (
    echo Build failed!
    exit /b 1
)
echo Build successful!
exit /b 0
