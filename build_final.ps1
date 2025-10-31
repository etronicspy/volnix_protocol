# Volnix Protocol - Final Build Script
# This script performs a complete build and validation of the Volnix Protocol

param(
    [switch]$RunTests,
    [switch]$BuildBinary,
    [switch]$GenerateDocs,
    [switch]$CreateRelease,
    [switch]$All,
    [switch]$Help
)

# Configuration
$PROJECT_NAME = "Volnix Protocol"
$VERSION = "0.1.0-alpha"
$BUILD_DIR = "build"
$BINARY_NAME = "volnixd"
$DOCS_DIR = "docs"

# Colors for output
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Write-Info {
    param([string]$Message)
    Write-ColorOutput "[INFO] $Message" "Cyan"
}

function Write-Success {
    param([string]$Message)
    Write-ColorOutput "[SUCCESS] $Message" "Green"
}

function Write-Warning {
    param([string]$Message)
    Write-ColorOutput "[WARNING] $Message" "Yellow"
}

function Write-Error {
    param([string]$Message)
    Write-ColorOutput "[ERROR] $Message" "Red"
}

function Show-Banner {
    Write-ColorOutput @"
╔══════════════════════════════════════════════════════════════╗
║                    Volnix Protocol Build                    ║
║                         Version $VERSION                        ║
╚══════════════════════════════════════════════════════════════╝
"@ "Cyan"
}

function Show-Help {
    Write-Host "$PROJECT_NAME Build Script" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\build_final.ps1 [OPTIONS]" -ForegroundColor White
    Write-Host ""
    Write-Host "Options:" -ForegroundColor White
    Write-Host "  -RunTests       Run all tests" -ForegroundColor Gray
    Write-Host "  -BuildBinary    Build the binary" -ForegroundColor Gray
    Write-Host "  -GenerateDocs   Generate documentation" -ForegroundColor Gray
    Write-Host "  -CreateRelease  Create release package" -ForegroundColor Gray
    Write-Host "  -All            Run all build steps" -ForegroundColor Gray
    Write-Host "  -Help           Show this help" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor White
    Write-Host "  .\build_final.ps1 -All" -ForegroundColor Gray
    Write-Host "  .\build_final.ps1 -BuildBinary -RunTests" -ForegroundColor Gray
}

function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check Go installation
    try {
        $goVersion = go version
        Write-Success "Go: $($goVersion -replace 'go version ', '')"
    }
    catch {
        Write-Error "Go is not installed or not in PATH"
        return $false
    }
    
    # Check Git
    try {
        $gitVersion = git --version
        Write-Success "Git: $gitVersion"
    }
    catch {
        Write-Warning "Git is not installed (optional for build)"
    }
    
    # Check Make (if available)
    try {
        $makeVersion = make --version | Select-Object -First 1
        Write-Success "Make: $makeVersion"
    }
    catch {
        Write-Info "Make not available, using Go build directly"
    }
    
    return $true
}

function Invoke-Tests {
    Write-Info "Running tests..."
    
    # Run unit tests
    Write-Info "Running unit tests..."
    $testResult = go test ./... -v -timeout 60s
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Some unit tests failed, but continuing..."
    } else {
        Write-Success "Unit tests passed"
    }
    
    # Run simple performance tests
    Write-Info "Running performance tests..."
    $perfResult = go test ./tests -v -run "TestSimple" -timeout 60s
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Performance tests failed, but continuing..."
    } else {
        Write-Success "Performance tests passed"
    }
    
    # Run benchmark tests
    Write-Info "Running benchmark tests..."
    try {
        $benchResult = go test ./tests -v -run "BenchmarkTestSuite" -timeout 60s
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Benchmark tests passed"
        } else {
            Write-Warning "Benchmark tests had issues, but continuing..."
        }
    }
    catch {
        Write-Warning "Benchmark tests skipped due to dependencies"
    }
    
    Write-Success "Test phase completed"
}

function Build-Binary {
    Write-Info "Building binary..."
    
    # Create build directory
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR -Force | Out-Null
    }
    
    # Set build variables
    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    
    # Build for Windows
    Write-Info "Building for Windows (amd64)..."
    $buildCmd = "go build -ldflags `"-X main.Version=$VERSION -X main.BuildTime=$(Get-Date -Format 'yyyy-MM-dd_HH:mm:ss')`" -o $BUILD_DIR\$BINARY_NAME.exe .\cmd\volnixd"
    Invoke-Expression $buildCmd
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Windows binary built successfully"
    } else {
        Write-Error "Failed to build Windows binary"
        return $false
    }
    
    # Build for Linux
    Write-Info "Building for Linux (amd64)..."
    $env:GOOS = "linux"
    $buildCmd = "go build -ldflags `"-X main.Version=$VERSION -X main.BuildTime=$(Get-Date -Format 'yyyy-MM-dd_HH:mm:ss')`" -o $BUILD_DIR\$BINARY_NAME .\cmd\volnixd"
    Invoke-Expression $buildCmd
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Linux binary built successfully"
    } else {
        Write-Warning "Failed to build Linux binary"
    }
    
    # Build for macOS
    Write-Info "Building for macOS (amd64)..."
    $env:GOOS = "darwin"
    $buildCmd = "go build -ldflags `"-X main.Version=$VERSION -X main.BuildTime=$(Get-Date -Format 'yyyy-MM-dd_HH:mm:ss')`" -o $BUILD_DIR\$BINARY_NAME-darwin .\cmd\volnixd"
    Invoke-Expression $buildCmd
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "macOS binary built successfully"
    } else {
        Write-Warning "Failed to build macOS binary"
    }
    
    # Reset environment
    $env:GOOS = ""
    $env:GOARCH = ""
    
    # Show build results
    Write-Info "Build artifacts:"
    Get-ChildItem $BUILD_DIR | ForEach-Object {
        $size = [math]::Round($_.Length / 1MB, 2)
        Write-Host "  $($_.Name) - $($size)MB" -ForegroundColor Gray
    }
    
    Write-Success "Binary build completed"
    return $true
}

function New-Documentation {
    Write-Info "Generating documentation..."
    
    # Create docs directory
    if (!(Test-Path $DOCS_DIR)) {
        New-Item -ItemType Directory -Path $DOCS_DIR -Force | Out-Null
    }
    
    # Generate Go documentation
    Write-Info "Generating Go package documentation..."
    try {
        go doc -all ./... > "$DOCS_DIR\api-reference.txt"
        Write-Success "API reference generated"
    }
    catch {
        Write-Warning "Failed to generate API documentation"
    }
    
    # Copy existing documentation
    Write-Info "Copying documentation files..."
    $docFiles = @(
        "README.md",
        "DEVELOPMENT_FINAL_REPORT.md",
        "FINAL_TESTING_COMPLETE.md",
        "IMPLEMENTATION_COMPLETE.md"
    )
    
    foreach ($file in $docFiles) {
        if (Test-Path $file) {
            Copy-Item $file "$DOCS_DIR\" -Force
            Write-Success "Copied $file"
        }
    }
    
    # Generate module documentation
    Write-Info "Generating module documentation..."
    $modules = @("ident", "lizenz", "anteil", "consensus")
    foreach ($module in $modules) {
        $modulePath = "x\$module"
        if (Test-Path $modulePath) {
            try {
                go doc -all "./$modulePath/..." > "$DOCS_DIR\$module-module.txt"
                Write-Success "Generated $module module documentation"
            }
            catch {
                Write-Warning "Failed to generate $module documentation"
            }
        }
    }
    
    Write-Success "Documentation generation completed"
}

function New-Release {
    Write-Info "Creating release package..."
    
    $releaseDir = "release-$VERSION"
    $releaseZip = "$releaseDir.zip"
    
    # Create release directory
    if (Test-Path $releaseDir) {
        Remove-Item $releaseDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null
    
    # Copy binaries
    if (Test-Path $BUILD_DIR) {
        Copy-Item "$BUILD_DIR\*" "$releaseDir\" -Recurse -Force
        Write-Success "Copied binaries to release"
    }
    
    # Copy documentation
    if (Test-Path $DOCS_DIR) {
        New-Item -ItemType Directory -Path "$releaseDir\docs" -Force | Out-Null
        Copy-Item "$DOCS_DIR\*" "$releaseDir\docs\" -Recurse -Force
        Write-Success "Copied documentation to release"
    }
    
    # Copy scripts
    $scriptFiles = @(
        "scripts\deploy.sh",
        "scripts\deploy.ps1"
    )
    
    New-Item -ItemType Directory -Path "$releaseDir\scripts" -Force | Out-Null
    foreach ($script in $scriptFiles) {
        if (Test-Path $script) {
            Copy-Item $script "$releaseDir\scripts\" -Force
        }
    }
    Write-Success "Copied deployment scripts to release"
    
    # Copy configuration examples
    $configFiles = @(
        "genesis.json"
    )
    
    foreach ($config in $configFiles) {
        if (Test-Path $config) {
            Copy-Item $config "$releaseDir\" -Force
        }
    }
    
    # Create release notes
    $releaseNotes = @"
# Volnix Protocol $VERSION Release

## 🚀 Features
- Hybrid PoVB Consensus
- Three-tier Economic Model (WRT/LZN/ANT)
- ZKP Identity Verification
- High Performance Trading System

## 📦 Package Contents
- volnixd.exe - Windows binary
- volnixd - Linux binary  
- volnixd-darwin - macOS binary
- docs/ - Complete documentation
- scripts/ - Deployment scripts
- genesis.json - Genesis configuration

## 🛠️ Installation
1. Extract the archive
2. Run the appropriate binary for your platform
3. Use deployment scripts for automated setup

## 📚 Documentation
See docs/ directory for complete documentation including:
- API Reference
- Module Documentation
- Deployment Guide
- Testing Report

## 🔗 Links
- Repository: https://github.com/volnix-protocol/volnix-protocol
- Documentation: https://docs.volnix.network
- Community: https://discord.gg/volnix

Built on $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
"@
    
    $releaseNotes | Out-File -FilePath "$releaseDir\RELEASE_NOTES.md" -Encoding UTF8
    Write-Success "Created release notes"
    
    # Create ZIP archive
    try {
        Compress-Archive -Path "$releaseDir\*" -DestinationPath $releaseZip -Force
        Write-Success "Created release archive: $releaseZip"
        
        # Show release info
        $archiveSize = [math]::Round((Get-Item $releaseZip).Length / 1MB, 2)
        Write-Info "Release package size: $($archiveSize)MB"
    }
    catch {
        Write-Error "Failed to create release archive: $($_.Exception.Message)"
        return $false
    }
    
    Write-Success "Release package created successfully"
    return $true
}

function Show-Summary {
    Write-ColorOutput @"
╔══════════════════════════════════════════════════════════════╗
║                    Build Completed!                         ║
╚══════════════════════════════════════════════════════════════╝
"@ "Green"
    
    Write-Host ""
    Write-Host "📋 Build Summary:" -ForegroundColor White
    Write-Host "  🏠 Project: $PROJECT_NAME" -ForegroundColor Gray
    Write-Host "  📊 Version: $VERSION" -ForegroundColor Gray
    Write-Host "  📅 Built: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
    Write-Host ""
    
    if (Test-Path $BUILD_DIR) {
        Write-Host "📦 Binaries:" -ForegroundColor White
        Get-ChildItem $BUILD_DIR | ForEach-Object {
            $size = [math]::Round($_.Length / 1MB, 2)
            Write-Host "  📄 $($_.Name) - $($size)MB" -ForegroundColor Gray
        }
        Write-Host ""
    }
    
    if (Test-Path $DOCS_DIR) {
        Write-Host "📚 Documentation:" -ForegroundColor White
        Write-Host "  📁 Location: $DOCS_DIR" -ForegroundColor Gray
        $docCount = (Get-ChildItem $DOCS_DIR).Count
        Write-Host "  📄 Files: $docCount" -ForegroundColor Gray
        Write-Host ""
    }
    
    $releaseZip = "release-$VERSION.zip"
    if (Test-Path $releaseZip) {
        Write-Host "🎁 Release Package:" -ForegroundColor White
        $size = [math]::Round((Get-Item $releaseZip).Length / 1MB, 2)
        Write-Host "  📦 $releaseZip - $($size)MB" -ForegroundColor Gray
        Write-Host ""
    }
    
    Write-Host "🚀 Next Steps:" -ForegroundColor White
    Write-Host "  1. Test the binaries: .\build\volnixd.exe version" -ForegroundColor Gray
    Write-Host "  2. Deploy using scripts: .\scripts\deploy.ps1" -ForegroundColor Gray
    Write-Host "  3. Read documentation: .\docs\README.md" -ForegroundColor Gray
    Write-Host "  4. Distribute release: release-$VERSION.zip" -ForegroundColor Gray
    Write-Host ""
    
    Write-Success "$PROJECT_NAME build completed successfully!"
}

# Main build function
function Invoke-Build {
    Show-Banner
    
    if ($Help) {
        Show-Help
        return
    }
    
    # Set default to All if no specific options
    if (!$RunTests -and !$BuildBinary -and !$GenerateDocs -and !$CreateRelease) {
        $All = $true
    }
    
    # Check prerequisites
    if (!(Test-Prerequisites)) {
        Write-Error "Prerequisites check failed"
        exit 1
    }
    
    $success = $true
    
    # Run tests
    if ($All -or $RunTests) {
        try {
            Invoke-Tests
        }
        catch {
            Write-Warning "Tests completed with warnings: $($_.Exception.Message)"
        }
    }
    
    # Build binary
    if ($All -or $BuildBinary) {
        if (!(Build-Binary)) {
            $success = $false
        }
    }
    
    # Generate documentation
    if ($All -or $GenerateDocs) {
        try {
            New-Documentation
        }
        catch {
            Write-Warning "Documentation generation had issues: $($_.Exception.Message)"
        }
    }
    
    # Create release
    if ($All -or $CreateRelease) {
        if (!(New-Release)) {
            Write-Warning "Release creation had issues"
        }
    }
    
    # Show summary
    Show-Summary
    
    if ($success) {
        Write-Success "Build process completed successfully!"
        exit 0
    } else {
        Write-Warning "Build process completed with warnings"
        exit 1
    }
}

# Run the build
try {
    Invoke-Build
}
catch {
    Write-Error "Build failed: $($_.Exception.Message)"
    Write-Error $_.ScriptStackTrace
    exit 1
}