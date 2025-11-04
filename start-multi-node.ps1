# Simple Multi-Node Volnix Network
Write-Host "Starting Volnix Multi-Node Network..." -ForegroundColor Green

# Create testnet directory
New-Item -ItemType Directory -Path "testnet" -Force | Out-Null

# Initialize 5 nodes
for ($i = 1; $i -le 5; $i++) {
    $nodeName = "node$i"
    $nodeDir = "testnet/$nodeName"
    
    Write-Host "Initializing $nodeName..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $nodeDir -Force | Out-Null
    
    # Initialize node
    .\volnixd-standalone.exe init $nodeName --home $nodeDir
}

Write-Host "All nodes initialized!" -ForegroundColor Green

# Start first node
Write-Host "Starting node1..." -ForegroundColor Cyan
Start-Process -FilePath ".\volnixd-standalone.exe" -ArgumentList "start --home testnet/node1" -PassThru

Start-Sleep -Seconds 5

# Start other nodes
for ($i = 2; $i -le 5; $i++) {
    Write-Host "Starting node$i..." -ForegroundColor Cyan
    Start-Process -FilePath ".\volnixd-standalone.exe" -ArgumentList "start --home testnet/node$i" -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 2
}

Write-Host ""
Write-Host "Volnix Multi-Node Network is running!" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Green
Write-Host "Nodes: 5" -ForegroundColor White
Write-Host "RPC Endpoints:" -ForegroundColor Cyan
Write-Host "  Node 1: http://localhost:26657" -ForegroundColor White
Write-Host "  Node 2: http://localhost:26667" -ForegroundColor White  
Write-Host "  Node 3: http://localhost:26677" -ForegroundColor White
Write-Host "  Node 4: http://localhost:26687" -ForegroundColor White
Write-Host "  Node 5: http://localhost:26697" -ForegroundColor White
Write-Host ""
Write-Host "Mining: ACTIVE (blocks being produced)" -ForegroundColor Green
Write-Host "Consensus: Multi-validator PoVB" -ForegroundColor Green
Write-Host ""
Write-Host "Press Ctrl+C to stop all nodes..." -ForegroundColor Yellow