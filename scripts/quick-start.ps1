# Volnix Protocol Quick Start
# Быстрый запуск основных компонентов

Write-Host "🚀 Volnix Protocol Quick Start" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan

# 1. Сборка проекта
Write-Host "🔨 Building project..." -ForegroundColor Yellow
go build -o volnixd.exe ./cmd/volnixd

# 2. Инициализация узла (если нужно)
if (-not (Test-Path ".volnix")) {
    Write-Host "🏗️ Initializing node..." -ForegroundColor Yellow
    .\volnixd.exe init testnode --chain-id volnix-testnet
}

# 3. Запуск блокчейн узла
Write-Host "🌐 Starting blockchain node..." -ForegroundColor Yellow
Start-Process -FilePath ".\volnixd.exe" -ArgumentList "start" -WindowStyle Hidden

# Ожидание запуска узла
Start-Sleep -Seconds 5

# 4. Запуск Wallet UI
Write-Host "💰 Starting Wallet UI..." -ForegroundColor Yellow
Push-Location "wallet-ui"
if (-not (Test-Path "node_modules")) {
    npm install
}
Start-Process -FilePath "npm" -ArgumentList "start" -WindowStyle Hidden
Pop-Location

# 5. Запуск Blockchain Explorer
Write-Host "🔍 Starting Blockchain Explorer..." -ForegroundColor Yellow
Push-Location "blockchain-explorer"
Start-Process -FilePath "powershell" -ArgumentList "-ExecutionPolicy Bypass -File start-explorer.ps1" -WindowStyle Hidden
Pop-Location

# Ожидание запуска всех сервисов
Start-Sleep -Seconds 10

Write-Host ""
Write-Host "🎉 Volnix Protocol is running!" -ForegroundColor Green
Write-Host "==============================" -ForegroundColor Green
Write-Host ""
Write-Host "📊 Available Services:" -ForegroundColor Cyan
Write-Host "  🌐 Blockchain Node: http://localhost:26657" -ForegroundColor Green
Write-Host "  💰 Wallet UI:       http://localhost:3000" -ForegroundColor Green  
Write-Host "  🔍 Explorer:        http://localhost:8080" -ForegroundColor Green
Write-Host ""
Write-Host "🚀 Open your browser and visit the URLs above!" -ForegroundColor Magenta
Write-Host ""
Write-Host "Press any key to exit..." -ForegroundColor Yellow
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")