@echo off
SETLOCAL ENABLEDELAYEDEXPANSION

SET CGO_ENABLED=0
SET GO111MODULE=on
SET GOPROXY=https://goproxy.io
SET NAME=ncmdump

:: 2. 提取当前日期（过滤掉斜杠和星期，完美提取 8 位数字）
set "RAW_DATE=%date%"
set "TODAY="
for /L %%i in (0,1,20) do (
    set "CHAR=!RAW_DATE:~%%i,1!"
    if "!CHAR!" geq "0" if "!CHAR!" leq "9" set "TODAY=!TODAY!!CHAR!"
)
set "TODAY=%TODAY:~0,8%"


set "VERSION=v1.0.0_!TODAY!"
set "LDFLAGS=-X cmd.VERSION=!VERSION! -w -s"

:: 创建输出目录
md .\dist 2>nul

echo [INFO] 开始跨平台编译与编译时优化，目标版本: %VERSION%


:: Windows amd64
set GOOS=windows& set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o ./dist/%NAME%-%GOOS%-%GOARCH%.exe main.go
if exist .\upx-5.2.0-win64\upx.exe .\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH%.exe >nul 2>nul

:: Windows i386
set GOOS=windows& set GOARCH=386
go build -ldflags "%LDFLAGS%" -o ./dist/%NAME%-%GOOS%-%GOARCH%.exe main.go
if exist .\upx-5.2.0-win64\upx.exe .\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH%.exe >nul 2>nul

:: macOS amd64
set GOOS=darwin& set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o ./dist/%NAME%-%GOOS%-%GOARCH% main.go
if exist .\upx-5.2.0-win64\upx.exe .\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH% >nul 2>nul

:: Linux amd64
set GOOS=linux& set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o ./dist/%NAME%-%GOOS%-%GOARCH% main.go
if exist .\upx-5.2.0-win64\upx.exe .\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH% >nul 2>nul

:: Linux i386
set GOOS=linux& set GOARCH=386
go build -ldflags "%LDFLAGS%" -o ./dist/%NAME%-%GOOS%-%GOARCH% main.go
if exist .\upx-5.2.0-win64\upx.exe .\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH% >nul 2>nul

:: Linux arm64
set GOOS=linux& set GOARCH=arm64
go build -ldflags "%LDFLAGS%" -o ./dist/%NAME%-%GOOS%-%GOARCH% main.go
if exist .\upx-5.2.0-win64\upx.exe .\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH% >nul 2>nul


echo [INFO] 正在将二进制文件归档打包为 ZIP...

powershell -NoProfile -Command "Compress-Archive -Path './dist/%NAME%-windows-amd64.exe' -DestinationPath './dist/%NAME%-windows-amd64-%VERSION%.zip' -Force" 2>nul
powershell -NoProfile -Command "Compress-Archive -Path './dist/%NAME%-windows-386.exe' -DestinationPath './dist/%NAME%-windows-386-%VERSION%.zip' -Force" 2>nul
powershell -NoProfile -Command "Compress-Archive -Path './dist/%NAME%-darwin-amd64' -DestinationPath './dist/%NAME%-darwin-amd64-%VERSION%.zip' -Force" 2>nul
powershell -NoProfile -Command "Compress-Archive -Path './dist/%NAME%-linux-amd64' -DestinationPath './dist/%NAME%-linux-amd64-%VERSION%.zip' -Force" 2>nul
powershell -NoProfile -Command "Compress-Archive -Path './dist/%NAME%-linux-386' -DestinationPath './dist/%NAME%-linux-386-%VERSION%.zip' -Force" 2>nul
powershell -NoProfile -Command "Compress-Archive -Path './dist/%NAME%-linux-arm64' -DestinationPath './dist/%NAME%-linux-arm64-%VERSION%.zip' -Force" 2>nul

echo [SUCCESS] 多平台发布包已成功构建并打包至 .\dist\ 目录下！
ENDLOCAL