@echo off
setlocal enabledelayedexpansion

set BINARY_NAME=volnixd.exe
set VERSION=0.1.0-alpha

if "%1"=="" goto help
if "%1"=="help" goto help
if "%1"=="build" goto build
if "%1"=="test" goto test
if "%1"=="clean" goto clean
if "%1"=="init" goto init
if "%1"=="start" goto start
if "%1"=="status" goto status
if "%1"=="version" goto version
if "%1"=="info" goto info

echo ❌ Unknown command: %1
echo Use 'build.bat help' to see available commands
goto end

:help
echo 🚀 Volnix Protocol - Build Commands
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo Build Commands:
echo   build          Build the volnixd binary
echo   test           Run all tests
echo   clean          Clean build artifacts
echo.
echo Node Commands:
echo   init           Initialize a new node
echo   start          Start the node
echo   status         Show node status
echo   version        Show version information
echo.
echo Development:
echo   info           Show project information
echo.
echo Examples:
echo   build.bat build
echo   build.bat test
echo   build.bat init
goto end

:build
echo 🔨 Building Volnix Protocol...
go build -o %BINARY_NAME% ./cmd/volnixd
if %errorlevel% equ 0 (
    echo ✅ Build completed: %BINARY_NAME%
) else (
    echo ❌ Build failed
    exit /b 1
)
goto end

:test
echo 🧪 Running tests...
go test ./... -v
goto end

:clean
echo 🧹 Cleaning build artifacts...
if exist %BINARY_NAME% del %BINARY_NAME%
if exist volnixd-linux del volnixd-linux
if exist volnixd-darwin del volnixd-darwin
if exist coverage.out del coverage.out
if exist coverage.html del coverage.html
echo ✅ Clean completed
goto end

:init
call :build
echo 🚀 Initializing Volnix node...
%BINARY_NAME% init testnode
goto end

:start
call :build
echo 🚀 Starting Volnix node...
%BINARY_NAME% start
goto end

:status
call :build
echo 📊 Checking node status...
%BINARY_NAME% status
goto end

:version
call :build
%BINARY_NAME% version
goto end

:info
echo 🚀 Volnix Protocol
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo Version: %VERSION%
echo Build Target: %BINARY_NAME%
echo.
echo 🏗️  Architecture:
echo   • Cosmos SDK v0.53.x
echo   • CometBFT v0.38.x
echo   • GoLevelDB storage
echo.
echo 📦 Modules:
echo   • ident - Identity ^& ZKP verification
echo   • lizenz - LZN license management
echo   • anteil - ANT internal market
echo   • consensus - PoVB consensus
echo.
echo 🌟 Features:
echo   • Hybrid PoVB Consensus
echo   • ZKP Identity Verification
echo   • Three-tier Economy (WRT/LZN/ANT)
echo   • High Performance (10,000+ TPS)
goto end

:end