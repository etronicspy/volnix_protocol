# Volnix Protocol Transaction Manager
# Управление кошельками, ключами и транзакциями

param(
    [string]$Action = "help",
    [string]$From = "",
    [string]$To = "",
    [string]$Amount = "",
    [string]$KeyName = "",
    [string]$NodeRPC = "http://localhost:26657"
)

Write-Host "💰 Volnix Protocol Transaction Manager" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

# Функция для создания кошелька
function New-Wallet {
    param([string]$Name)
    
    Write-Host "👛 Creating wallet: $Name" -ForegroundColor Yellow
    
    # Создание директории для кошельков
    $walletDir = ".volnix/wallets"
    if (-not (Test-Path $walletDir)) {
        New-Item -ItemType Directory -Path $walletDir -Force | Out-Null
    }
    
    # Генерация мнемоники (24 слова)
    $words = @(
        "abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
        "absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
        "acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual",
        "adapt", "add", "addict", "address", "adjust", "admit", "adult", "advance"
    )
    
    $mnemonic = @()
    for ($i = 0; $i -lt 24; $i++) {
        $mnemonic += $words | Get-Random
    }
    $mnemonicString = $mnemonic -join " "
    
    # Генерация адреса (симуляция)
    $address = "volnix1" + (-join ((1..39) | ForEach-Object { 
        Get-Random -InputObject @('a'..'z' + '2'..'9') 
    }))
    
    # Генерация приватного ключа (симуляция)
    $privateKey = -join ((1..64) | ForEach-Object { 
        Get-Random -InputObject @('a'..'f' + '0'..'9') 
    })
    
    # Создание файла кошелька
    $walletData = @{
        name = $Name
        address = $address
        mnemonic = $mnemonicString
        privateKey = $privateKey
        balance = @{
            uvx = "1000000000"  # 1000 VX начальный баланс для тестирования
            ulzn = "500000000"  # 500 LZN
            uant = "100000000"  # 100 ANT
        }
        created = (Get-Date).ToString()
    }
    
    $walletFile = "$walletDir/$Name.json"
    $walletData | ConvertTo-Json -Depth 3 | Out-File -FilePath $walletFile -Encoding UTF8
    
    Write-Host "✅ Wallet created successfully!" -ForegroundColor Green
    Write-Host "📍 Address: $address" -ForegroundColor Cyan
    Write-Host "🔑 Mnemonic: $mnemonicString" -ForegroundColor Yellow
    Write-Host "💾 Saved to: $walletFile" -ForegroundColor Gray
    Write-Host ""
    Write-Host "⚠️  IMPORTANT: Save your mnemonic phrase securely!" -ForegroundColor Red
    
    return $walletData
}

# Функция для получения списка кошельков
function Get-Wallets {
    Write-Host "👛 Available Wallets:" -ForegroundColor Yellow
    
    $walletDir = ".volnix/wallets"
    if (-not (Test-Path $walletDir)) {
        Write-Host "No wallets found. Create one with: -Action create -KeyName <name>" -ForegroundColor Gray
        return
    }
    
    $wallets = Get-ChildItem -Path $walletDir -Filter "*.json"
    if ($wallets.Count -eq 0) {
        Write-Host "No wallets found. Create one with: -Action create -KeyName <name>" -ForegroundColor Gray
        return
    }
    
    foreach ($walletFile in $wallets) {
        $wallet = Get-Content $walletFile.FullName | ConvertFrom-Json
        Write-Host ""
        Write-Host "📛 Name: $($wallet.name)" -ForegroundColor Cyan
        Write-Host "📍 Address: $($wallet.address)" -ForegroundColor White
        Write-Host "💰 Balances:" -ForegroundColor Yellow
        Write-Host "   VX:  $([math]::Round($wallet.balance.uvx / 1000000, 2)) VX" -ForegroundColor Green
        Write-Host "   LZN: $([math]::Round($wallet.balance.ulzn / 1000000, 2)) LZN" -ForegroundColor Green
        Write-Host "   ANT: $([math]::Round($wallet.balance.uant / 1000000, 2)) ANT" -ForegroundColor Green
    }
}

# Функция для получения баланса
function Get-Balance {
    param([string]$WalletName)
    
    $walletFile = ".volnix/wallets/$WalletName.json"
    if (-not (Test-Path $walletFile)) {
        Write-Host "❌ Wallet '$WalletName' not found" -ForegroundColor Red
        return
    }
    
    $wallet = Get-Content $walletFile | ConvertFrom-Json
    
    Write-Host "💰 Balance for $WalletName ($($wallet.address)):" -ForegroundColor Cyan
    Write-Host "   VX:  $([math]::Round($wallet.balance.uvx / 1000000, 2)) VX" -ForegroundColor Green
    Write-Host "   LZN: $([math]::Round($wallet.balance.ulzn / 1000000, 2)) LZN" -ForegroundColor Green
    Write-Host "   ANT: $([math]::Round($wallet.balance.uant / 1000000, 2)) ANT" -ForegroundColor Green
}

# Функция для отправки транзакции
function Send-Transaction {
    param(
        [string]$FromWallet,
        [string]$ToAddress,
        [string]$Amount,
        [string]$Denom = "uvx"
    )
    
    Write-Host "💸 Sending Transaction..." -ForegroundColor Yellow
    
    # Проверка отправителя
    $fromWalletFile = ".volnix/wallets/$FromWallet.json"
    if (-not (Test-Path $fromWalletFile)) {
        Write-Host "❌ Sender wallet '$FromWallet' not found" -ForegroundColor Red
        return
    }
    
    $fromWalletData = Get-Content $fromWalletFile | ConvertFrom-Json
    
    # Проверка баланса
    $currentBalance = [int64]$fromWalletData.balance.$Denom
    $sendAmount = [int64]$Amount
    
    if ($currentBalance -lt $sendAmount) {
        Write-Host "❌ Insufficient balance. Available: $currentBalance $Denom, Required: $sendAmount $Denom" -ForegroundColor Red
        return
    }
    
    # Генерация хеша транзакции
    $txHash = "0x" + (-join ((1..64) | ForEach-Object { 
        Get-Random -InputObject @('a'..'f' + '0'..'9') 
    }))
    
    # Создание транзакции
    $transaction = @{
        hash = $txHash
        from = $fromWalletData.address
        to = $ToAddress
        amount = $sendAmount
        denom = $Denom
        fee = 1000  # 0.001 VX комиссия
        timestamp = (Get-Date).ToString()
        status = "pending"
        block_height = Get-Random -Minimum 1000 -Maximum 9999
    }
    
    Write-Host "📤 Transaction Details:" -ForegroundColor Cyan
    Write-Host "   Hash: $txHash" -ForegroundColor White
    Write-Host "   From: $($fromWalletData.address)" -ForegroundColor White
    Write-Host "   To: $ToAddress" -ForegroundColor White
    Write-Host "   Amount: $sendAmount $Denom" -ForegroundColor White
    Write-Host "   Fee: 1000 uvx" -ForegroundColor White
    
    # Симуляция отправки в блокчейн
    Write-Host ""
    Write-Host "🔄 Broadcasting transaction..." -ForegroundColor Yellow
    Start-Sleep -Seconds 2
    
    # Обновление баланса отправителя
    $fromWalletData.balance.$Denom = [string]($currentBalance - $sendAmount - 1000)
    $fromWalletData | ConvertTo-Json -Depth 3 | Out-File -FilePath $fromWalletFile -Encoding UTF8
    
    # Проверка получателя (если это наш кошелек)
    $toWalletName = ""
    $walletDir = ".volnix/wallets"
    if (Test-Path $walletDir) {
        $wallets = Get-ChildItem -Path $walletDir -Filter "*.json"
        foreach ($walletFile in $wallets) {
            $wallet = Get-Content $walletFile.FullName | ConvertFrom-Json
            if ($wallet.address -eq $ToAddress) {
                $toWalletName = $wallet.name
                # Обновление баланса получателя
                $wallet.balance.$Denom = [string]([int64]$wallet.balance.$Denom + $sendAmount)
                $wallet | ConvertTo-Json -Depth 3 | Out-File -FilePath $walletFile.FullName -Encoding UTF8
                break
            }
        }
    }
    
    # Сохранение транзакции в историю
    $txDir = ".volnix/transactions"
    if (-not (Test-Path $txDir)) {
        New-Item -ItemType Directory -Path $txDir -Force | Out-Null
    }
    
    $txFile = "$txDir/$txHash.json"
    $transaction.status = "confirmed"
    $transaction | ConvertTo-Json -Depth 3 | Out-File -FilePath $txFile -Encoding UTF8
    
    Write-Host "✅ Transaction confirmed!" -ForegroundColor Green
    Write-Host "📊 Transaction hash: $txHash" -ForegroundColor Cyan
    Write-Host "🔗 Block height: $($transaction.block_height)" -ForegroundColor Cyan
    
    if ($toWalletName) {
        Write-Host "💰 Recipient wallet '$toWalletName' balance updated" -ForegroundColor Green
    }
    
    return $transaction
}

# Функция для просмотра истории транзакций
function Get-TransactionHistory {
    param([string]$WalletName = "")
    
    Write-Host "📊 Transaction History:" -ForegroundColor Yellow
    
    $txDir = ".volnix/transactions"
    if (-not (Test-Path $txDir)) {
        Write-Host "No transactions found" -ForegroundColor Gray
        return
    }
    
    $transactions = Get-ChildItem -Path $txDir -Filter "*.json" | Sort-Object LastWriteTime -Descending
    
    if ($transactions.Count -eq 0) {
        Write-Host "No transactions found" -ForegroundColor Gray
        return
    }
    
    # Получение адреса кошелька если указан
    $walletAddress = ""
    if ($WalletName) {
        $walletFile = ".volnix/wallets/$WalletName.json"
        if (Test-Path $walletFile) {
            $wallet = Get-Content $walletFile | ConvertFrom-Json
            $walletAddress = $wallet.address
        }
    }
    
    $count = 0
    foreach ($txFile in $transactions) {
        if ($count -ge 10) { break }  # Показать только последние 10
        
        $tx = Get-Content $txFile.FullName | ConvertFrom-Json
        
        # Фильтрация по кошельку если указан
        if ($walletAddress -and $tx.from -ne $walletAddress -and $tx.to -ne $walletAddress) {
            continue
        }
        
        $count++
        
        Write-Host ""
        Write-Host "🔗 Transaction #$count" -ForegroundColor Cyan
        Write-Host "   Hash: $($tx.hash)" -ForegroundColor White
        Write-Host "   From: $($tx.from)" -ForegroundColor White
        Write-Host "   To: $($tx.to)" -ForegroundColor White
        Write-Host "   Amount: $($tx.amount) $($tx.denom)" -ForegroundColor Green
        Write-Host "   Status: $($tx.status)" -ForegroundColor $(if ($tx.status -eq "confirmed") { "Green" } else { "Yellow" })
        Write-Host "   Time: $($tx.timestamp)" -ForegroundColor Gray
        Write-Host "   Block: $($tx.block_height)" -ForegroundColor Gray
    }
}

# Функция для создания тестовых кошельков
function New-TestWallets {
    Write-Host "🧪 Creating test wallets..." -ForegroundColor Yellow
    
    $testWallets = @("alice", "bob", "charlie", "validator1", "trader1")
    
    foreach ($walletName in $testWallets) {
        if (-not (Test-Path ".volnix/wallets/$walletName.json")) {
            New-Wallet -Name $walletName | Out-Null
            Write-Host "✅ Created wallet: $walletName" -ForegroundColor Green
        } else {
            Write-Host "⏭️ Wallet already exists: $walletName" -ForegroundColor Yellow
        }
    }
    
    Write-Host ""
    Write-Host "🎉 Test wallets ready!" -ForegroundColor Green
    Get-Wallets
}

# Функция для демонстрации транзакций
function Start-TransactionDemo {
    Write-Host "🎬 Starting transaction demo..." -ForegroundColor Yellow
    
    # Создание тестовых кошельков если их нет
    New-TestWallets
    
    Write-Host ""
    Write-Host "💸 Sending demo transactions..." -ForegroundColor Cyan
    
    # Демо транзакции
    $demoTxs = @(
        @{ From = "alice"; To = "bob"; Amount = "50000000"; Denom = "uvx" },
        @{ From = "bob"; To = "charlie"; Amount = "25000000"; Denom = "uvx" },
        @{ From = "charlie"; To = "validator1"; Amount = "10000000"; Denom = "ulzn" },
        @{ From = "validator1"; To = "trader1"; Amount = "5000000"; Denom = "uant" },
        @{ From = "trader1"; To = "alice"; Amount = "15000000"; Denom = "uvx" }
    )
    
    foreach ($tx in $demoTxs) {
        # Получение адреса получателя
        $toWalletFile = ".volnix/wallets/$($tx.To).json"
        if (Test-Path $toWalletFile) {
            $toWallet = Get-Content $toWalletFile | ConvertFrom-Json
            $toAddress = $toWallet.address
            
            Write-Host ""
            Write-Host "📤 $($tx.From) → $($tx.To): $($tx.Amount) $($tx.Denom)" -ForegroundColor Cyan
            Send-Transaction -FromWallet $tx.From -ToAddress $toAddress -Amount $tx.Amount -Denom $tx.Denom | Out-Null
            Start-Sleep -Seconds 1
        }
    }
    
    Write-Host ""
    Write-Host "🎉 Demo completed! Check balances and transaction history." -ForegroundColor Green
}

# Функция для создания genesis транзакций
function New-GenesisAccounts {
    Write-Host "🌟 Creating genesis accounts with initial balances..." -ForegroundColor Yellow
    
    $genesisAccounts = @(
        @{ Name = "genesis"; Balance = @{ uvx = "10000000000000"; ulzn = "5000000000000"; uant = "1000000000000" } },
        @{ Name = "faucet"; Balance = @{ uvx = "5000000000000"; ulzn = "2500000000000"; uant = "500000000000" } },
        @{ Name = "validator"; Balance = @{ uvx = "1000000000000"; ulzn = "500000000000"; uant = "100000000000" } }
    )
    
    foreach ($account in $genesisAccounts) {
        $walletFile = ".volnix/wallets/$($account.Name).json"
        if (-not (Test-Path $walletFile)) {
            $wallet = New-Wallet -Name $account.Name
            # Обновить баланс
            $wallet.balance = $account.Balance
            $wallet | ConvertTo-Json -Depth 3 | Out-File -FilePath $walletFile -Encoding UTF8
            Write-Host "✅ Genesis account created: $($account.Name)" -ForegroundColor Green
        }
    }
}

# Функция для получения средств из faucet
function Request-Faucet {
    param([string]$WalletName, [string]$Amount = "1000000000")
    
    Write-Host "🚰 Requesting funds from faucet..." -ForegroundColor Yellow
    
    $faucetFile = ".volnix/wallets/faucet.json"
    $walletFile = ".volnix/wallets/$WalletName.json"
    
    if (-not (Test-Path $faucetFile)) {
        Write-Host "❌ Faucet not found. Creating genesis accounts..." -ForegroundColor Red
        New-GenesisAccounts
    }
    
    if (-not (Test-Path $walletFile)) {
        Write-Host "❌ Wallet '$WalletName' not found" -ForegroundColor Red
        return
    }
    
    $faucet = Get-Content $faucetFile | ConvertFrom-Json
    $wallet = Get-Content $walletFile | ConvertFrom-Json
    
    # Проверка баланса faucet
    $faucetBalance = [int64]$faucet.balance.uvx
    $requestAmount = [int64]$Amount
    
    if ($faucetBalance -lt $requestAmount) {
        Write-Host "❌ Faucet has insufficient funds" -ForegroundColor Red
        return
    }
    
    # Перевод средств
    Send-Transaction -FromWallet "faucet" -ToAddress $wallet.address -Amount $Amount -Denom "uvx" | Out-Null
    
    Write-Host "✅ Faucet request completed!" -ForegroundColor Green
    Write-Host "💰 Received: $([math]::Round($requestAmount / 1000000, 2)) VX" -ForegroundColor Cyan
}

# Функция для стейкинга
function Stake-Tokens {
    param(
        [string]$WalletName,
        [string]$ValidatorAddress,
        [string]$Amount
    )
    
    Write-Host "🏛️ Staking tokens..." -ForegroundColor Yellow
    
    $walletFile = ".volnix/wallets/$WalletName.json"
    if (-not (Test-Path $walletFile)) {
        Write-Host "❌ Wallet '$WalletName' not found" -ForegroundColor Red
        return
    }
    
    $wallet = Get-Content $walletFile | ConvertFrom-Json
    $stakeAmount = [int64]$Amount
    $currentBalance = [int64]$wallet.balance.uvx
    
    if ($currentBalance -lt $stakeAmount) {
        Write-Host "❌ Insufficient balance for staking" -ForegroundColor Red
        return
    }
    
    # Создание стейкинг транзакции
    $txHash = "0x" + (-join ((1..64) | ForEach-Object { 
        Get-Random -InputObject @('a'..'f' + '0'..'9') 
    }))
    
    $stakeTransaction = @{
        hash = $txHash
        type = "stake"
        delegator = $wallet.address
        validator = $ValidatorAddress
        amount = $stakeAmount
        denom = "uvx"
        timestamp = (Get-Date).ToString()
        status = "confirmed"
    }
    
    # Обновление баланса
    $wallet.balance.uvx = [string]($currentBalance - $stakeAmount)
    $wallet | ConvertTo-Json -Depth 3 | Out-File -FilePath $walletFile -Encoding UTF8
    
    # Сохранение стейкинг транзакции
    $stakeDir = ".volnix/staking"
    if (-not (Test-Path $stakeDir)) {
        New-Item -ItemType Directory -Path $stakeDir -Force | Out-Null
    }
    
    $stakeFile = "$stakeDir/$txHash.json"
    $stakeTransaction | ConvertTo-Json -Depth 3 | Out-File -FilePath $stakeFile -Encoding UTF8
    
    Write-Host "✅ Tokens staked successfully!" -ForegroundColor Green
    Write-Host "📊 Staked: $([math]::Round($stakeAmount / 1000000, 2)) VX" -ForegroundColor Cyan
    Write-Host "🏛️ Validator: $ValidatorAddress" -ForegroundColor Cyan
}

# Функция для создания валидатора
function Create-Validator {
    param(
        [string]$WalletName,
        [string]$Moniker,
        [string]$SelfDelegation = "1000000000"
    )
    
    Write-Host "🏛️ Creating validator..." -ForegroundColor Yellow
    
    $walletFile = ".volnix/wallets/$WalletName.json"
    if (-not (Test-Path $walletFile)) {
        Write-Host "❌ Wallet '$WalletName' not found" -ForegroundColor Red
        return
    }
    
    $wallet = Get-Content $walletFile | ConvertFrom-Json
    
    # Создание валидатора
    $validator = @{
        operator_address = $wallet.address
        consensus_pubkey = "volnixvalconspub1" + (-join ((1..64) | ForEach-Object { 
            Get-Random -InputObject @('a'..'f' + '0'..'9') 
        }))
        moniker = $Moniker
        identity = ""
        website = ""
        security_contact = ""
        details = "Volnix Protocol Validator"
        commission_rate = "0.10"
        commission_max_rate = "0.20"
        commission_max_change_rate = "0.01"
        min_self_delegation = $SelfDelegation
        delegator_shares = $SelfDelegation
        status = "BOND_STATUS_BONDED"
        jailed = $false
        created = (Get-Date).ToString()
    }
    
    # Сохранение валидатора
    $validatorDir = ".volnix/validators"
    if (-not (Test-Path $validatorDir)) {
        New-Item -ItemType Directory -Path $validatorDir -Force | Out-Null
    }
    
    $validatorFile = "$validatorDir/$($wallet.address).json"
    $validator | ConvertTo-Json -Depth 3 | Out-File -FilePath $validatorFile -Encoding UTF8
    
    # Стейкинг self-delegation
    Stake-Tokens -WalletName $WalletName -ValidatorAddress $wallet.address -Amount $SelfDelegation
    
    Write-Host "✅ Validator created successfully!" -ForegroundColor Green
    Write-Host "🏛️ Moniker: $Moniker" -ForegroundColor Cyan
    Write-Host "📍 Address: $($wallet.address)" -ForegroundColor Cyan
}

# Функция для получения информации о валидаторах
function Get-Validators {
    Write-Host "🏛️ Active Validators:" -ForegroundColor Yellow
    
    $validatorDir = ".volnix/validators"
    if (-not (Test-Path $validatorDir)) {
        Write-Host "No validators found" -ForegroundColor Gray
        return
    }
    
    $validators = Get-ChildItem -Path $validatorDir -Filter "*.json"
    if ($validators.Count -eq 0) {
        Write-Host "No validators found" -ForegroundColor Gray
        return
    }
    
    foreach ($validatorFile in $validators) {
        $validator = Get-Content $validatorFile.FullName | ConvertFrom-Json
        
        Write-Host ""
        Write-Host "🏛️ Validator: $($validator.moniker)" -ForegroundColor Cyan
        Write-Host "   Address: $($validator.operator_address)" -ForegroundColor White
        Write-Host "   Commission: $($validator.commission_rate)" -ForegroundColor White
        Write-Host "   Status: $($validator.status)" -ForegroundColor $(if ($validator.status -eq "BOND_STATUS_BONDED") { "Green" } else { "Yellow" })
        Write-Host "   Jailed: $($validator.jailed)" -ForegroundColor $(if ($validator.jailed) { "Red" } else { "Green" })
    }
}

# Функция для создания предложения governance
function Create-Proposal {
    param(
        [string]$WalletName,
        [string]$Title,
        [string]$Description,
        [string]$Deposit = "10000000"
    )
    
    Write-Host "🗳️ Creating governance proposal..." -ForegroundColor Yellow
    
    $walletFile = ".volnix/wallets/$WalletName.json"
    if (-not (Test-Path $walletFile)) {
        Write-Host "❌ Wallet '$WalletName' not found" -ForegroundColor Red
        return
    }
    
    $wallet = Get-Content $walletFile | ConvertFrom-Json
    
    # Создание предложения
    $proposalId = Get-Random -Minimum 1 -Maximum 1000
    $proposal = @{
        proposal_id = $proposalId
        title = $Title
        description = $Description
        proposer = $wallet.address
        initial_deposit = $Deposit
        submit_time = (Get-Date).ToString()
        deposit_end_time = (Get-Date).AddDays(14).ToString()
        voting_start_time = (Get-Date).ToString()
        voting_end_time = (Get-Date).AddDays(14).ToString()
        status = "PROPOSAL_STATUS_VOTING_PERIOD"
        final_tally_result = @{
            yes = "0"
            abstain = "0"
            no = "0"
            no_with_veto = "0"
        }
    }
    
    # Сохранение предложения
    $proposalDir = ".volnix/proposals"
    if (-not (Test-Path $proposalDir)) {
        New-Item -ItemType Directory -Path $proposalDir -Force | Out-Null
    }
    
    $proposalFile = "$proposalDir/$proposalId.json"
    $proposal | ConvertTo-Json -Depth 3 | Out-File -FilePath $proposalFile -Encoding UTF8
    
    Write-Host "✅ Proposal created successfully!" -ForegroundColor Green
    Write-Host "🗳️ Proposal ID: $proposalId" -ForegroundColor Cyan
    Write-Host "📋 Title: $Title" -ForegroundColor Cyan
}

# Функция для голосования
function Vote-Proposal {
    param(
        [string]$WalletName,
        [int]$ProposalId,
        [string]$Option = "yes"
    )
    
    Write-Host "🗳️ Voting on proposal..." -ForegroundColor Yellow
    
    $walletFile = ".volnix/wallets/$WalletName.json"
    $proposalFile = ".volnix/proposals/$ProposalId.json"
    
    if (-not (Test-Path $walletFile)) {
        Write-Host "❌ Wallet '$WalletName' not found" -ForegroundColor Red
        return
    }
    
    if (-not (Test-Path $proposalFile)) {
        Write-Host "❌ Proposal $ProposalId not found" -ForegroundColor Red
        return
    }
    
    $wallet = Get-Content $walletFile | ConvertFrom-Json
    $proposal = Get-Content $proposalFile | ConvertFrom-Json
    
    # Создание голоса
    $vote = @{
        proposal_id = $ProposalId
        voter = $wallet.address
        option = $Option.ToUpper()
        timestamp = (Get-Date).ToString()
    }
    
    # Сохранение голоса
    $voteDir = ".volnix/votes"
    if (-not (Test-Path $voteDir)) {
        New-Item -ItemType Directory -Path $voteDir -Force | Out-Null
    }
    
    $voteFile = "$voteDir/$ProposalId-$($wallet.address).json"
    $vote | ConvertTo-Json -Depth 3 | Out-File -FilePath $voteFile -Encoding UTF8
    
    Write-Host "✅ Vote submitted successfully!" -ForegroundColor Green
    Write-Host "🗳️ Proposal: $ProposalId" -ForegroundColor Cyan
    Write-Host "✅ Vote: $Option" -ForegroundColor Cyan
}

# Основная логика
switch ($Action.ToLower()) {
    "create" {
        if (-not $KeyName) {
            Write-Host "❌ Please specify wallet name: -KeyName <name>" -ForegroundColor Red
        } else {
            New-Wallet -Name $KeyName
        }
    }
    "list" {
        Get-Wallets
    }
    "balance" {
        if (-not $KeyName) {
            Write-Host "❌ Please specify wallet name: -KeyName <name>" -ForegroundColor Red
        } else {
            Get-Balance -WalletName $KeyName
        }
    }
    "send" {
        if (-not $From -or -not $To -or -not $Amount) {
            Write-Host "❌ Please specify: -From <wallet> -To <address> -Amount <amount>" -ForegroundColor Red
        } else {
            Send-Transaction -FromWallet $From -ToAddress $To -Amount $Amount
        }
    }
    "history" {
        Get-TransactionHistory -WalletName $KeyName
    }
    "demo" {
        Start-TransactionDemo
    }
    "test" {
        New-TestWallets
    }
    "faucet" {
        if (-not $KeyName) {
            Write-Host "❌ Please specify wallet name: -KeyName <name>" -ForegroundColor Red
        } else {
            Request-Faucet -WalletName $KeyName -Amount $Amount
        }
    }
    "genesis" {
        New-GenesisAccounts
    }
    "stake" {
        if (-not $KeyName -or -not $To -or -not $Amount) {
            Write-Host "❌ Please specify: -KeyName <wallet> -To <validator> -Amount <amount>" -ForegroundColor Red
        } else {
            Stake-Tokens -WalletName $KeyName -ValidatorAddress $To -Amount $Amount
        }
    }
    "create-validator" {
        if (-not $KeyName) {
            Write-Host "❌ Please specify: -KeyName <wallet>" -ForegroundColor Red
        } else {
            $moniker = if ($From) { $From } else { "$KeyName-validator" }
            Create-Validator -WalletName $KeyName -Moniker $moniker -SelfDelegation $Amount
        }
    }
    "validators" {
        Get-Validators
    }
    "propose" {
        if (-not $KeyName) {
            Write-Host "❌ Please specify: -KeyName <wallet>" -ForegroundColor Red
        } else {
            $title = if ($From) { $From } else { "Test Proposal" }
            $description = if ($To) { $To } else { "Test governance proposal" }
            Create-Proposal -WalletName $KeyName -Title $title -Description $description -Deposit $Amount
        }
    }
    "vote" {
        if (-not $KeyName -or -not $Amount) {
            Write-Host "❌ Please specify: -KeyName <wallet> -Amount <proposal_id>" -ForegroundColor Red
        } else {
            $option = if ($From) { $From } else { "yes" }
            Vote-Proposal -WalletName $KeyName -ProposalId $Amount -Option $option
        }
    }
    default {
        Write-Host "📋 Volnix Transaction Manager Commands:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "👛 Wallet Management:" -ForegroundColor Yellow
        Write-Host "  create   -KeyName <name>                    Create new wallet" -ForegroundColor White
        Write-Host "  list                                        List all wallets" -ForegroundColor White
        Write-Host "  balance  -KeyName <name>                    Show wallet balance" -ForegroundColor White
        Write-Host "  faucet   -KeyName <name> [-Amount <amount>] Request funds from faucet" -ForegroundColor White
        Write-Host ""
        Write-Host "💸 Transactions:" -ForegroundColor Yellow
        Write-Host "  send     -From <wallet> -To <address> -Amount <amount>  Send transaction" -ForegroundColor White
        Write-Host "  history  [-KeyName <name>]                  Show transaction history" -ForegroundColor White
        Write-Host ""
        Write-Host "🏛️ Staking & Validators:" -ForegroundColor Yellow
        Write-Host "  stake           -KeyName <wallet> -To <validator> -Amount <amount>  Stake tokens" -ForegroundColor White
        Write-Host "  create-validator -KeyName <wallet> [-From <moniker>] [-Amount <self_delegation>]  Create validator" -ForegroundColor White
        Write-Host "  validators                                   List all validators" -ForegroundColor White
        Write-Host ""
        Write-Host "🗳️ Governance:" -ForegroundColor Yellow
        Write-Host "  propose  -KeyName <wallet> [-From <title>] [-To <description>] [-Amount <deposit>]  Create proposal" -ForegroundColor White
        Write-Host "  vote     -KeyName <wallet> -Amount <proposal_id> [-From <option>]  Vote on proposal" -ForegroundColor White
        Write-Host ""
        Write-Host "🧪 Testing:" -ForegroundColor Yellow
        Write-Host "  genesis                                      Create genesis accounts" -ForegroundColor White
        Write-Host "  test                                         Create test wallets" -ForegroundColor White
        Write-Host "  demo                                         Run transaction demo" -ForegroundColor White
        Write-Host ""
        Write-Host "📝 Examples:" -ForegroundColor Cyan
        Write-Host "  .\transaction-manager.ps1 -Action create -KeyName alice" -ForegroundColor Gray
        Write-Host "  .\transaction-manager.ps1 -Action faucet -KeyName alice" -ForegroundColor Gray
        Write-Host "  .\transaction-manager.ps1 -Action send -From alice -To volnix1abc... -Amount 1000000" -ForegroundColor Gray
        Write-Host "  .\transaction-manager.ps1 -Action stake -KeyName alice -To volnix1validator... -Amount 100000000" -ForegroundColor Gray
        Write-Host "  .\transaction-manager.ps1 -Action create-validator -KeyName alice -From 'My Validator'" -ForegroundColor Gray
        Write-Host "  .\transaction-manager.ps1 -Action demo" -ForegroundColor Gray
    }
}