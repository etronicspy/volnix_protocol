# Simple Volnix Wallet
Write-Host "Volnix Protocol Wallet" -ForegroundColor Cyan
Write-Host "=====================" -ForegroundColor Cyan

# Create wallets directory
$walletDir = ".volnix/wallets"
if (-not (Test-Path $walletDir)) {
    New-Item -ItemType Directory -Path $walletDir -Force | Out-Null
}

# Function to create wallet
function New-SimpleWallet($name) {
    $address = "volnix1" + (-join ((1..39) | ForEach-Object { Get-Random -InputObject @('a'..'z' + '2'..'9') }))
    
    $wallet = @{
        name = $name
        address = $address
        balance = @{
            vx = 1000
            lzn = 500
            ant = 100
        }
    }
    
    $walletFile = "$walletDir/$name.json"
    $wallet | ConvertTo-Json | Out-File -FilePath $walletFile -Encoding UTF8
    
    Write-Host "Created wallet: $name" -ForegroundColor Green
    Write-Host "Address: $address" -ForegroundColor Cyan
    Write-Host "Initial balance: 1000 VX, 500 LZN, 100 ANT" -ForegroundColor Yellow
    Write-Host ""
    
    return $wallet
}

# Function to list wallets
function Get-SimpleWallets {
    Write-Host "Available Wallets:" -ForegroundColor Yellow
    
    $wallets = Get-ChildItem -Path $walletDir -Filter "*.json" -ErrorAction SilentlyContinue
    
    foreach ($walletFile in $wallets) {
        $wallet = Get-Content $walletFile.FullName | ConvertFrom-Json
        Write-Host "Name: $($wallet.name)" -ForegroundColor Cyan
        Write-Host "Address: $($wallet.address)" -ForegroundColor White
        Write-Host "Balance: $($wallet.balance.vx) VX, $($wallet.balance.lzn) LZN, $($wallet.balance.ant) ANT" -ForegroundColor Green
        Write-Host ""
    }
}

# Function to send transaction
function Send-SimpleTransaction($from, $to, $amount) {
    $fromFile = "$walletDir/$from.json"
    if (-not (Test-Path $fromFile)) {
        Write-Host "Sender wallet not found: $from" -ForegroundColor Red
        return
    }
    
    $fromWallet = Get-Content $fromFile | ConvertFrom-Json
    
    if ($fromWallet.balance.vx -lt $amount) {
        Write-Host "Insufficient balance" -ForegroundColor Red
        return
    }
    
    # Update sender balance
    $fromWallet.balance.vx = $fromWallet.balance.vx - $amount
    $fromWallet | ConvertTo-Json | Out-File -FilePath $fromFile -Encoding UTF8
    
    # Generate transaction hash
    $txHash = "0x" + (-join ((1..8) | ForEach-Object { Get-Random -InputObject @('a'..'f' + '0'..'9') }))
    
    Write-Host "Transaction sent!" -ForegroundColor Green
    Write-Host "Hash: $txHash" -ForegroundColor Cyan
    Write-Host "From: $($fromWallet.address)" -ForegroundColor White
    Write-Host "To: $to" -ForegroundColor White
    Write-Host "Amount: $amount VX" -ForegroundColor Yellow
    Write-Host ""
}

# Create test wallets
Write-Host "Creating test wallets..." -ForegroundColor Yellow
New-SimpleWallet "alice" | Out-Null
New-SimpleWallet "bob" | Out-Null
New-SimpleWallet "charlie" | Out-Null

# Show wallets
Get-SimpleWallets

# Demo transactions
Write-Host "Sending demo transactions..." -ForegroundColor Yellow
Write-Host ""

$aliceWallet = Get-Content "$walletDir/alice.json" | ConvertFrom-Json
$bobWallet = Get-Content "$walletDir/bob.json" | ConvertFrom-Json

Write-Host "Alice sends 100 VX to Bob..." -ForegroundColor Cyan
Send-SimpleTransaction "alice" $bobWallet.address 100

Write-Host "Bob sends 50 VX to Charlie..." -ForegroundColor Cyan
$charlieWallet = Get-Content "$walletDir/charlie.json" | ConvertFrom-Json
Send-SimpleTransaction "bob" $charlieWallet.address 50

Write-Host "Final balances:" -ForegroundColor Yellow
Get-SimpleWallets

Write-Host "Wallet demo completed!" -ForegroundColor Green
Write-Host "You can now send transactions between wallets." -ForegroundColor Cyan