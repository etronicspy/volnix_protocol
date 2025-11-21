# Volnix Protocol Quick Start (Windows)
# Объединенный скрипт для быстрого запуска всех компонентов

param(
    [switch]$SkipBuild,
    [switch]$CleanStart,
    [string]$ChainId = "volnix-testnet",
    [string]$Moniker = "volnix-node"
)

# Цвета для вывода
$Green = "Green"
$Yellow = "Yellow"
$Red = "Red"
$Cyan = "Cyan"

Write-Host "🚀 Volnix Protocol Quick Start" -ForegroundColor $Cyan
Write-Host "==============================" -ForegroundColor $Cyan
Write-Host ""

# Функция для проверки зависимостей
function Test-Dependencies {
    Write-Host "🔍 Checking dependencies..." -ForegroundColor $Yellow
    
    try {
        $goVersion = go version
        Write-Host "✅ Go: $goVersion" -ForegroundColor $Green
    } catch {
        Write-Host "❌ Go not found. Please install Go 1.21+" -ForegroundColor $Red
        exit 1
    }
    
    try {
        $nodeVersion = node --version
        Write-Host "✅ Node.js: $nodeVersion" -ForegroundColor $Green
    } catch {
        Write-Host "⚠️  Node.js not found (optional for blockchain node only)" -ForegroundColor $Yellow
    }
    
    Write-Host ""
}

# Функция для сборки проекта
function Build-Project {
    if (-not $SkipBuild) {
        Write-Host "🔨 Building project..." -ForegroundColor $Yellow
        
        New-Item -ItemType Directory -Force -Path "build" | Out-Null
        go build -o build/volnixd-standalone.exe ./cmd/volnixd-standalone
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Failed to build" -ForegroundColor $Red
            exit 1
        }
        Write-Host "✅ Build completed" -ForegroundColor $Green
        Write-Host ""
    }
}

# Функция для инициализации узла
function Initialize-Node {
    if ($CleanStart -and (Test-Path ".volnix")) {
        Write-Host "🧹 Cleaning existing node data..." -ForegroundColor $Yellow
        Remove-Item -Recurse -Force ".volnix"
    }
    
    if (-not (Test-Path ".volnix")) {
        Write-Host "🏗️ Initializing node: $Moniker" -ForegroundColor $Yellow
        .\build\volnixd-standalone.exe init $Moniker
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Failed to initialize node" -ForegroundColor $Red
            exit 1
        }
        Write-Host "✅ Node initialized" -ForegroundColor $Green
    } else {
        Write-Host "✅ Using existing node configuration" -ForegroundColor $Green
    }
    Write-Host ""
}

# Функция для запуска блокчейн узла
function Start-BlockchainNode {
    Write-Host "🌐 Starting blockchain node..." -ForegroundColor $Yellow
    $nodeProcess = Start-Process -FilePath ".\build\volnixd-standalone.exe" -ArgumentList "start" -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 5
    Write-Host "✅ Blockchain node started (PID: $($nodeProcess.Id))" -ForegroundColor $Green
    Write-Host "🔗 RPC: http://localhost:26657" -ForegroundColor $Cyan
    return $nodeProcess
}

# Функция для запуска Wallet UI (опционально)
function Start-WalletUI {
    if (Test-Path "frontend/wallet-ui") {
        Write-Host "💰 Starting Wallet UI..." -ForegroundColor $Yellow
        Push-Location "frontend/wallet-ui"
        try {
            if (-not (Test-Path "node_modules")) {
                npm install
            }
            $walletProcess = Start-Process -FilePath "npm" -ArgumentList "start" -PassThru -WindowStyle Hidden
            Write-Host "✅ Wallet UI started (PID: $($walletProcess.Id))" -ForegroundColor $Green
            Write-Host "🌐 Wallet UI: http://localhost:3000" -ForegroundColor $Cyan
            return $walletProcess
        } finally {
            Pop-Location
        }
    }
}

# Основная функция
function Main {
    try {
        Test-Dependencies
        Build-Project
        Initialize-Node
        
        Write-Host "🚀 Starting services..." -ForegroundColor $Cyan
        Write-Host ""
        
        $nodeProcess = Start-BlockchainNode
        $walletProcess = Start-WalletUI
        
        Write-Host ""
        Write-Host "🎉 Volnix Protocol is running!" -ForegroundColor $Green
        Write-Host "==============================" -ForegroundColor $Green
        Write-Host ""
        Write-Host "📊 Available Services:" -ForegroundColor $Cyan
        Write-Host "  🌐 Blockchain Node: http://localhost:26657" -ForegroundColor $Green
        if ($walletProcess) {
            Write-Host "  💰 Wallet UI:       http://localhost:3000" -ForegroundColor $Green
        }
        Write-Host ""
        Write-Host "Press Ctrl+C to stop..." -ForegroundColor $Yellow
        
        # Ожидание
        try {
            while ($true) {
                Start-Sleep -Seconds 1
            }
        } catch {
            Write-Host "`n🛑 Shutting down..." -ForegroundColor $Yellow
            if ($nodeProcess) { Stop-Process -Id $nodeProcess.Id -Force -ErrorAction SilentlyContinue }
            if ($walletProcess) { Stop-Process -Id $walletProcess.Id -Force -ErrorAction SilentlyContinue }
        }
    } catch {
        Write-Host "❌ Error: $($_.Exception.Message)" -ForegroundColor $Red
        exit 1
    }
}

Main