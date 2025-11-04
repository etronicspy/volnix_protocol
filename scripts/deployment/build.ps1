# Volnix Protocol Build Script for Windows PowerShell

param(
    [Parameter(Position=0)]
    [string]$Command = "help",
    [string]$Name = "mykey"
)

$BinaryName = "volnixd.exe"
$Version = "0.1.0-alpha"

function Show-Help {
    Write-Host "🚀 Volnix Protocol - Build Commands" -ForegroundColor Cyan
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Build Commands:" -ForegroundColor Yellow
    Write-Host "  build          Build the volnixd binary" -ForegroundColor White
    Write-Host "  build-all      Build for all platforms" -ForegroundColor White
    Write-Host "  clean          Clean build artifacts" -ForegroundColor White
    Write-Host ""
    Write-Host "Test Commands:" -ForegroundColor Yellow
    Write-Host "  test           Run all tests" -ForegroundColor White
    Write-Host "  test-unit      Run unit tests only" -ForegroundColor White
    Write-Host "  test-coverage  Run tests with coverage" -ForegroundColor White
    Write-Host ""
    Write-Host "Node Commands:" -ForegroundColor Yellow
    Write-Host "  init           Initialize a new node" -ForegroundColor White
    Write-Host "  start          Start the node" -ForegroundColor White
    Write-Host "  status         Show node status" -ForegroundColor White
    Write-Host "  version        Show version information" -ForegroundColor White
    Write-Host ""
    Write-Host "Key Commands:" -ForegroundColor Yellow
    Write-Host "  keys-add       Add a new key (use -Name parameter)" -ForegroundColor White
    Write-Host "  keys-list      List all keys" -ForegroundColor White
    Write-Host ""
    Write-Host "Development:" -ForegroundColor Yellow
    Write-Host "  deps           Download dependencies" -ForegroundColor White
    Write-Host "  fmt            Format Go code" -ForegroundColor White
    Write-Host "  info           Show project information" -ForegroundColor White
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Green
    Write-Host "  .\build.ps1 build" -ForegroundColor Gray
    Write-Host "  .\build.ps1 test" -ForegroundColor Gray
    Write-Host "  .\build.ps1 keys-add -Name mykey" -ForegroundColor Gray
}

function Build-Binary {
    Write-Host "🔨 Building Volnix Protocol..." -ForegroundColor Green
    go build -o $BinaryName ./cmd/volnixd
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Build completed: $BinaryName" -ForegroundColor Green
    } else {
        Write-Host "❌ Build failed" -ForegroundColor Red
        exit 1
    }
}

function Build-All {
    Write-Host "🔨 Building for all platforms..." -ForegroundColor Green
    
    # Windows
    $env:GOOS = "windows"; $env:GOARCH = "amd64"
    go build -o "volnixd.exe" ./cmd/volnixd
    
    # Linux
    $env:GOOS = "linux"; $env:GOARCH = "amd64"
    go build -o "volnixd-linux" ./cmd/volnixd
    
    # macOS
    $env:GOOS = "darwin"; $env:GOARCH = "amd64"
    go build -o "volnixd-darwin" ./cmd/volnixd
    
    # Reset environment
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    
    Write-Host "🎉 All platform builds completed!" -ForegroundColor Green
}

function Run-Tests {
    Write-Host "🧪 Running tests..." -ForegroundColor Blue
    go test ./... -v
}

function Run-UnitTests {
    Write-Host "🧪 Running unit tests..." -ForegroundColor Blue
    go test ./x/*/keeper -v
    go test ./x/*/types -v
}

function Run-Coverage {
    Write-Host "🧪 Running tests with coverage..." -ForegroundColor Blue
    go test ./... -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    Write-Host "✅ Coverage report generated: coverage.html" -ForegroundColor Green
}

function Clean-Build {
    Write-Host "🧹 Cleaning build artifacts..." -ForegroundColor Yellow
    Remove-Item -Path "volnixd.exe", "volnixd-linux", "volnixd-darwin", "coverage.out", "coverage.html" -ErrorAction SilentlyContinue
    Write-Host "✅ Clean completed" -ForegroundColor Green
}

function Update-Dependencies {
    Write-Host "📦 Managing dependencies..." -ForegroundColor Blue
    go mod download
    go mod tidy
    Write-Host "✅ Dependencies updated" -ForegroundColor Green
}

function Format-Code {
    Write-Host "🎨 Formatting code..." -ForegroundColor Blue
    go fmt ./...
    Write-Host "✅ Code formatted" -ForegroundColor Green
}

function Initialize-Node {
    Build-Binary
    Write-Host "🚀 Initializing Volnix node..." -ForegroundColor Magenta
    & ".\$BinaryName" init testnode
}

function Start-Node {
    Build-Binary
    Write-Host "🚀 Starting Volnix node..." -ForegroundColor Magenta
    & ".\$BinaryName" start
}

function Show-Status {
    Build-Binary
    Write-Host "📊 Checking node status..." -ForegroundColor Magenta
    & ".\$BinaryName" status
}

function Show-Version {
    Build-Binary
    & ".\$BinaryName" version
}

function Add-Key {
    Build-Binary
    Write-Host "🔑 Adding new key: $Name" -ForegroundColor Magenta
    & ".\$BinaryName" keys add $Name
}

function List-Keys {
    Build-Binary
    Write-Host "🔑 Listing keys..." -ForegroundColor Magenta
    & ".\$BinaryName" keys list
}

function Show-Info {
    Write-Host "🚀 Volnix Protocol" -ForegroundColor Cyan
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    Write-Host "Version: $Version" -ForegroundColor Yellow
    Write-Host "Go Version: $(go version)" -ForegroundColor Yellow
    Write-Host "Build Target: $BinaryName" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "🏗️  Architecture:" -ForegroundColor Blue
    Write-Host "  • Cosmos SDK v0.53.x"
    Write-Host "  • CometBFT v0.38.x"
    Write-Host "  • GoLevelDB storage"
    Write-Host ""
    Write-Host "📦 Modules:" -ForegroundColor Blue
    Write-Host "  • ident - Identity & ZKP verification"
    Write-Host "  • lizenz - LZN license management"
    Write-Host "  • anteil - ANT internal market"
    Write-Host "  • consensus - PoVB consensus"
    Write-Host ""
    Write-Host "🌟 Features:" -ForegroundColor Blue
    Write-Host "  • Hybrid PoVB Consensus"
    Write-Host "  • ZKP Identity Verification"
    Write-Host "  • Three-tier Economy (WRT/LZN/ANT)"
    Write-Host "  • High Performance (10,000+ TPS)"
}

# Main command dispatcher
switch ($Command.ToLower()) {
    "help" { Show-Help }
    "build" { Build-Binary }
    "build-all" { Build-All }
    "test" { Run-Tests }
    "test-unit" { Run-UnitTests }
    "test-coverage" { Run-Coverage }
    "clean" { Clean-Build }
    "deps" { Update-Dependencies }
    "fmt" { Format-Code }
    "init" { Initialize-Node }
    "start" { Start-Node }
    "status" { Show-Status }
    "version" { Show-Version }
    "keys-add" { Add-Key }
    "keys-list" { List-Keys }
    "info" { Show-Info }
    default {
        Write-Host "❌ Unknown command: $Command" -ForegroundColor Red
        Write-Host "Use '.\build.ps1 help' to see available commands" -ForegroundColor Yellow
    }
}