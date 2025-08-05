@echo off
REM AI Prompt Proxy 多平台编译脚本 (Windows版本)
REM 支持 Windows、Linux、macOS 平台的 amd64 和 arm64 架构

setlocal enabledelayedexpansion

REM 项目信息
set APP_NAME=ai-prompt-proxy
if "%VERSION%"=="" set VERSION=1.0.0

REM 获取构建时间
for /f "tokens=1-4 delims=/ " %%i in ('date /t') do set BUILD_DATE=%%i-%%j-%%k
for /f "tokens=1-2 delims=: " %%i in ('time /t') do set BUILD_TIME=%%i:%%j
set BUILD_TIME=%BUILD_DATE%_%BUILD_TIME%

REM 获取Git提交（如果可用）
git rev-parse --short HEAD >nul 2>&1
if %errorlevel%==0 (
    for /f %%i in ('git rev-parse --short HEAD') do set GIT_COMMIT=%%i
) else (
    set GIT_COMMIT=unknown
)

REM 构建信息
set LDFLAGS=-s -w -X main.Version=%VERSION% -X main.BuildTime=%BUILD_TIME% -X main.GitCommit=%GIT_COMMIT%

REM 输出目录
set BUILD_DIR=build
set DIST_DIR=dist

echo 🧹 清理旧的构建文件...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
if exist %DIST_DIR% rmdir /s /q %DIST_DIR%
mkdir %BUILD_DIR%
mkdir %DIST_DIR%

echo 🚀 开始编译 AI Prompt Proxy v%VERSION%
echo 📅 构建时间: %BUILD_TIME%
echo 🔗 Git提交: %GIT_COMMIT%
echo.

REM 检查 Go 环境
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go 未安装或不在 PATH 中
    exit /b 1
)

for /f "tokens=*" %%i in ('go version') do echo 🔍 Go 版本: %%i
echo.

echo 📦 下载依赖...
go mod download
go mod tidy
echo.

REM 支持的平台和架构
set PLATFORMS=windows/amd64 windows/arm64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
set SUCCESS_COUNT=0
set TOTAL_COUNT=6

for %%p in (%PLATFORMS%) do (
    call :build_binary %%p
    echo.
)

REM 创建 README
call :create_readme

echo 🎉 编译完成！
echo 📊 成功: %SUCCESS_COUNT%/%TOTAL_COUNT%
echo.
echo 📁 构建文件位于: %BUILD_DIR%\
echo 📦 发布包位于: %DIST_DIR%\
echo.

if exist %DIST_DIR% (
    echo 📋 发布包列表:
    dir /b %DIST_DIR%
)

goto :eof

:build_binary
set PLATFORM=%1
for /f "tokens=1 delims=/" %%a in ("%PLATFORM%") do set OS=%%a
for /f "tokens=2 delims=/" %%a in ("%PLATFORM%") do set ARCH=%%a

set OUTPUT_NAME=%APP_NAME%
if "%OS%"=="windows" set OUTPUT_NAME=%APP_NAME%.exe

set OUTPUT_PATH=%BUILD_DIR%\%APP_NAME%-%OS%-%ARCH%
mkdir %OUTPUT_PATH%

echo 🔨 编译 %OS%/%ARCH%...

REM 设置环境变量并编译
set GOOS=%OS%
set GOARCH=%ARCH%
set CGO_ENABLED=0

go build -ldflags="%LDFLAGS%" -o "%OUTPUT_PATH%\%OUTPUT_NAME%" .

if %errorlevel%==0 (
    echo ✅ %OS%/%ARCH% 编译成功
    set /a SUCCESS_COUNT+=1
    
    REM 创建配置目录
    mkdir "%OUTPUT_PATH%\configs" >nul 2>&1
    
    REM 创建启动脚本
    call :create_startup_scripts "%OUTPUT_PATH%" "%OS%" "%OUTPUT_NAME%"
    
    REM 创建压缩包
    call :create_archive "%OUTPUT_PATH%" "%OS%" "%ARCH%"
) else (
    echo ❌ %OS%/%ARCH% 编译失败
)
goto :eof

:create_startup_scripts
set OUTPUT_PATH=%~1
set OS=%~2
set BINARY_NAME=%~3

if "%OS%"=="windows" (
    REM Windows 批处理文件
    echo @echo off > "%OUTPUT_PATH%\start.bat"
    echo echo Starting AI Prompt Proxy... >> "%OUTPUT_PATH%\start.bat"
    echo echo Web interface will be available at: http://localhost:8081 >> "%OUTPUT_PATH%\start.bat"
    echo echo Press Ctrl+C to stop the server >> "%OUTPUT_PATH%\start.bat"
    echo %APP_NAME%.exe >> "%OUTPUT_PATH%\start.bat"
    echo pause >> "%OUTPUT_PATH%\start.bat"
    
    REM Windows PowerShell 脚本
    echo Write-Host "Starting AI Prompt Proxy..." -ForegroundColor Green > "%OUTPUT_PATH%\start.ps1"
    echo Write-Host "Web interface will be available at: http://localhost:8081" -ForegroundColor Yellow >> "%OUTPUT_PATH%\start.ps1"
    echo Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow >> "%OUTPUT_PATH%\start.ps1"
    echo ^& ".\%APP_NAME%.exe" >> "%OUTPUT_PATH%\start.ps1"
) else (
    REM Unix shell 脚本
    echo #!/bin/bash > "%OUTPUT_PATH%\start.sh"
    echo echo "Starting AI Prompt Proxy..." >> "%OUTPUT_PATH%\start.sh"
    echo echo "Web interface will be available at: http://localhost:8081" >> "%OUTPUT_PATH%\start.sh"
    echo echo "Press Ctrl+C to stop the server" >> "%OUTPUT_PATH%\start.sh"
    echo ./%BINARY_NAME% >> "%OUTPUT_PATH%\start.sh"
)
goto :eof

:create_archive
set OUTPUT_PATH=%~1
set OS=%~2
set ARCH=%~3

set ARCHIVE_NAME=%APP_NAME%-%VERSION%-%OS%-%ARCH%

echo 📦 创建压缩包 %ARCHIVE_NAME%...

cd %BUILD_DIR%

if "%OS%"=="windows" (
    REM Windows 使用内置压缩（需要PowerShell）
    powershell -command "Compress-Archive -Path '%APP_NAME%-%OS%-%ARCH%' -DestinationPath '..\%DIST_DIR%\%ARCHIVE_NAME%.zip'" >nul 2>&1
    if %errorlevel% neq 0 (
        echo ⚠️  PowerShell 压缩失败，跳过 Windows 压缩包创建
    )
) else (
    REM 尝试使用 tar（Git Bash 或 WSL）
    tar -czf "..\%DIST_DIR%\%ARCHIVE_NAME%.tar.gz" "%APP_NAME%-%OS%-%ARCH%" >nul 2>&1
    if %errorlevel% neq 0 (
        echo ⚠️  tar 命令不可用，跳过 Unix 压缩包创建
    )
)

cd ..
goto :eof

:create_readme
echo # AI Prompt Proxy v%VERSION% > "%DIST_DIR%\README.md"
echo. >> "%DIST_DIR%\README.md"
echo ## 快速开始 >> "%DIST_DIR%\README.md"
echo. >> "%DIST_DIR%\README.md"
echo ### Windows >> "%DIST_DIR%\README.md"
echo 1. 解压 `ai-prompt-proxy-%VERSION%-windows-amd64.zip` 或 `ai-prompt-proxy-%VERSION%-windows-arm64.zip` >> "%DIST_DIR%\README.md"
echo 2. 双击 `start.bat` 或运行 `start.ps1` >> "%DIST_DIR%\README.md"
echo 3. 打开浏览器访问 http://localhost:8081 >> "%DIST_DIR%\README.md"
echo. >> "%DIST_DIR%\README.md"
echo ### Linux >> "%DIST_DIR%\README.md"
echo 1. 解压 `ai-prompt-proxy-%VERSION%-linux-amd64.tar.gz` 或 `ai-prompt-proxy-%VERSION%-linux-arm64.tar.gz` >> "%DIST_DIR%\README.md"
echo 2. 运行 `./start.sh` 或直接运行 `./ai-prompt-proxy` >> "%DIST_DIR%\README.md"
echo 3. 打开浏览器访问 http://localhost:8081 >> "%DIST_DIR%\README.md"
echo. >> "%DIST_DIR%\README.md"
echo ### macOS >> "%DIST_DIR%\README.md"
echo 1. 解压 `ai-prompt-proxy-%VERSION%-darwin-amd64.tar.gz` 或 `ai-prompt-proxy-%VERSION%-darwin-arm64.tar.gz` >> "%DIST_DIR%\README.md"
echo 2. 运行 `./start.sh` 或直接运行 `./ai-prompt-proxy` >> "%DIST_DIR%\README.md"
echo 3. 打开浏览器访问 http://localhost:8081 >> "%DIST_DIR%\README.md"
echo. >> "%DIST_DIR%\README.md"
echo ## 构建信息 >> "%DIST_DIR%\README.md"
echo - 版本: %VERSION% >> "%DIST_DIR%\README.md"
echo - 构建时间: %BUILD_TIME% >> "%DIST_DIR%\README.md"
echo - Git提交: %GIT_COMMIT% >> "%DIST_DIR%\README.md"
goto :eof