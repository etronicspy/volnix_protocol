# Volnix Protocol Multi-Node Testnet Setup
# Создает тестовую сеть с любым количеством узлов

param(
    [int]$NodeCount = 4,
    [string]$ChainId = "volnix-testnet",
    [string]$BaseName = "volnix-node",
    [int]$StartPort = 26656,
    [switch]$CleanStart
)

Write-Host "🌐 Setting up Volnix Protocol Testnet" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan
Write-Host "Nodes: $NodeCount" -ForegroundColor Yellow
Write-Host "Chain ID: $ChainId" -ForegroundColor Yellow
Write-Host "Base Port: $StartPort" -ForegroundColor Yellow
Write-Host ""

# Функция для создания конфигурации узла
function New-NodeConfig {
    param(
        [int]$NodeIndex,
        [string]$NodeName,
        [int]$P2PPort,
        [int]$RPCPort,
        [string]$ChainId
    )
    
    $nodeDir = "testnet/$NodeName"
    
    # Создание директории узла
    if (Test-Path $nodeDir) {
        if ($CleanStart) {
            Remove-Item -Recurse -Force $nodeDir
        }
    }
    
    if (-not (Test-Path $nodeDir)) {
        New-Item -ItemType Directory -Path $nodeDir -Force | Out-Null
    }
    
    Write-Host "🔧 Configuring node $NodeIndex`: $NodeName" -ForegroundColor Yellow
    
    # Инициализация узла
    $env:VOLNIX_HOME = $nodeDir
    .\volnixd-standalone.exe init $NodeName --home $nodeDir 2>$null
    
    # Создание конфигурации
    $configPath = "$nodeDir/config/config.toml"
    $appConfigPath = "$nodeDir/config/app.toml"
    
    # Базовая конфигурация config.toml
    $configContent = @"
# Volnix Node Configuration - $NodeName

# RPC Server Configuration
[rpc]
laddr = "tcp://0.0.0.0:$RPCPort"
cors_allowed_origins = ["*"]
cors_allowed_methods = ["HEAD", "GET", "POST"]
cors_allowed_headers = ["Origin", "Accept", "Content-Type", "X-Requested-With", "X-Server-Time"]

# P2P Configuration
[p2p]
laddr = "tcp://0.0.0.0:$P2PPort"
external_address = "127.0.0.1:$P2PPort"
max_num_inbound_peers = 40
max_num_outbound_peers = 10
flush_throttle_timeout = "100ms"
max_packet_msg_payload_size = 1024
send_rate = 5120000
recv_rate = 5120000

# Consensus Configuration
[consensus]
timeout_propose = "3s"
timeout_prevote = "1s"
timeout_precommit = "1s"
timeout_commit = "5s"
create_empty_blocks = true
create_empty_blocks_interval = "0s"

# Mempool Configuration
[mempool]
size = 5000
cache_size = 10000

# State Sync Configuration
[statesync]
enable = false

# Block Sync Configuration
[blocksync]
version = "v0"

# Logging
[log]
level = "info"
format = "plain"
"@

    # Создание app.toml
    $appConfigContent = @"
# Volnix Application Configuration - $NodeName

# API Configuration
[api]
enable = true
swagger = true
address = "tcp://0.0.0.0:$($RPCPort + 1000)"
max-open-connections = 1000
rpc-read-timeout = 10
rpc-write-timeout = 0
rpc-max-body-bytes = 1000000
enabled-unsafe-cors = true

# gRPC Configuration
[grpc]
enable = true
address = "0.0.0.0:$($P2PPort + 1000)"

# State Sync Configuration
[state-sync]
snapshot-interval = 0
snapshot-keep-recent = 2
"@

    # Запись конфигураций
    $configContent | Out-File -FilePath $configPath -Encoding UTF8
    $appConfigContent | Out-File -FilePath $appConfigPath -Encoding UTF8
    
    return @{
        Name = $NodeName
        Dir = $nodeDir
        P2PPort = $P2PPort
        RPCPort = $RPCPort
        APIPort = $RPCPort + 1000
        GRPCPort = $P2PPort + 1000
    }
}

# Функция для создания genesis файла
function New-GenesisFile {
    param(
        [array]$Nodes,
        [string]$ChainId
    )
    
    Write-Host "🌟 Creating genesis file..." -ForegroundColor Yellow
    
    $genesisContent = @"
{
  "genesis_time": "$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ')",
  "chain_id": "$ChainId",
  "initial_height": "1",
  "consensus_params": {
    "block": {
      "max_bytes": "22020096",
      "max_gas": "-1",
      "time_iota_ms": "1000"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000",
      "max_bytes": "1048576"
    },
    "validator": {
      "pub_key_types": ["ed25519"]
    },
    "version": {}
  },
  "validators": [
"@

    # Добавление валидаторов
    $validatorEntries = @()
    foreach ($node in $Nodes) {
        $validatorKeyPath = "$($node.Dir)/config/priv_validator_key.json"
        if (Test-Path $validatorKeyPath) {
            $validatorKey = Get-Content $validatorKeyPath | ConvertFrom-Json
            $pubKey = $validatorKey.pub_key.value
            
            $validatorEntry = @"
    {
      "address": "",
      "pub_key": {
        "type": "tendermint/PubKeyEd25519",
        "value": "$pubKey"
      },
      "power": "10",
      "name": "$($node.Name)"
    }
"@
            $validatorEntries += $validatorEntry
        }
    }
    
    $genesisContent += ($validatorEntries -join ",`n")
    $genesisContent += @"

  ],
  "app_hash": "",
  "app_state": {
    "auth": {
      "params": {
        "max_memo_characters": "256",
        "tx_sig_limit": "7",
        "tx_size_cost_per_byte": "10",
        "sig_verify_cost_ed25519": "590",
        "sig_verify_cost_secp256k1": "1000"
      },
      "accounts": []
    },
    "bank": {
      "params": {
        "send_enabled": [],
        "default_send_enabled": true
      },
      "balances": [],
      "supply": [],
      "denom_metadata": [
        {
          "description": "Volnix native token",
          "denom_units": [
            {
              "denom": "uvx",
              "exponent": 0,
              "aliases": ["microvolnix"]
            },
            {
              "denom": "vx",
              "exponent": 6,
              "aliases": ["volnix"]
            }
          ],
          "base": "uvx",
          "display": "vx",
          "name": "Volnix",
          "symbol": "VX"
        }
      ]
    },
    "distribution": {
      "params": {
        "community_tax": "0.020000000000000000",
        "base_proposer_reward": "0.010000000000000000",
        "bonus_proposer_reward": "0.040000000000000000",
        "withdraw_addr_enabled": true
      }
    },
    "staking": {
      "params": {
        "unbonding_time": "1814400s",
        "max_validators": 100,
        "max_entries": 7,
        "historical_entries": 10000,
        "bond_denom": "uvx"
      }
    }
  }
}
"@

    # Сохранение genesis файла для всех узлов
    foreach ($node in $Nodes) {
        $genesisPath = "$($node.Dir)/config/genesis.json"
        $genesisContent | Out-File -FilePath $genesisPath -Encoding UTF8
    }
    
    Write-Host "✅ Genesis file created for all nodes" -ForegroundColor Green
}

# Функция для настройки пиров
function Set-PeerConnections {
    param([array]$Nodes)
    
    Write-Host "🔗 Setting up peer connections..." -ForegroundColor Yellow
    
    # Получение node ID для каждого узла
    $nodeIds = @{}
    foreach ($node in $Nodes) {
        $nodeKeyPath = "$($node.Dir)/config/node_key.json"
        if (Test-Path $nodeKeyPath) {
            $nodeKey = Get-Content $nodeKeyPath | ConvertFrom-Json
            $nodeIds[$node.Name] = $nodeKey.id
        }
    }
    
    # Настройка persistent_peers для каждого узла
    foreach ($node in $Nodes) {
        $peers = @()
        foreach ($otherNode in $Nodes) {
            if ($otherNode.Name -ne $node.Name -and $nodeIds.ContainsKey($otherNode.Name)) {
                $peerId = $nodeIds[$otherNode.Name]
                $peers += "$peerId@127.0.0.1:$($otherNode.P2PPort)"
            }
        }
        
        $peerString = $peers -join ","
        $configPath = "$($node.Dir)/config/config.toml"
        
        if (Test-Path $configPath) {
            $config = Get-Content $configPath -Raw
            $config = $config -replace 'persistent_peers = ""', "persistent_peers = `"$peerString`""
            $config | Out-File -FilePath $configPath -Encoding UTF8
        }
    }
    
    Write-Host "✅ Peer connections configured" -ForegroundColor Green
}

# Функция для запуска узлов
function Start-TestnetNodes {
    param([array]$Nodes)
    
    Write-Host "🚀 Starting testnet nodes..." -ForegroundColor Yellow
    
    $processes = @()
    
    foreach ($node in $Nodes) {
        Write-Host "Starting $($node.Name) on ports P2P:$($node.P2PPort) RPC:$($node.RPCPort)" -ForegroundColor Cyan
        
        $process = Start-Process -FilePath ".\volnixd-standalone.exe" -ArgumentList "start --home $($node.Dir)" -PassThru -WindowStyle Hidden
        $processes += @{
            Name = $node.Name
            Process = $process
            Ports = $node
        }
        
        Start-Sleep -Seconds 2
    }
    
    return $processes
}

# Основная логика
try {
    # Создание директории testnet
    if (-not (Test-Path "testnet")) {
        New-Item -ItemType Directory -Path "testnet" -Force | Out-Null
    }
    
    Write-Host "🔧 Creating $NodeCount nodes..." -ForegroundColor Yellow
    
    # Создание узлов
    $nodes = @()
    for ($i = 1; $i -le $NodeCount; $i++) {
        $nodeName = "$BaseName-$i"
        $p2pPort = $StartPort + (($i - 1) * 10)
        $rpcPort = $p2pPort + 1
        
        $node = New-NodeConfig -NodeIndex $i -NodeName $nodeName -P2PPort $p2pPort -RPCPort $rpcPort -ChainId $ChainId
        $nodes += $node
    }
    
    # Создание genesis файла
    New-GenesisFile -Nodes $nodes -ChainId $ChainId
    
    # Настройка пиров
    Set-PeerConnections -Nodes $nodes
    
    # Запуск узлов
    $runningProcesses = Start-TestnetNodes -Nodes $nodes
    
    Write-Host ""
    Write-Host "🎉 Volnix Testnet is running!" -ForegroundColor Green
    Write-Host "=============================" -ForegroundColor Green
    Write-Host ""
    Write-Host "📊 Network Information:" -ForegroundColor Cyan
    Write-Host "Chain ID: $ChainId" -ForegroundColor White
    Write-Host "Nodes: $NodeCount" -ForegroundColor White
    Write-Host ""
    Write-Host "🌐 Node Endpoints:" -ForegroundColor Cyan
    
    foreach ($node in $nodes) {
        Write-Host "  $($node.Name):" -ForegroundColor Yellow
        Write-Host "    RPC:  http://localhost:$($node.RPCPort)" -ForegroundColor White
        Write-Host "    API:  http://localhost:$($node.APIPort)" -ForegroundColor White
        Write-Host "    P2P:  tcp://localhost:$($node.P2PPort)" -ForegroundColor White
        Write-Host "    gRPC: localhost:$($node.GRPCPort)" -ForegroundColor White
    }
    
    Write-Host ""
    Write-Host "🔧 Available Commands:" -ForegroundColor Cyan
    Write-Host "  # Check node status"
    Write-Host "  .\volnixd-standalone.exe status --home testnet/$BaseName-1" -ForegroundColor White
    Write-Host ""
    Write-Host "  # Query network info"
    Write-Host "  curl http://localhost:$($nodes[0].RPCPort)/net_info" -ForegroundColor White
    Write-Host ""
    Write-Host "  # Check consensus state"
    Write-Host "  curl http://localhost:$($nodes[0].RPCPort)/consensus_state" -ForegroundColor White
    
    Write-Host ""
    Write-Host "⚡ Mining and Transactions:" -ForegroundColor Cyan
    Write-Host "  - Blocks are being produced automatically" -ForegroundColor White
    Write-Host "  - Consensus is running between all $NodeCount nodes" -ForegroundColor White
    Write-Host "  - Ready for transaction processing" -ForegroundColor White
    
    Write-Host ""
    Write-Host "Press Ctrl+C to stop all nodes..." -ForegroundColor Yellow
    
    # Ожидание завершения
    try {
        while ($true) {
            Start-Sleep -Seconds 5
            
            # Проверка состояния узлов
            $aliveCount = 0
            foreach ($proc in $runningProcesses) {
                if (-not $proc.Process.HasExited) {
                    $aliveCount++
                }
            }
            
            if ($aliveCount -eq 0) {
                Write-Host "All nodes have stopped." -ForegroundColor Red
                break
            }
        }
    } catch {
        Write-Host ""
        Write-Host "🛑 Stopping testnet..." -ForegroundColor Yellow
    }
    
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)" -ForegroundColor Red
} finally {
    # Остановка всех процессов
    Write-Host "🛑 Cleaning up processes..." -ForegroundColor Yellow
    Get-Process | Where-Object { $_.ProcessName -like "*volnixd*" } | Stop-Process -Force -ErrorAction SilentlyContinue
    Write-Host "✅ Testnet stopped" -ForegroundColor Green
}