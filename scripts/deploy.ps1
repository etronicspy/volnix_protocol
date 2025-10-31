# Volnix Protocol Deployment Script for Windows
# This script automates the deployment of Volnix Protocol nodes on Windows

param(
    [string]$Moniker = $env:COMPUTERNAME,
    [string]$ChainId = "volnix-1",
    [switch]$EnableStateSync,
    [switch]$EnableRPC,
    [switch]$EnableMonitoring,
    [switch]$SkipBuild,
    [switch]$Help
)

# Configuration
$VOLNIX_VERSION = "0.1.0-alpha"
$NODE_HOME = "$env:USERPROFILE\.volnix"
$BINARY_NAME = "volnixd.exe"
$GENESIS_URL = "https://raw.githubusercontent.com/volnix-protocol/mainnet/main/genesis.json"
$SEEDS = "seed1.volnix.network:26656,seed2.volnix.network:26656"

# Functions
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
║                    Volnix Protocol Deployment               ║
║                         Version $VOLNIX_VERSION                        ║
╚══════════════════════════════════════════════════════════════╝
"@ "Cyan"
}

function Test-Requirements {
    Write-Info "Checking system requirements..."
    
    # Check Windows version
    $osVersion = [System.Environment]::OSVersion.Version
    Write-Success "Operating System: Windows $($osVersion.Major).$($osVersion.Minor)"
    
    # Check Go installation
    try {
        $goVersion = go version
        Write-Success "Go version: $($goVersion -replace 'go version ', '')"
    }
    catch {
        Write-Error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    }
    
    # Check available disk space (minimum 100GB)
    $drive = Get-WmiObject -Class Win32_LogicalDisk -Filter "DeviceID='C:'"
    $availableSpaceGB = [math]::Round($drive.FreeSpace / 1GB, 2)
    if ($availableSpaceGB -lt 100) {
        Write-Warning "Available disk space: ${availableSpaceGB}GB (recommended: 100GB+)"
    } else {
        Write-Success "Available disk space: ${availableSpaceGB}GB"
    }
    
    # Check RAM (minimum 8GB)
    $totalRAMGB = [math]::Round((Get-WmiObject -Class Win32_ComputerSystem).TotalPhysicalMemory / 1GB, 2)
    if ($totalRAMGB -lt 8) {
        Write-Warning "Total RAM: ${totalRAMGB}GB (recommended: 8GB+)"
    } else {
        Write-Success "Total RAM: ${totalRAMGB}GB"
    }
}

function Install-Binary {
    Write-Info "Installing Volnix Protocol binary..."
    
    # Check if binary already exists
    if (Get-Command $BINARY_NAME -ErrorAction SilentlyContinue) {
        try {
            $currentVersion = & $BINARY_NAME version 2>$null | Select-String "v\d+\.\d+\.\d+" | ForEach-Object { $_.Matches[0].Value }
            Write-Info "Current version: $currentVersion"
        }
        catch {
            Write-Info "Current version: unknown"
        }
    }
    
    # Build from source
    if (Test-Path "volnix-protocol") {
        Write-Info "Using existing source code..."
        Set-Location volnix-protocol
        git pull origin main
    } else {
        Write-Info "Cloning Volnix Protocol repository..."
        git clone https://github.com/volnix-protocol/volnix-protocol.git
        Set-Location volnix-protocol
    }
    
    Write-Info "Building binary..."
    & make build
    
    # Copy binary to a location in PATH
    $binPath = "$env:USERPROFILE\bin"
    if (!(Test-Path $binPath)) {
        New-Item -ItemType Directory -Path $binPath -Force | Out-Null
    }
    
    Copy-Item "build\$BINARY_NAME" "$binPath\$BINARY_NAME" -Force
    
    # Add to PATH if not already there
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$binPath*") {
        [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$binPath", "User")
        $env:PATH += ";$binPath"
    }
    
    Write-Success "Binary installed successfully"
    
    # Verify installation
    try {
        $installedVersion = & $BINARY_NAME version 2>$null | Select-Object -First 1
        Write-Success "Installed version: $installedVersion"
    }
    catch {
        Write-Error "Installation verification failed"
    }
    
    Set-Location ..
}

function Initialize-Node {
    Write-Info "Initializing Volnix Protocol node..."
    
    Write-Info "Using moniker: $Moniker"
    
    # Initialize node
    & $BINARY_NAME init $Moniker --chain-id $ChainId --home $NODE_HOME
    
    Write-Success "Node initialized with moniker: $Moniker"
}

function Get-Genesis {
    Write-Info "Downloading genesis file..."
    
    $genesisPath = "$NODE_HOME\config\genesis.json"
    
    try {
        Invoke-WebRequest -Uri $GENESIS_URL -OutFile $genesisPath
        Write-Success "Genesis file downloaded successfully"
    }
    catch {
        Write-Warning "Failed to download genesis file from $GENESIS_URL"
        Write-Info "Using default genesis file..."
    }
    
    # Verify genesis file
    if (Test-Path $genesisPath) {
        $genesisHash = Get-FileHash $genesisPath -Algorithm SHA256
        Write-Info "Genesis hash: $($genesisHash.Hash)"
    }
}

function Set-NodeConfiguration {
    Write-Info "Configuring node settings..."
    
    $configFile = "$NODE_HOME\config\config.toml"
    $appFile = "$NODE_HOME\config\app.toml"
    
    if (Test-Path $configFile) {
        # Configure P2P settings
        (Get-Content $configFile) -replace 'seeds = ""', "seeds = `"$SEEDS`"" | Set-Content $configFile
        (Get-Content $configFile) -replace 'max_num_inbound_peers = 40', 'max_num_inbound_peers = 100' | Set-Content $configFile
        (Get-Content $configFile) -replace 'max_num_outbound_peers = 10', 'max_num_outbound_peers = 50' | Set-Content $configFile
        
        # Configure consensus settings
        (Get-Content $configFile) -replace 'timeout_commit = "5s"', 'timeout_commit = "3s"' | Set-Content $configFile
        (Get-Content $configFile) -replace 'timeout_propose = "3s"', 'timeout_propose = "2s"' | Set-Content $configFile
    }
    
    if (Test-Path $appFile) {
        # Configure pruning
        (Get-Content $appFile) -replace 'pruning = "default"', 'pruning = "custom"' | Set-Content $appFile
        (Get-Content $appFile) -replace 'pruning-keep-recent = "0"', 'pruning-keep-recent = "100000"' | Set-Content $appFile
        (Get-Content $appFile) -replace 'pruning-interval = "0"', 'pruning-interval = "10"' | Set-Content $appFile
    }
    
    Write-Success "Node configuration completed"
}

function Install-WindowsService {
    Write-Info "Setting up Windows service..."
    
    # Create service wrapper script
    $serviceScript = "$NODE_HOME\service.ps1"
    @"
# Volnix Protocol Service Wrapper
Set-Location "$NODE_HOME"
& "$env:USERPROFILE\bin\$BINARY_NAME" start --home "$NODE_HOME"
"@ | Out-File -FilePath $serviceScript -Encoding UTF8
    
    # Install NSSM (Non-Sucking Service Manager) if not present
    if (!(Get-Command nssm -ErrorAction SilentlyContinue)) {
        Write-Info "Installing NSSM for service management..."
        # In a real deployment, you would download and install NSSM
        Write-Warning "Please install NSSM manually to create Windows service"
        Write-Info "Download from: https://nssm.cc/download"
    } else {
        # Create service using NSSM
        & nssm install VolnixProtocol powershell.exe
        & nssm set VolnixProtocol Arguments "-ExecutionPolicy Bypass -File `"$serviceScript`""
        & nssm set VolnixProtocol DisplayName "Volnix Protocol Node"
        & nssm set VolnixProtocol Description "Volnix Protocol Blockchain Node"
        & nssm set VolnixProtocol Start SERVICE_AUTO_START
        
        Write-Success "Windows service configured"
    }
}

function Set-Monitoring {
    Write-Info "Setting up monitoring..."
    
    $monitoringDir = "$NODE_HOME\monitoring"
    if (!(Test-Path $monitoringDir)) {
        New-Item -ItemType Directory -Path $monitoringDir -Force | Out-Null
    }
    
    # Create monitoring script
    $monitorScript = "$monitoringDir\monitor.ps1"
    @"
# Simple monitoring script for Volnix Protocol node
try {
    `$status = Invoke-RestMethod -Uri "http://localhost:26657/status" -TimeoutSec 5
    `$catchingUp = `$status.result.sync_info.catching_up
    `$latestBlock = `$status.result.sync_info.latest_block_height
    
    Write-Output "Node Status: `$catchingUp"
    Write-Output "Latest Block: `$latestBlock"
}
catch {
    Write-Output "Node Status: Error - `$(`$_.Exception.Message)"
}

# Check if process is running
`$process = Get-Process -Name "volnixd" -ErrorAction SilentlyContinue
if (`$process) {
    Write-Output "Node Process: Running (PID: `$(`$process.Id))"
} else {
    Write-Output "Node Process: Not Running"
    # Restart service if available
    try {
        Restart-Service -Name "VolnixProtocol" -ErrorAction SilentlyContinue
        Write-Output "Service restart attempted"
    }
    catch {
        Write-Output "Service restart failed: `$(`$_.Exception.Message)"
    }
}
"@ | Out-File -FilePath $monitorScript -Encoding UTF8
    
    # Create scheduled task for monitoring
    $taskName = "VolnixProtocolMonitoring"
    $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -File `"$monitorScript`""
    $trigger = New-ScheduledTaskTrigger -RepetitionInterval (New-TimeSpan -Minutes 5) -Once -At (Get-Date)
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    
    try {
        Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Force | Out-Null
        Write-Success "Monitoring scheduled task created"
    }
    catch {
        Write-Warning "Failed to create monitoring scheduled task: $($_.Exception.Message)"
    }
}

function New-Validator {
    Write-Info "Setting up validator..."
    
    $validatorKeyPath = "$NODE_HOME\config\priv_validator_key.json"
    if (!(Test-Path $validatorKeyPath)) {
        Write-Error "Validator key not found. Node initialization may have failed."
        return
    }
    
    # Get validator public key
    try {
        $validatorPubkey = & $BINARY_NAME tendermint show-validator --home $NODE_HOME
        Write-Info "Validator public key: $validatorPubkey"
    }
    catch {
        Write-Error "Failed to get validator public key"
        return
    }
    
    # Create validator transaction template
    $validatorTemplate = "$NODE_HOME\create-validator.json"
    @"
{
  "pubkey": $validatorPubkey,
  "amount": "1000000ant",
  "moniker": "$Moniker",
  "identity": "",
  "website": "",
  "security_contact": "",
  "details": "Volnix Protocol Validator",
  "commission-rate": "0.10",
  "commission-max-rate": "0.20",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
"@ | Out-File -FilePath $validatorTemplate -Encoding UTF8
    
    Write-Success "Validator setup template created at $validatorTemplate"
    Write-Info "To create validator, run: $BINARY_NAME tx staking create-validator $validatorTemplate --from <key-name> --chain-id $ChainId"
}

function Set-Firewall {
    Write-Info "Configuring Windows Firewall..."
    
    try {
        # Allow P2P port
        New-NetFirewallRule -DisplayName "Volnix P2P" -Direction Inbound -Protocol TCP -LocalPort 26656 -Action Allow -ErrorAction SilentlyContinue
        
        # Allow RPC port (optional)
        if ($EnableRPC) {
            New-NetFirewallRule -DisplayName "Volnix RPC" -Direction Inbound -Protocol TCP -LocalPort 26657 -Action Allow -ErrorAction SilentlyContinue
        }
        
        # Allow monitoring port
        if ($EnableMonitoring) {
            New-NetFirewallRule -DisplayName "Volnix Monitoring" -Direction Inbound -Protocol TCP -LocalPort 8080 -Action Allow -ErrorAction SilentlyContinue
        }
        
        Write-Success "Windows Firewall configured"
    }
    catch {
        Write-Warning "Failed to configure Windows Firewall: $($_.Exception.Message)"
    }
}

function Start-Node {
    Write-Info "Starting Volnix Protocol node..."
    
    # Try to start as service first
    try {
        Start-Service -Name "VolnixProtocol" -ErrorAction Stop
        Write-Success "Node service started successfully"
        
        # Wait for startup
        Start-Sleep -Seconds 10
        
        # Check sync status
        try {
            $status = Invoke-RestMethod -Uri "http://localhost:26657/status" -TimeoutSec 10
            $catchingUp = $status.result.sync_info.catching_up
            
            if ($catchingUp -eq $false) {
                Write-Success "Node is synced"
            } elseif ($catchingUp -eq $true) {
                Write-Info "Node is syncing..."
            }
        }
        catch {
            Write-Warning "Unable to determine sync status"
        }
    }
    catch {
        Write-Warning "Service not available, starting manually..."
        
        # Start manually in background
        $job = Start-Job -ScriptBlock {
            param($BinaryPath, $NodeHome)
            Set-Location $NodeHome
            & $BinaryPath start --home $NodeHome
        } -ArgumentList "$env:USERPROFILE\bin\$BINARY_NAME", $NODE_HOME
        
        Write-Info "Node started as background job (ID: $($job.Id))"
        Write-Info "Use 'Get-Job' to check status and 'Stop-Job $($job.Id)' to stop"
    }
}

function Show-Summary {
    Write-ColorOutput @"
╔══════════════════════════════════════════════════════════════╗
║                    Deployment Completed!                    ║
╚══════════════════════════════════════════════════════════════╝
"@ "Green"
    
    Write-Host ""
    Write-Host "📋 Deployment Summary:" -ForegroundColor White
    Write-Host "  🏠 Node Home: $NODE_HOME" -ForegroundColor Gray
    Write-Host "  🔗 Chain ID: $ChainId" -ForegroundColor Gray
    Write-Host "  🏷️  Moniker: $Moniker" -ForegroundColor Gray
    Write-Host "  📊 Version: $VOLNIX_VERSION" -ForegroundColor Gray
    Write-Host ""
    Write-Host "🔧 Useful Commands:" -ForegroundColor White
    Write-Host "  📊 Check status: Get-Service VolnixProtocol" -ForegroundColor Gray
    Write-Host "  🔄 Restart node: Restart-Service VolnixProtocol" -ForegroundColor Gray
    Write-Host "  ⏹️  Stop node: Stop-Service VolnixProtocol" -ForegroundColor Gray
    Write-Host "  📜 View jobs: Get-Job" -ForegroundColor Gray
    Write-Host ""
    Write-Host "🌐 Endpoints:" -ForegroundColor White
    Write-Host "  🔗 RPC: http://localhost:26657" -ForegroundColor Gray
    Write-Host "  📡 API: http://localhost:1317" -ForegroundColor Gray
    Write-Host "  📊 Monitoring: http://localhost:8080" -ForegroundColor Gray
    Write-Host ""
    Write-Host "📁 Important Files:" -ForegroundColor White
    Write-Host "  ⚙️  Config: $NODE_HOME\config\config.toml" -ForegroundColor Gray
    Write-Host "  🌱 Genesis: $NODE_HOME\config\genesis.json" -ForegroundColor Gray
    Write-Host "  🔑 Validator Key: $NODE_HOME\config\priv_validator_key.json" -ForegroundColor Gray
    Write-Host "  📊 Monitoring: $NODE_HOME\monitoring\" -ForegroundColor Gray
    Write-Host ""
    Write-Host "🚀 Next Steps:" -ForegroundColor White
    Write-Host "  1. Wait for node to sync" -ForegroundColor Gray
    Write-Host "  2. Create a wallet: $BINARY_NAME keys add <wallet-name>" -ForegroundColor Gray
    Write-Host "  3. Get tokens from faucet or exchange" -ForegroundColor Gray
    Write-Host "  4. Create validator using the template" -ForegroundColor Gray
    Write-Host ""
    Write-Success "Volnix Protocol node deployment completed successfully!"
}

function Show-Help {
    Write-Host "Volnix Protocol Deployment Script for Windows" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\deploy.ps1 [OPTIONS]" -ForegroundColor White
    Write-Host ""
    Write-Host "Options:" -ForegroundColor White
    Write-Host "  -Moniker <name>         Set node moniker (default: computer name)" -ForegroundColor Gray
    Write-Host "  -ChainId <id>           Set chain ID (default: volnix-1)" -ForegroundColor Gray
    Write-Host "  -EnableStateSync        Enable state sync" -ForegroundColor Gray
    Write-Host "  -EnableRPC              Enable RPC access" -ForegroundColor Gray
    Write-Host "  -EnableMonitoring       Enable monitoring" -ForegroundColor Gray
    Write-Host "  -SkipBuild              Skip binary build" -ForegroundColor Gray
    Write-Host "  -Help                   Show this help" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor White
    Write-Host "  .\deploy.ps1 -Moniker 'MyValidator' -EnableMonitoring" -ForegroundColor Gray
    Write-Host "  .\deploy.ps1 -ChainId 'volnix-testnet-1' -EnableRPC" -ForegroundColor Gray
}

# Main deployment function
function Main {
    if ($Help) {
        Show-Help
        return
    }
    
    Show-Banner
    
    # Run deployment steps
    Test-Requirements
    
    if (!$SkipBuild) {
        Install-Binary
    }
    
    Initialize-Node
    Get-Genesis
    Set-NodeConfiguration
    Install-WindowsService
    
    if ($EnableMonitoring) {
        Set-Monitoring
    }
    
    Set-Firewall
    New-Validator
    Start-Node
    Show-Summary
}

# Run main function
try {
    Main
}
catch {
    Write-Error "Deployment failed: $($_.Exception.Message)"
    Write-Error $_.ScriptStackTrace
    exit 1
}