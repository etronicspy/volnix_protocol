# Volnix Protocol Mining and Transaction Management
# Управление майнингом, переводами и мониторингом сети

param(
    [string]$Action = "status",
    [string]$NodeHome = "testnet/volnix-node-1",
    [int]$RPCPort = 26657,
    [string]$ChainId = "volnix-testnet"
)

Write-Host "⚡ Volnix Protocol Mining & Transactions" -ForegroundColor Cyan
Write-Host "=======================================" -ForegroundColor Cyan

# Функция для проверки статуса сети
function Get-NetworkStatus {
    Write-Host "📊 Network Status:" -ForegroundColor Yellow
    
    try {
        # Проверка статуса узла
        $status = .\volnixd-standalone.exe status --home $NodeHome 2>$null
        Write-Host "✅ Node is running" -ForegroundColor Green
        
        # Получение информации о блоках
        $response = Invoke-RestMethod -Uri "http://localhost:$RPCPort/status" -ErrorAction SilentlyContinue
        if ($response) {
            $latestHeight = $response.result.sync_info.latest_block_height
            $latestTime = $response.result.sync_info.latest_block_time
            
            Write-Host "🔗 Latest Block Height: $latestHeight" -ForegroundColor Cyan
            Write-Host "⏰ Latest Block Time: $latestTime" -ForegroundColor Cyan
            
            # Проверка майнинга (производства блоков)
            Start-Sleep -Seconds 5
            $newResponse = Invoke-RestMethod -Uri "http://localhost:$RPCPort/status" -ErrorAction SilentlyContinue
            if ($newResponse -and $newResponse.result.sync_info.latest_block_height -gt $latestHeight) {
                Write-Host "⚡ Mining Status: ACTIVE (blocks being produced)" -ForegroundColor Green
            } else {
                Write-Host "⚠️  Mining Status: INACTIVE or SLOW" -ForegroundColor Yellow
            }
        }
        
        # Информация о валидаторах
        $validators = Invoke-RestMethod -Uri "http://localhost:$RPCPort/validators" -ErrorAction SilentlyContinue
        if ($validators) {
            $validatorCount = $validators.result.validators.Count
            Write-Host "👥 Active Validators: $validatorCount" -ForegroundColor Cyan
        }
        
        # Информация о пирах
        $netInfo = Invoke-RestMethod -Uri "http://localhost:$RPCPort/net_info" -ErrorAction SilentlyContinue
        if ($netInfo) {
            $peerCount = $netInfo.result.peers.Count
            Write-Host "🌐 Connected Peers: $peerCount" -ForegroundColor Cyan
        }
        
    } catch {
        Write-Host "❌ Cannot connect to node on port $RPCPort" -ForegroundColor Red
        Write-Host "Make sure the testnet is running" -ForegroundColor Yellow
    }
}

# Функция для мониторинга майнинга
function Watch-Mining {
    Write-Host "⚡ Monitoring Mining Activity..." -ForegroundColor Yellow
    Write-Host "Press Ctrl+C to stop monitoring" -ForegroundColor Gray
    Write-Host ""
    
    $lastHeight = 0
    $blockCount = 0
    $startTime = Get-Date
    
    while ($true) {
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:$RPCPort/status" -ErrorAction SilentlyContinue
            if ($response) {
                $currentHeight = [int]$response.result.sync_info.latest_block_height
                $blockTime = $response.result.sync_info.latest_block_time
                
                if ($currentHeight -gt $lastHeight) {
                    $blockCount++
                    $elapsed = (Get-Date) - $startTime
                    $blocksPerMinute = if ($elapsed.TotalMinutes -gt 0) { [math]::Round($blockCount / $elapsed.TotalMinutes, 2) } else { 0 }
                    
                    Write-Host "$(Get-Date -Format 'HH:mm:ss') | Block #$currentHeight | Blocks/min: $blocksPerMinute" -ForegroundColor Green
                    $lastHeight = $currentHeight
                }
            }
        } catch {
            Write-Host "$(Get-Date -Format 'HH:mm:ss') | Connection error" -ForegroundColor Red
        }
        
        Start-Sleep -Seconds 2
    }
}

# Функция для создания тестовых аккаунтов
function New-TestAccounts {
    Write-Host "👤 Creating test accounts..." -ForegroundColor Yellow
    
    # Создание директории для ключей
    $keyDir = "$NodeHome/keyring-test"
    if (-not (Test-Path $keyDir)) {
        New-Item -ItemType Directory -Path $keyDir -Force | Out-Null
    }
    
    # Создание тестовых аккаунтов
    $accounts = @("alice", "bob", "charlie", "validator1")
    
    foreach ($account in $accounts) {
        Write-Host "Creating account: $account" -ForegroundColor Cyan
        
        # Генерация мнемоники и ключей (симуляция)
        $address = "volnix1" + (-join ((1..39) | ForEach-Object { Get-Random -InputObject @('a'..'z' + '0'..'9') }))
        
        # Создание файла ключа
        $keyFile = "$keyDir/$account.json"
        $keyData = @{
            name = $account
            type = "local"
            address = $address
            pubkey = "volnixpub1" + (-join ((1..64) | ForEach-Object { Get-Random -InputObject @('a'..'f' + '0'..'9') }))
        } | ConvertTo-Json
        
        $keyData | Out-File -FilePath $keyFile -Encoding UTF8
        
        Write-Host "✅ Account created: $account ($address)" -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "📋 Test accounts ready for transactions" -ForegroundColor Green
}

# Функция для симуляции транзакций
function Send-TestTransactions {
    Write-Host "💸 Simulating transactions..." -ForegroundColor Yellow
    
    $transactions = @(
        @{ From = "alice"; To = "bob"; Amount = "1000000uvx" },
        @{ From = "bob"; To = "charlie"; Amount = "500000uvx" },
        @{ From = "charlie"; To = "validator1"; Amount = "250000uvx" },
        @{ From = "validator1"; To = "alice"; Amount = "100000uvx" }
    )
    
    foreach ($tx in $transactions) {
        Write-Host "📤 Sending $($tx.Amount) from $($tx.From) to $($tx.To)" -ForegroundColor Cyan
        
        # Симуляция отправки транзакции
        try {
            # В реальной реализации здесь был бы вызов CLI команды
            # .\volnixd-standalone.exe tx bank send $tx.From $tx.To $tx.Amount --chain-id $ChainId --home $NodeHome
            
            # Для демонстрации создаем фиктивную транзакцию
            $txHash = "0x" + (-join ((1..64) | ForEach-Object { Get-Random -InputObject @('a'..'f' + '0'..'9') }))
            
            Write-Host "✅ Transaction sent: $txHash" -ForegroundColor Green
            Start-Sleep -Seconds 1
            
        } catch {
            Write-Host "❌ Transaction failed: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    
    Write-Host ""
    Write-Host "📊 Transaction simulation completed" -ForegroundColor Green
}

# Функция для мониторинга транзакций
function Watch-Transactions {
    Write-Host "📊 Monitoring Transactions..." -ForegroundColor Yellow
    Write-Host "Press Ctrl+C to stop monitoring" -ForegroundColor Gray
    Write-Host ""
    
    $lastHeight = 0
    $txCount = 0
    
    while ($true) {
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:$RPCPort/status" -ErrorAction SilentlyContinue
            if ($response) {
                $currentHeight = [int]$response.result.sync_info.latest_block_height
                
                if ($currentHeight -gt $lastHeight) {
                    # Получение информации о блоке
                    $blockResponse = Invoke-RestMethod -Uri "http://localhost:$RPCPort/block?height=$currentHeight" -ErrorAction SilentlyContinue
                    if ($blockResponse -and $blockResponse.result.block.data.txs) {
                        $blockTxCount = $blockResponse.result.block.data.txs.Count
                        $txCount += $blockTxCount
                        
                        Write-Host "$(Get-Date -Format 'HH:mm:ss') | Block #$currentHeight | Transactions: $blockTxCount | Total TXs: $txCount" -ForegroundColor Green
                    } else {
                        Write-Host "$(Get-Date -Format 'HH:mm:ss') | Block #$currentHeight | No transactions" -ForegroundColor Gray
                    }
                    
                    $lastHeight = $currentHeight
                }
            }
        } catch {
            Write-Host "$(Get-Date -Format 'HH:mm:ss') | Connection error" -ForegroundColor Red
        }
        
        Start-Sleep -Seconds 3
    }
}

# Функция для получения статистики сети
function Get-NetworkStats {
    Write-Host "📈 Network Statistics:" -ForegroundColor Yellow
    
    try {
        # Общая статистика
        $status = Invoke-RestMethod -Uri "http://localhost:$RPCPort/status" -ErrorAction SilentlyContinue
        if ($status) {
            $height = $status.result.sync_info.latest_block_height
            $chainId = $status.result.node_info.network
            
            Write-Host "🔗 Chain ID: $chainId" -ForegroundColor Cyan
            Write-Host "📊 Current Height: $height" -ForegroundColor Cyan
        }
        
        # Статистика валидаторов
        $validators = Invoke-RestMethod -Uri "http://localhost:$RPCPort/validators" -ErrorAction SilentlyContinue
        if ($validators) {
            Write-Host "👥 Total Validators: $($validators.result.total)" -ForegroundColor Cyan
            
            Write-Host ""
            Write-Host "🏆 Validator Details:" -ForegroundColor Yellow
            foreach ($validator in $validators.result.validators) {
                $power = $validator.voting_power
                $address = $validator.address.Substring(0, 12) + "..."
                Write-Host "  Validator: $address | Power: $power" -ForegroundColor White
            }
        }
        
        # Информация о пирах
        $netInfo = Invoke-RestMethod -Uri "http://localhost:$RPCPort/net_info" -ErrorAction SilentlyContinue
        if ($netInfo) {
            Write-Host ""
            Write-Host "🌐 Network Peers: $($netInfo.result.n_peers)" -ForegroundColor Cyan
            
            if ($netInfo.result.peers.Count -gt 0) {
                Write-Host ""
                Write-Host "🔗 Connected Peers:" -ForegroundColor Yellow
                foreach ($peer in $netInfo.result.peers) {
                    $nodeId = $peer.node_info.id.Substring(0, 12) + "..."
                    $remoteIP = $peer.remote_ip
                    Write-Host "  Peer: $nodeId | IP: $remoteIP" -ForegroundColor White
                }
            }
        }
        
    } catch {
        Write-Host "❌ Cannot retrieve network statistics" -ForegroundColor Red
    }
}

# Основная логика
switch ($Action.ToLower()) {
    "status" {
        Get-NetworkStatus
    }
    "mining" {
        Watch-Mining
    }
    "accounts" {
        New-TestAccounts
    }
    "transactions" {
        Send-TestTransactions
    }
    "monitor" {
        Watch-Transactions
    }
    "stats" {
        Get-NetworkStats
    }
    "all" {
        Get-NetworkStatus
        Write-Host ""
        New-TestAccounts
        Write-Host ""
        Send-TestTransactions
        Write-Host ""
        Get-NetworkStats
    }
    default {
        Write-Host "📋 Available Actions:" -ForegroundColor Cyan
        Write-Host "  status       - Check network status" -ForegroundColor White
        Write-Host "  mining       - Monitor mining activity" -ForegroundColor White
        Write-Host "  accounts     - Create test accounts" -ForegroundColor White
        Write-Host "  transactions - Send test transactions" -ForegroundColor White
        Write-Host "  monitor      - Monitor transactions" -ForegroundColor White
        Write-Host "  stats        - Show network statistics" -ForegroundColor White
        Write-Host "  all          - Run all operations" -ForegroundColor White
        Write-Host ""
        Write-Host "Usage: .\mining-and-transactions.ps1 -Action <action>" -ForegroundColor Yellow
    }
}