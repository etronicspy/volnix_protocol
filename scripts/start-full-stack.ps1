# Volnix Protocol Full Stack Startup Script
# Запускает все компоненты системы: блокчейн узел, wallet UI, blockchain explorer

param(
    [switch]$SkipBuild,
    [switch]$CleanStart,
    [string]$ChainId = "volnix-testnet",
    [string]$Moniker = "volnix-node-1"
)

# Цвета для вывода
$Green = "Green"
$Yellow = "Yellow"
$Red = "Red"
$Cyan = "Cyan"
$Magenta = "Magenta"

Write-Host "🚀 Volnix Protocol Full Stack Startup" -ForegroundColor $Cyan
Write-Host "=======================================" -ForegroundColor $Cyan
Write-Host ""

# Функция для проверки зависимостей
function Test-Dependencies {
    Write-Host "🔍 Checking dependencies..." -ForegroundColor $Yellow
    
    # Проверка Go
    try {
        $goVersion = go version
        Write-Host "✅ Go: $goVersion" -ForegroundColor $Green
    } catch {
        Write-Host "❌ Go not found. Please install Go 1.21+" -ForegroundColor $Red
        exit 1
    }
    
    # Проверка Node.js
    try {
        $nodeVersion = node --version
        Write-Host "✅ Node.js: $nodeVersion" -ForegroundColor $Green
    } catch {
        Write-Host "❌ Node.js not found. Please install Node.js 18+" -ForegroundColor $Red
        exit 1
    }
    
    # Проверка npm
    try {
        $npmVersion = npm --version
        Write-Host "✅ npm: $npmVersion" -ForegroundColor $Green
    } catch {
        Write-Host "❌ npm not found. Please install npm" -ForegroundColor $Red
        exit 1
    }
    
    Write-Host ""
}

# Функция для сборки проекта
function Build-Project {
    if (-not $SkipBuild) {
        Write-Host "🔨 Building Volnix Protocol..." -ForegroundColor $Yellow
        
        # Сборка основного бинарника
        Write-Host "Building volnixd binary..." -ForegroundColor $Yellow
        go build -o build/volnixd.exe ./cmd/volnixd
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Failed to build volnixd" -ForegroundColor $Red
            exit 1
        }
        Write-Host "✅ volnixd built successfully" -ForegroundColor $Green
        
        # Сборка standalone версии
        Write-Host "Building volnixd-standalone binary..." -ForegroundColor $Yellow
        New-Item -ItemType Directory -Force -Path "build" | Out-Null
        go build -o build/volnixd-standalone.exe ./cmd/volnixd-standalone
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Failed to build volnixd-standalone" -ForegroundColor $Red
            exit 1
        }
        Write-Host "✅ volnixd-standalone built successfully" -ForegroundColor $Green
        
        Write-Host ""
    } else {
        Write-Host "⏭️ Skipping build (using existing binaries)" -ForegroundColor $Yellow
        Write-Host ""
    }
}

# Функция для инициализации узла
function Initialize-Node {
    Write-Host "🏗️ Initializing blockchain node..." -ForegroundColor $Yellow
    
    if ($CleanStart -and (Test-Path ".volnix")) {
        Write-Host "🧹 Cleaning existing node data..." -ForegroundColor $Yellow
        Remove-Item -Recurse -Force ".volnix"
    }
    
    if (-not (Test-Path ".volnix")) {
        Write-Host "Initializing new node: $Moniker" -ForegroundColor $Yellow
        .\build\volnixd.exe init $Moniker --chain-id $ChainId
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Failed to initialize node" -ForegroundColor $Red
            exit 1
        }
        Write-Host "✅ Node initialized successfully" -ForegroundColor $Green
    } else {
        Write-Host "✅ Using existing node configuration" -ForegroundColor $Green
    }
    
    Write-Host ""
}

# Функция для установки зависимостей wallet UI
function Install-WalletDependencies {
    Write-Host "📦 Installing Wallet UI dependencies..." -ForegroundColor $Yellow
    
    Push-Location "frontend/wallet-ui"
    try {
        if (-not (Test-Path "node_modules")) {
            npm install
            if ($LASTEXITCODE -ne 0) {
                Write-Host "❌ Failed to install wallet dependencies" -ForegroundColor $Red
                exit 1
            }
        }
        Write-Host "✅ Wallet UI dependencies ready" -ForegroundColor $Green
    } finally {
        Pop-Location
    }
    
    Write-Host ""
}
}

# Функция для запуска блокчейн узла
function Start-BlockchainNode {
    Write-Host "🌐 Starting blockchain node..." -ForegroundColor $Yellow
    
    # Запуск в фоновом режиме
    $nodeProcess = Start-Process -FilePath ".\build\volnixd.exe" -ArgumentList "start" -PassThru -WindowStyle Hidden
    
    # Ожидание запуска
    Start-Sleep -Seconds 5
    
    # Проверка статуса
    try {
        $status = .\build\volnixd.exe status 2>$null
        Write-Host "✅ Blockchain node started (PID: $($nodeProcess.Id))" -ForegroundColor $Green
        Write-Host "🔗 RPC endpoint: http://localhost:26657" -ForegroundColor $Cyan
        Write-Host "🌐 P2P endpoint: tcp://localhost:26656" -ForegroundColor $Cyan
    } catch {
        Write-Host "⚠️ Node starting... (may take a moment)" -ForegroundColor $Yellow
    }
    
    return $nodeProcess
}

# Функция для запуска Wallet UI
function Start-WalletUI {
    Write-Host "💰 Starting Wallet UI..." -ForegroundColor $Yellow
    
    Push-Location "frontend/wallet-ui"
    try {
        # Запуск в фоновом режиме
        $walletProcess = Start-Process -FilePath "npm" -ArgumentList "start" -PassThru -WindowStyle Hidden
        
        Write-Host "✅ Wallet UI started (PID: $($walletProcess.Id))" -ForegroundColor $Green
        Write-Host "🌐 Wallet UI: http://localhost:3000" -ForegroundColor $Cyan
        
        return $walletProcess
    } finally {
        Pop-Location
    }
}
}

# Функция для запуска Blockchain Explorer
function Start-BlockchainExplorer {
    Write-Host "🔍 Starting Blockchain Explorer..." -ForegroundColor $Yellow
    
    Push-Location "frontend/blockchain-explorer"
    try {
        # Запуск в фоновом режиме
        $explorerProcess = Start-Process -FilePath "powershell" -ArgumentList "-ExecutionPolicy Bypass -File start-explorer.ps1" -PassThru -WindowStyle Hidden
        
        Write-Host "✅ Blockchain Explorer started (PID: $($explorerProcess.Id))" -ForegroundColor $Green
        Write-Host "🌐 Explorer: http://localhost:8080" -ForegroundColor $Cyan
        
        return $explorerProcess
    } finally {
        Pop-Location
    }
}
}

# Функция для отображения статуса
function Show-Status {
    Write-Host ""
    Write-Host "🎉 Volnix Protocol Full Stack is Running!" -ForegroundColor $Green
    Write-Host "=========================================" -ForegroundColor $Green
    Write-Host ""
    Write-Host "📊 Services Status:" -ForegroundColor $Cyan
    Write-Host "  🌐 Blockchain Node: http://localhost:26657" -ForegroundColor $Green
    Write-Host "  💰 Wallet UI:       http://localhost:3000" -ForegroundColor $Green
    Write-Host "  🔍 Explorer:        http://localhost:8080" -ForegroundColor $Green
    Write-Host ""
    Write-Host "🔧 Available Commands:" -ForegroundColor $Cyan
    Write-Host "  .\build\volnixd.exe status                    # Check node status"
    Write-Host "  .\build\volnixd.exe keys list                 # List wallet keys"
    Write-Host "  .\build\volnixd.exe query bank balances <addr> # Check balance"
    Write-Host ""
    Write-Host "📚 Quick Start:" -ForegroundColor $Cyan
    Write-Host "  1. Open Wallet UI:    http://localhost:3000"
    Write-Host "  2. Create new wallet or connect existing"
    Write-Host "  3. View blockchain:   http://localhost:8080"
    Write-Host ""
    Write-Host "⚠️  Note: Identity validation is disabled for this demo" -ForegroundColor $Yellow
    Write-Host ""
}

# Функция для ожидания завершения
function Wait-ForExit {
    Write-Host "Press Ctrl+C to stop all services..." -ForegroundColor $Yellow
    Write-Host ""
    
    try {
        while ($true) {
            Start-Sleep -Seconds 1
        }
    } catch {
        Write-Host ""
        Write-Host "🛑 Shutting down services..." -ForegroundColor $Yellow
    }
}

# Функция для остановки всех процессов
function Stop-AllServices {
    Write-Host "🛑 Stopping all Volnix Protocol services..." -ForegroundColor $Yellow
    
    # Остановка по имени процесса
    Get-Process | Where-Object { $_.ProcessName -like "*volnixd*" -or $_.ProcessName -like "*node*" -or $_.ProcessName -like "*powershell*" } | ForEach-Object {
        try {
            Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
            Write-Host "✅ Stopped process: $($_.ProcessName) (PID: $($_.Id))" -ForegroundColor $Green
        } catch {
            # Игнорируем ошибки
        }
    }
    
    Write-Host "✅ All services stopped" -ForegroundColor $Green
}

# Основная функция
function Main {
    try {
        Test-Dependencies
        Build-Project
        Initialize-Node
        Install-WalletDependencies
        
        Write-Host "🚀 Starting all services..." -ForegroundColor $Cyan
        Write-Host ""
        
        # Запуск всех сервисов
        $nodeProcess = Start-BlockchainNode
        Start-Sleep -Seconds 3
        
        $walletProcess = Start-WalletUI
        Start-Sleep -Seconds 2
        
        $explorerProcess = Start-BlockchainExplorer
        Start-Sleep -Seconds 2
        
        Show-Status
        Wait-ForExit
        
    } catch {
        Write-Host "❌ Error occurred: $($_.Exception.Message)" -ForegroundColor $Red
    } finally {
        Stop-AllServices
    }
}

# Обработка Ctrl+C
$null = Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action {
    Stop-AllServices
}

# Запуск основной функции
Main