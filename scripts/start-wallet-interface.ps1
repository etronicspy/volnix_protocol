# Volnix Protocol - Complete Wallet Interface Launcher
# Запускает блокчейн, веб-кошелек и все необходимые сервисы

Write-Host "🚀 Volnix Protocol - Complete Wallet Interface" -ForegroundColor Cyan
Write-Host "===============================================" -ForegroundColor Cyan

# Функция для проверки портов
function Test-Port($port) {
    try {
        $connection = Test-NetConnection -ComputerName localhost -Port $port -WarningAction SilentlyContinue
        return -not $connection.TcpTestSucceeded
    } catch {
        return $true
    }
}

# Проверка доступности портов
Write-Host "🔍 Checking ports..." -ForegroundColor Yellow
$ports = @(26657, 3000, 8080)
$portsAvailable = $true

foreach ($port in $ports) {
    if (-not (Test-Port $port)) {
        Write-Host "❌ Port $port is already in use" -ForegroundColor Red
        $portsAvailable = $false
    } else {
        Write-Host "✅ Port $port is available" -ForegroundColor Green
    }
}

if (-not $portsAvailable) {
    Write-Host ""
    Write-Host "⚠️ Some ports are in use. Stop other services or change ports." -ForegroundColor Yellow
    Write-Host "Press any key to continue anyway..." -ForegroundColor Gray
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}

Write-Host ""
Write-Host "🔧 Setting up Volnix Protocol..." -ForegroundColor Yellow

# 1. Создание genesis аккаунтов
Write-Host "🌟 Creating genesis accounts..." -ForegroundColor Cyan
powershell -ExecutionPolicy Bypass -File scripts/transaction-manager.ps1 -Action genesis

# 2. Создание тестовых кошельков
Write-Host "👛 Creating test wallets..." -ForegroundColor Cyan
powershell -ExecutionPolicy Bypass -File scripts/transaction-manager.ps1 -Action test

Write-Host ""
Write-Host "🚀 Starting services..." -ForegroundColor Yellow

# 3. Запуск блокчейн узла
Write-Host "🌐 Starting blockchain node..." -ForegroundColor Cyan
$nodeProcess = Start-Process -FilePath ".\bin\volnixd.exe" -ArgumentList "start" -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 5

# 4. Запуск Blockchain Explorer
Write-Host "🔍 Starting Blockchain Explorer..." -ForegroundColor Cyan
$explorerProcess = Start-Process -FilePath "powershell" -ArgumentList "-ExecutionPolicy Bypass -File blockchain-explorer/start-explorer.ps1" -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 2

# 5. Запуск Wallet Web Interface
Write-Host "💰 Starting Wallet Web Interface..." -ForegroundColor Cyan
$walletProcess = Start-Process -FilePath "powershell" -ArgumentList "-ExecutionPolicy Bypass -File wallet-web/server.ps1" -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 3

Write-Host ""
Write-Host "🎉 Volnix Protocol is fully operational!" -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Green
Write-Host ""

Write-Host "🌐 Available Services:" -ForegroundColor Cyan
Write-Host "  💰 Wallet Interface:    http://localhost:3000" -ForegroundColor Green
Write-Host "  🔍 Blockchain Explorer: http://localhost:8080" -ForegroundColor Green
Write-Host "  🌐 Blockchain Node:     http://localhost:26657" -ForegroundColor Green

Write-Host ""
Write-Host "💰 Wallet Features:" -ForegroundColor Cyan
Write-Host "  ✅ Create and manage wallets" -ForegroundColor White
Write-Host "  ✅ Send VX, LZN, ANT tokens" -ForegroundColor White
Write-Host "  ✅ View balances and transaction history" -ForegroundColor White
Write-Host "  ✅ Real-time transaction processing" -ForegroundColor White
Write-Host "  ✅ Test wallets with initial balances" -ForegroundColor White

Write-Host ""
Write-Host "🔧 CLI Commands Available:" -ForegroundColor Cyan
Write-Host "  # Create wallet"
Write-Host "  .\scripts\transaction-manager.ps1 -Action create -KeyName myWallet" -ForegroundColor Gray
Write-Host ""
Write-Host "  # Get funds from faucet"
Write-Host "  .\scripts\transaction-manager.ps1 -Action faucet -KeyName myWallet" -ForegroundColor Gray
Write-Host ""
Write-Host "  # Send transaction"
Write-Host "  .\scripts\transaction-manager.ps1 -Action send -From alice -To bob -Amount 1000000" -ForegroundColor Gray
Write-Host ""
Write-Host "  # Create validator"
Write-Host "  .\scripts\transaction-manager.ps1 -Action create-validator -KeyName myWallet" -ForegroundColor Gray

Write-Host ""
Write-Host "🎯 Quick Start:" -ForegroundColor Cyan
Write-Host "  1. Open Wallet: http://localhost:3000" -ForegroundColor White
Write-Host "  2. Select a test wallet (alice, bob, charlie)" -ForegroundColor White
Write-Host "  3. Send transactions between wallets" -ForegroundColor White
Write-Host "  4. Monitor on Explorer: http://localhost:8080" -ForegroundColor White

Write-Host ""
Write-Host "💡 Test Wallets (already created):" -ForegroundColor Cyan
Write-Host "  👤 alice   - 1000 VX, 500 LZN, 100 ANT" -ForegroundColor White
Write-Host "  👤 bob     - 1000 VX, 500 LZN, 100 ANT" -ForegroundColor White
Write-Host "  👤 charlie - 1000 VX, 500 LZN, 100 ANT" -ForegroundColor White
Write-Host "  🏛️ validator1 - 1000 VX, 500 LZN, 100 ANT" -ForegroundColor White
Write-Host "  💼 trader1 - 1000 VX, 500 LZN, 100 ANT" -ForegroundColor White

Write-Host ""
Write-Host "🚀 READY TO USE! Open http://localhost:3000 in your browser!" -ForegroundColor Magenta
Write-Host ""
Write-Host "Press Ctrl+C to stop all services..." -ForegroundColor Yellow

# Функция для остановки всех сервисов
function Stop-AllServices {
    Write-Host ""
    Write-Host "🛑 Stopping all services..." -ForegroundColor Yellow
    
    # Остановка процессов
    Get-Process | Where-Object { 
        $_.ProcessName -like "*volnixd*" -or 
        $_.ProcessName -like "*powershell*" 
    } | Stop-Process -Force -ErrorAction SilentlyContinue
    
    Write-Host "✅ All services stopped" -ForegroundColor Green
}

# Обработка Ctrl+C
try {
    while ($true) {
        Start-Sleep -Seconds 5
        
        # Проверка работы основных процессов
        $nodeRunning = Get-Process | Where-Object { $_.ProcessName -like "*volnixd*" }
        if (-not $nodeRunning) {
            Write-Host "❌ Blockchain node stopped unexpectedly" -ForegroundColor Red
            break
        }
    }
} catch {
    # Пользователь нажал Ctrl+C
} finally {
    Stop-AllServices
}

Write-Host ""
Write-Host "👋 Thank you for using Volnix Protocol!" -ForegroundColor Cyan