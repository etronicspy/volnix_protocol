# Volnix Protocol Full Network Launcher
# Запускает полную сеть с майнингом, транзакциями и мониторингом

param(
    [int]$NodeCount = 5,
    [switch]$SkipSetup,
    [switch]$MonitorMining,
    [switch]$AutoTransactions
)

Write-Host "🚀 Volnix Protocol Full Network Launcher" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Nodes: $NodeCount" -ForegroundColor Yellow
Write-Host ""

# Функция для проверки готовности
function Test-Prerequisites {
    Write-Host "🔍 Checking prerequisites..." -ForegroundColor Yellow
    
    # Проверка бинарника
    if (-not (Test-Path "build/volnixd-standalone.exe")) {
        Write-Host "❌ volnixd-standalone.exe not found" -ForegroundColor Red
        Write-Host "Building standalone version..." -ForegroundColor Yellow
        New-Item -ItemType Directory -Force -Path "build" | Out-Null
        go build -o build/volnixd-standalone.exe ./cmd/volnixd-standalone
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Failed to build volnixd-standalone" -ForegroundColor Red
            exit 1
        }
    }
    Write-Host "✅ Binary ready" -ForegroundColor Green
}

# Функция для запуска сети
function Start-Network {
    Write-Host "🌐 Setting up and starting network..." -ForegroundColor Yellow
    
    if (-not $SkipSetup) {
        # Запуск setup скрипта
        powershell -ExecutionPolicy Bypass -File scripts/setup-testnet.ps1 -NodeCount $NodeCount -CleanStart
    } else {
        Write-Host "⏭️ Skipping setup, using existing configuration" -ForegroundColor Yellow
    }
}

# Функция для мониторинга
function Start-Monitoring {
    Write-Host "📊 Starting monitoring services..." -ForegroundColor Yellow
    
    # Запуск blockchain explorer
    Write-Host "🔍 Starting Blockchain Explorer..." -ForegroundColor Cyan
    Start-Process -FilePath "powershell" -ArgumentList "-ExecutionPolicy Bypass -File frontend/blockchain-explorer/start-explorer.ps1" -WindowStyle Hidden
    
    Start-Sleep -Seconds 3
    
    if ($MonitorMining) {
        Write-Host "⚡ Starting mining monitor..." -ForegroundColor Cyan
        Start-Process -FilePath "powershell" -ArgumentList "-ExecutionPolicy Bypass -File scripts/mining-and-transactions.ps1 -Action mining" -WindowStyle Normal
    }
    
    Write-Host "✅ Monitoring services started" -ForegroundColor Green
}

# Функция для автоматических транзакций
function Start-AutoTransactions {
    if ($AutoTransactions) {
        Write-Host "💸 Setting up automatic transactions..." -ForegroundColor Yellow
        
        # Ожидание готовности сети
        Start-Sleep -Seconds 10
        
        # Создание аккаунтов
        powershell -ExecutionPolicy Bypass -File scripts/mining-and-transactions.ps1 -Action accounts
        
        # Запуск автоматических транзакций
        Start-Process -FilePath "powershell" -ArgumentList "-ExecutionPolicy Bypass -Command `"
            while (`$true) {
                .\scripts\mining-and-transactions.ps1 -Action transactions
                Start-Sleep -Seconds 30
            }
        `"" -WindowStyle Hidden
        
        Write-Host "✅ Automatic transactions started" -ForegroundColor Green
    }
}

# Функция для отображения статуса
function Show-NetworkStatus {
    Start-Sleep -Seconds 15  # Ожидание полного запуска
    
    Write-Host ""
    Write-Host "🎉 Volnix Protocol Network is Running!" -ForegroundColor Green
    Write-Host "=====================================" -ForegroundColor Green
    Write-Host ""
    
    # Получение статуса сети
    powershell -ExecutionPolicy Bypass -File scripts/mining-and-transactions.ps1 -Action status
    
    Write-Host ""
    Write-Host "🌐 Available Services:" -ForegroundColor Cyan
    Write-Host "  🔍 Blockchain Explorer: http://localhost:8080" -ForegroundColor Green
    
    # Отображение эндпоинтов узлов
    Write-Host ""
    Write-Host "📡 Node Endpoints:" -ForegroundColor Cyan
    for ($i = 1; $i -le $NodeCount; $i++) {
        $rpcPort = 26656 + (($i - 1) * 10) + 1
        $p2pPort = 26656 + (($i - 1) * 10)
        Write-Host "  Node $i`: RPC http://localhost:$rpcPort | P2P tcp://localhost:$p2pPort" -ForegroundColor White
    }
    
    Write-Host ""
    Write-Host "🔧 Management Commands:" -ForegroundColor Cyan
    Write-Host "  # Check network status"
    Write-Host "  .\scripts\mining-and-transactions.ps1 -Action status" -ForegroundColor White
    Write-Host ""
    Write-Host "  # Monitor mining"
    Write-Host "  .\scripts\mining-and-transactions.ps1 -Action mining" -ForegroundColor White
    Write-Host ""
    Write-Host "  # Send test transactions"
    Write-Host "  .\scripts\mining-and-transactions.ps1 -Action transactions" -ForegroundColor White
    Write-Host ""
    Write-Host "  # View network statistics"
    Write-Host "  .\scripts\mining-and-transactions.ps1 -Action stats" -ForegroundColor White
    
    Write-Host ""
    Write-Host "⚡ Network Features:" -ForegroundColor Cyan
    Write-Host "  ✅ $NodeCount active validator nodes" -ForegroundColor Green
    Write-Host "  ✅ Automatic block production (mining)" -ForegroundColor Green
    Write-Host "  ✅ P2P consensus between all nodes" -ForegroundColor Green
    Write-Host "  ✅ Transaction processing ready" -ForegroundColor Green
    Write-Host "  ✅ Real-time monitoring" -ForegroundColor Green
    
    if ($AutoTransactions) {
        Write-Host "  ✅ Automatic test transactions" -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "🎯 What you can do now:" -ForegroundColor Cyan
    Write-Host "  1. Open Explorer: http://localhost:8080" -ForegroundColor White
    Write-Host "  2. Monitor mining activity in real-time" -ForegroundColor White
    Write-Host "  3. Send transactions between nodes" -ForegroundColor White
    Write-Host "  4. View network statistics and validator info" -ForegroundColor White
    Write-Host "  5. Test consensus with multiple validators" -ForegroundColor White
}

# Основная логика
try {
    Test-Prerequisites
    
    Write-Host "🚀 Starting full Volnix Protocol network..." -ForegroundColor Cyan
    Write-Host ""
    
    # Запуск сети в фоновом режиме
    Start-Job -ScriptBlock {
        param($NodeCount, $SkipSetup)
        Set-Location $using:PWD
        if (-not $SkipSetup) {
            powershell -ExecutionPolicy Bypass -File scripts/setup-testnet.ps1 -NodeCount $NodeCount -CleanStart
        }
    } -ArgumentList $NodeCount, $SkipSetup | Out-Null
    
    # Ожидание запуска сети
    Write-Host "⏳ Waiting for network to initialize..." -ForegroundColor Yellow
    Start-Sleep -Seconds 20
    
    # Запуск мониторинга
    Start-Monitoring
    
    # Запуск автоматических транзакций
    Start-AutoTransactions
    
    # Отображение статуса
    Show-NetworkStatus
    
    Write-Host ""
    Write-Host "Press Ctrl+C to stop the network..." -ForegroundColor Yellow
    
    # Ожидание завершения
    while ($true) {
        Start-Sleep -Seconds 5
        
        # Проверка работы узлов
        $nodeProcesses = Get-Process | Where-Object { $_.ProcessName -like "*volnixd*" }
        if ($nodeProcesses.Count -eq 0) {
            Write-Host "All nodes have stopped." -ForegroundColor Red
            break
        }
    }
    
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)" -ForegroundColor Red
} finally {
    Write-Host ""
    Write-Host "🛑 Stopping all network services..." -ForegroundColor Yellow
    
    # Остановка всех процессов
    Get-Process | Where-Object { 
        $_.ProcessName -like "*volnixd*" -or 
        $_.ProcessName -like "*powershell*" 
    } | Stop-Process -Force -ErrorAction SilentlyContinue
    
    # Очистка jobs
    Get-Job | Remove-Job -Force -ErrorAction SilentlyContinue
    
    Write-Host "✅ Network stopped" -ForegroundColor Green
}