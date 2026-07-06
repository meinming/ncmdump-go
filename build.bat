@echo off

SETLOCAL ENABLEDELAYEDEXPANSION

SET CGO_ENABLED=0
SET GO111MODULE=on
SET GOPROXY=https://goproxy.io
SET NAME=ncmdump

set "RAW_DATE=%date%"
set "TODAY="
for /L %%i in (0,1,20) do (
    set "CHAR=!RAW_DATE:~%%i,1!"
    if "!CHAR!" geq "0" if "!CHAR!" leq "9" set "TODAY=!TODAY!!CHAR!"
)
set "TODAY=%TODAY:~0,8%"

set VERSION=%TODAY%
set LDFLAGS="-X main.VERSION=%VERSION% -w -s"

md .\dist 2>nul

echo [INFO] Starting build for version: %VERSION%

:: Windows amd64
set GOOS=windows
set GOARCH=amd64
go build -ldflags %LDFLAGS% -o ./dist/%NAME%-%GOOS%-%GOARCH%.exe main.go
.\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH%.exe >nul 2>nul

:: Windows i386
set GOOS=windows
set GOARCH=386
go build -ldflags %LDFLAGS% -o ./dist/%NAME%-%GOOS%-%GOARCH%.exe main.go
.\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH%.exe >nul 2>nul

:: macOS amd64
set GOOS=darwin
set GOARCH=amd64
go build -ldflags %LDFLAGS% -o ./dist/%NAME%-%GOOS%-%GOARCH% main.go
.\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH% >nul 2>nul

:: Linux amd64
set GOOS=linux
set GOARCH=amd64
go build -ldflags %LDFLAGS% -o ./dist/%NAME%-%GOOS%-%GOARCH% main.go
.\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH% >nul 2>nul

:: Linux i386 
set GOOS=linux
set GOARCH=386
go build -ldflags %LDFLAGS% -o ./dist/%NAME%-%GOOS%-%GOARCH% main.go
.\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH% >nul 2>nul

:: Linux arm64
set GOOS=linux
set GOARCH=arm64
go build -ldflags %LDFLAGS% -o ./dist/%NAME%-%GOOS%-%GOARCH% main.go
.\upx-5.2.0-win64\upx.exe ./dist/%NAME%-%GOOS%-%GOARCH% >nul 2>nul

echo [INFO] Packaging files into ZIP...
cd .\dist
        
powershell -NoProfile -Command "Compress-Archive -Path '%NAME%-windows-amd64.exe' -DestinationPath '%NAME%-windows-amd64-%VERSION%.zip' -Force" 2>nul || tar -a -c -f %NAME%-windows-amd64-%VERSION%.zip %NAME%-windows-amd64.exe 2>nul
powershell -NoProfile -Command "Compress-Archive -Path '%NAME%-windows-386.exe' -DestinationPath '%NAME%-windows-386-%VERSION%.zip' -Force" 2>nul || tar -a -c -f %NAME%-windows-386-%VERSION%.zip %NAME%-windows-386.exe 2>nul
powershell -NoProfile -Command "Compress-Archive -Path '%NAME%-darwin-amd64' -DestinationPath '%NAME%-darwin-amd64-%VERSION%.zip' -Force" 2>nul || tar -a -c -f %NAME%-darwin-amd64-%VERSION%.zip %NAME%-darwin-amd64 2>nul
powershell -NoProfile -Command "Compress-Archive -Path '%NAME%-linux-amd64' -DestinationPath '%NAME%-linux-amd64-%VERSION%.zip' -Force" 2>nul || tar -a -c -f %NAME%-linux-amd64-%VERSION%.zip %NAME%-linux-amd64 2>nul
powershell -NoProfile -Command "Compress-Archive -Path '%NAME%-linux-386' -DestinationPath '%NAME%-linux-386-%VERSION%.zip' -Force" 2>nul || tar -a -c -f %NAME%-linux-386-%VERSION%.zip %NAME%-linux-386 2>nul
powershell -NoProfile -Command "Compress-Archive -Path '%NAME%-linux-arm64' -DestinationPath '%NAME%-linux-arm64-%VERSION%.zip' -Force" 2>nul || tar -a -c -f %NAME%-linux-arm64-%VERSION%.zip %NAME%-linux-arm64 2>nul

:: 打包完成，切回原项目根目录
cd ..

echo [SUCCESS] All builds and packaging completed successfully in .\dist\
ENDLOCAL