@echo off
CHCP 65001 && CLS
setlocal enabledelayedexpansion

set VERSION=1.0.1

if not exist "build" mkdir build

set PLATFORMS=windows_amd64 windows_arm64 linux_amd64 linux_arm64 darwin_amd64 darwin_arm64

for %%p in (%PLATFORMS%) do (
    echo Building for %%p...

    for /f "tokens=1,2 delims=_" %%a in ("%%p") do (
        set OS=%%a
        set ARCH=%%b

        set OUTFILE=lgxt_%VERSION%_%%p

        if "!OS!"=="windows" (
            set OUTFILE=!OUTFILE!.exe
        )

        set GOOS=!OS!
        set GOARCH=!ARCH!

        go build -o build/!OUTFILE! src/main.go

        if !errorlevel! equ 0 (
            echo Successfully built build/!OUTFILE!
        ) else (
            echo Failed to build for %%p
        )
    )
)

echo All builds completed!
pause
