# Volnix Protocol - Simple Build Script

Write-Host "Building Volnix Protocol..." -ForegroundColor Cyan

# Check Go
try {
    $goVersion = go version
    Write-Host "Go: $goVersion" -ForegroundColor Green
}
catch {
    Write-Host "Error: Go not found" -ForegroundColor Red
    exit 1
}

# Create build directory
if (!(Test-Path "build")) {
    New-Item -ItemType Directory -Path "build" | Out-Null
}

# Build binary
Write-Host "Building binary..." -ForegroundColor Yellow
$env:CGO_ENABLED = "0"

# Build for Windows
go build -o build\volnixd.exe .\cmd\volnixd
if ($LASTEXITCODE -eq 0) {
    Write-Host "Windows binary built successfully" -ForegroundColor Green
} else {
    Write-Host "Failed to build Windows binary" -ForegroundColor Red
}

# Build for Linux
$env:GOOS = "linux"
go build -o build\volnixd .\cmd\volnixd
if ($LASTEXITCODE -eq 0) {
    Write-Host "Linux binary built successfully" -ForegroundColor Green
} else {
    Write-Host "Failed to build Linux binary" -ForegroundColor Red
}

# Reset environment
$env:GOOS = ""

# Show results
Write-Host "`nBuild completed!" -ForegroundColor Cyan
Write-Host "Binaries in build/ directory:" -ForegroundColor White
Get-ChildItem build | ForEach-Object {
    $size = [math]::Round($_.Length / 1MB, 2)
    Write-Host "  $($_.Name) - $size MB" -ForegroundColor Gray
}

Write-Host "`nTest the binary:" -ForegroundColor Yellow
Write-Host "  .\build\volnixd.exe version" -ForegroundColor Gray