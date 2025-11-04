# Volnix Protocol Testnet Startup Script
# Запускает 3 узла для тестирования сети

Write-Host "🚀 Starting Volnix Protocol Testnet..." -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Green
Write-Host ""

# Проверяем, что исполняемый файл существует
if (-not (Test-Path ".\volnixd-integrated.exe")) {
    Write-Host "❌ volnixd-integrated.exe not found!" -ForegroundColor Red
    Write-Host "Please run: go build -o volnixd-integrated.exe ./cmd/volnixd" -ForegroundColor Yellow
    exit 1
}

# Инициализируем testnet
Write-Host "🔧 Initializing testnet..." -ForegroundColor Cyan
& .\volnixd-integrated.exe network init-testnet 3

Write-Host ""
Write-Host "🌐 Starting network nodes..." -ForegroundColor Cyan

# Запускаем узлы в фоновых процессах
Write-Host "🚀 Starting Node 0..." -ForegroundColor Yellow
Start-Process -FilePath ".\volnixd-integrated.exe" -ArgumentList "network", "start-node", "0" -WindowStyle Minimized

Start-Sleep -Seconds 2

Write-Host "🚀 Starting Node 1..." -ForegroundColor Yellow  
Start-Process -FilePath ".\volnixd-integrated.exe" -ArgumentList "network", "start-node", "1" -WindowStyle Minimized

Start-Sleep -Seconds 2

Write-Host "🚀 Starting Node 2..." -ForegroundColor Yellow
Start-Process -FilePath ".\volnixd-integrated.exe" -ArgumentList "network", "start-node", "2" -WindowStyle Minimized

Start-Sleep -Seconds 3

Write-Host ""
Write-Host "✅ All nodes started!" -ForegroundColor Green
Write-Host ""

# Показываем статус сети
Write-Host "📊 Network Status:" -ForegroundColor Cyan
& .\volnixd-integrated.exe network status

Write-Host ""
Write-Host "🧪 Testing consensus..." -ForegroundColor Cyan
& .\volnixd-integrated.exe network test-consensus

Write-Host ""
Write-Host "🔧 Testing modules..." -ForegroundColor Cyan
& .\volnixd-integrated.exe network test-modules

Write-Host ""
Write-Host "🎉 Volnix Protocol Testnet is running!" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Available commands:" -ForegroundColor White
Write-Host "  .\volnixd-integrated.exe network status" -ForegroundColor Gray
Write-Host "  .\volnixd-integrated.exe network test-consensus" -ForegroundColor Gray
Write-Host "  .\volnixd-integrated.exe network test-modules" -ForegroundColor Gray
Write-Host ""
Write-Host "🛑 To stop all nodes, close the PowerShell windows or press Ctrl+C" -ForegroundColor Yellow