# Volnix Protocol System Check
# Проверяет готовность системы к запуску

Write-Host "🔍 Volnix Protocol System Check" -ForegroundColor Cyan
Write-Host "===============================" -ForegroundColor Cyan
Write-Host ""

$allGood = $true

# Функция для проверки команды
function Test-Command($command, $name) {
    try {
        $result = Invoke-Expression $command 2>$null
        Write-Host "✅ $name`: $result" -ForegroundColor Green
        return $true
    } catch {
        Write-Host "❌ $name`: Not found or error" -ForegroundColor Red
        return $false
    }
}

# Функция для проверки файла
function Test-FileExists($path, $name) {
    if (Test-Path $path) {
        Write-Host "✅ $name`: Found" -ForegroundColor Green
        return $true
    } else {
        Write-Host "❌ $name`: Not found" -ForegroundColor Red
        return $false
    }
}

# Функция для проверки порта
function Test-Port($port, $name) {
    try {
        $connection = Test-NetConnection -ComputerName localhost -Port $port -WarningAction SilentlyContinue
        if ($connection.TcpTestSucceeded) {
            Write-Host "⚠️  $name (port $port): Already in use" -ForegroundColor Yellow
            return $false
        } else {
            Write-Host "✅ $name (port $port): Available" -ForegroundColor Green
            return $true
        }
    } catch {
        Write-Host "✅ $name (port $port): Available" -ForegroundColor Green
        return $true
    }
}

Write-Host "🔧 Checking Dependencies" -ForegroundColor Yellow
Write-Host "------------------------" -ForegroundColor Yellow

# Проверка Go
if (-not (Test-Command "go version" "Go")) {
    $allGood = $false
    Write-Host "   Install from: https://golang.org/dl/" -ForegroundColor Gray
}

# Проверка Node.js
if (-not (Test-Command "node --version" "Node.js")) {
    $allGood = $false
    Write-Host "   Install from: https://nodejs.org/" -ForegroundColor Gray
}

# Проверка npm
if (-not (Test-Command "npm --version" "npm")) {
    $allGood = $false
    Write-Host "   Usually comes with Node.js" -ForegroundColor Gray
}

Write-Host ""
Write-Host "📁 Checking Project Files" -ForegroundColor Yellow
Write-Host "-------------------------" -ForegroundColor Yellow

# Проверка основных файлов проекта
if (-not (Test-FileExists "go.mod" "Go module")) { $allGood = $false }
if (-not (Test-FileExists "cmd/volnixd/main.go" "Main volnixd source")) { $allGood = $false }
if (-not (Test-FileExists "cmd/volnixd-standalone/main.go" "Standalone source")) { $allGood = $false }
if (-not (Test-FileExists "frontend/wallet-ui/package.json" "Wallet UI config")) { $allGood = $false }
if (-not (Test-FileExists "frontend/blockchain-explorer/index.html" "Explorer files")) { $allGood = $false }

Write-Host ""
Write-Host "🔨 Checking Build Status" -ForegroundColor Yellow
Write-Host "------------------------" -ForegroundColor Yellow

# Проверка собранных бинарников
$hasBinary = $false
if (Test-FileExists "volnixd.exe" "volnixd binary") { $hasBinary = $true }
if (Test-FileExists "volnixd" "volnixd binary (Unix)") { $hasBinary = $true }

if (-not $hasBinary) {
    Write-Host "ℹ️  No binary found - will need to build" -ForegroundColor Blue
}

# Проверка зависимостей Go
Write-Host "Checking Go dependencies..." -ForegroundColor Gray
try {
    go mod verify > $null 2>&1
    Write-Host "✅ Go dependencies: Verified" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Go dependencies: Need download" -ForegroundColor Yellow
}

# Проверка зависимостей npm
if (Test-Path "frontend/wallet-ui/node_modules") {
    Write-Host "✅ npm dependencies: Installed" -ForegroundColor Green
} else {
    Write-Host "ℹ️  npm dependencies: Need installation" -ForegroundColor Blue
}

Write-Host ""
Write-Host "🌐 Checking Network Ports" -ForegroundColor Yellow
Write-Host "-------------------------" -ForegroundColor Yellow

# Проверка портов
$portsOk = $true
if (-not (Test-Port 26657 "RPC port")) { $portsOk = $false }
if (-not (Test-Port 26656 "P2P port")) { $portsOk = $false }
if (-not (Test-Port 3000 "Wallet UI port")) { $portsOk = $false }
if (-not (Test-Port 8080 "Explorer port")) { $portsOk = $false }

Write-Host ""
Write-Host "Checking Node Configuration" -ForegroundColor Yellow
Write-Host "------------------------------" -ForegroundColor Yellow

# Проверка конфигурации узла
if (Test-Path ".volnix") {
    Write-Host "✅ Node configuration: Found" -ForegroundColor Green
    
    if (Test-Path ".volnix/config/genesis.json") {
        Write-Host "✅ Genesis file: Found" -ForegroundColor Green
    } else {
        Write-Host "❌ Genesis file: Missing" -ForegroundColor Red
        $allGood = $false
    }
    
    if (Test-Path ".volnix/config/config.toml") {
        Write-Host "✅ Config file: Found" -ForegroundColor Green
    } else {
        Write-Host "❌ Config file: Missing" -ForegroundColor Red
        $allGood = $false
    }
} else {
    Write-Host "ℹ️  Node configuration: Not initialized" -ForegroundColor Blue
}

Write-Host ""
Write-Host "System Summary" -ForegroundColor Cyan
Write-Host "=================" -ForegroundColor Cyan

if ($allGood -and $portsOk) {
    Write-Host "🎉 System is ready to run Volnix Protocol!" -ForegroundColor Green
    Write-Host ""
    Write-Host "🚀 Quick Start Commands:" -ForegroundColor Cyan
    Write-Host "  powershell -ExecutionPolicy Bypass -File scripts/quick-start.ps1" -ForegroundColor White
    Write-Host "  powershell -ExecutionPolicy Bypass -File scripts/start-full-stack.ps1" -ForegroundColor White
} elseif (-not $allGood) {
    Write-Host "❌ System has missing dependencies or files" -ForegroundColor Red
    Write-Host ""
    Write-Host "🔧 Recommended Actions:" -ForegroundColor Yellow
    Write-Host "  1. Install missing dependencies" -ForegroundColor White
    Write-Host "  2. Run: go mod download" -ForegroundColor White
    Write-Host "  3. Run: go build -o volnixd.exe ./cmd/volnixd" -ForegroundColor White
    Write-Host "  4. Run system check again" -ForegroundColor White
} elseif (-not $portsOk) {
    Write-Host "⚠️  System ready but some ports are in use" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "🔧 Recommended Actions:" -ForegroundColor Yellow
    Write-Host "  1. Stop services using the ports" -ForegroundColor White
    Write-Host "  2. Or use different ports in configuration" -ForegroundColor White
    Write-Host "  3. Run: netstat -ano | findstr :PORT to find processes" -ForegroundColor White
}

Write-Host ""
Write-Host "📚 For detailed instructions, see: README.md or deprecated/guides/QUICK_START_GUIDE.md" -ForegroundColor Cyan