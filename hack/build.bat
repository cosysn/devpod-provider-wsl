@echo off
setlocal enabledelayedexpansion

REM Build script for devpod-provider-wsl
REM Usage: .\hack\build.bat [version]
REM Example: .\hack\build.bat v0.0.1

if "%1"=="" (
    set VERSION=v0.0.1
) else (
    set VERSION=%1
)

REM Get script directory
set SCRIPT_DIR=%~dp0
set ROOT_DIR=%SCRIPT_DIR%..

echo Building devpod-provider-wsl %VERSION%...
echo.

REM Create release directory
if not exist "%ROOT_DIR%\release" mkdir "%ROOT_DIR%\release"

set LDFLAGS=-s -w -X main.version=%VERSION%

REM Step 1: Build Linux agent
echo [1/3] Building Linux agent...
set GOOS=linux
set GOARCH=amd64
cd /d "%ROOT_DIR%"
go build -ldflags="%LDFLAGS%" -o pkg\agent\agent-linux .

if errorlevel 1 (
    echo Linux agent build failed!
    exit /b 1
)

REM Step 2: Build Windows provider (with embed tag)
echo [2/3] Building Windows provider...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -tags=embed -o "release\devpod-provider-wsl-amd64.exe" .

if errorlevel 1 (
    echo Windows provider build failed!
    exit /b 1
)

REM Cleanup
del pkg\agent\agent-linux

REM Step 3: Generate provider.yaml
echo [3/3] Generating provider.yaml...
cd /d "%ROOT_DIR%"
go run ./hack/provider/main.go %VERSION% > provider.yaml

REM Generate SHA256 checksum
echo Generating SHA256 checksum...
cd /d "%ROOT_DIR%\release"
for /f "skip=1 tokens=*" %%a in ('certutil -hashfile devpod-provider-wsl-amd64.exe SHA256') do (
    echo %%a > devpod-provider-wsl-amd64.exe.sha256
    goto checksum_done
)
:checksum_done

REM Create zip package
echo Packaging...
del devpod-provider-wsl-%VERSION%.zip 2>nul
powershell -Command "Compress-Archive -Path 'devpod-provider-wsl-amd64.exe', 'devpod-provider-wsl-amd64.exe.sha256' -DestinationPath 'devpod-provider-wsl-%VERSION%.zip' -Force"

echo.
echo ========================================
echo Build complete!
echo ========================================
echo.
echo Binary: release/devpod-provider-wsl-amd64.exe
echo SHA256: release/devpod-provider-wsl-amd64.exe.sha256
echo Zip:    release/devpod-provider-wsl-%VERSION%.zip
