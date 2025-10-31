#!/bin/bash

# Volnix Protocol Deployment Script
# This script automates the deployment of Volnix Protocol nodes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VOLNIX_VERSION="0.1.0-alpha"
CHAIN_ID="volnix-1"
NODE_HOME="$HOME/.volnix"
BINARY_NAME="volnixd"
GENESIS_URL="https://raw.githubusercontent.com/volnix-protocol/mainnet/main/genesis.json"
SEEDS="seed1.volnix.network:26656,seed2.volnix.network:26656"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_banner() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                    Volnix Protocol Deployment               ║"
    echo "║                         Version $VOLNIX_VERSION                        ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_requirements() {
    log_info "Checking system requirements..."
    
    # Check OS
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        log_success "Operating System: Linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        log_success "Operating System: macOS"
    else
        log_error "Unsupported operating system: $OSTYPE"
        exit 1
    fi
    
    # Check Go version
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_success "Go version: $GO_VERSION"
    else
        log_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi
    
    # Check available disk space (minimum 100GB)
    AVAILABLE_SPACE=$(df -BG "$HOME" | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$AVAILABLE_SPACE" -lt 100 ]; then
        log_warning "Available disk space: ${AVAILABLE_SPACE}GB (recommended: 100GB+)"
    else
        log_success "Available disk space: ${AVAILABLE_SPACE}GB"
    fi
    
    # Check RAM (minimum 8GB)
    TOTAL_RAM=$(free -g | awk 'NR==2{print $2}')
    if [ "$TOTAL_RAM" -lt 8 ]; then
        log_warning "Total RAM: ${TOTAL_RAM}GB (recommended: 8GB+)"
    else
        log_success "Total RAM: ${TOTAL_RAM}GB"
    fi
}

install_binary() {
    log_info "Installing Volnix Protocol binary..."
    
    # Check if binary already exists
    if command -v $BINARY_NAME &> /dev/null; then
        CURRENT_VERSION=$($BINARY_NAME version 2>/dev/null | grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+" || echo "unknown")
        log_info "Current version: $CURRENT_VERSION"
    fi
    
    # Build from source
    if [ -d "volnix-protocol" ]; then
        log_info "Using existing source code..."
        cd volnix-protocol
        git pull origin main
    else
        log_info "Cloning Volnix Protocol repository..."
        git clone https://github.com/volnix-protocol/volnix-protocol.git
        cd volnix-protocol
    fi
    
    log_info "Building binary..."
    make build
    
    # Install binary
    sudo cp build/$BINARY_NAME /usr/local/bin/
    sudo chmod +x /usr/local/bin/$BINARY_NAME
    
    log_success "Binary installed successfully"
    
    # Verify installation
    INSTALLED_VERSION=$($BINARY_NAME version 2>/dev/null | head -n1 || echo "Installation failed")
    log_success "Installed version: $INSTALLED_VERSION"
    
    cd ..
}

initialize_node() {
    log_info "Initializing Volnix Protocol node..."
    
    # Get moniker from user or use hostname
    if [ -z "$MONIKER" ]; then
        MONIKER=$(hostname)
        log_info "Using hostname as moniker: $MONIKER"
    fi
    
    # Initialize node
    $BINARY_NAME init "$MONIKER" --chain-id "$CHAIN_ID" --home "$NODE_HOME"
    
    log_success "Node initialized with moniker: $MONIKER"
}

download_genesis() {
    log_info "Downloading genesis file..."
    
    # Download genesis file
    if curl -s "$GENESIS_URL" -o "$NODE_HOME/config/genesis.json"; then
        log_success "Genesis file downloaded successfully"
    else
        log_warning "Failed to download genesis file from $GENESIS_URL"
        log_info "Using default genesis file..."
    fi
    
    # Verify genesis file
    GENESIS_HASH=$(sha256sum "$NODE_HOME/config/genesis.json" | awk '{print $1}')
    log_info "Genesis hash: $GENESIS_HASH"
}

configure_node() {
    log_info "Configuring node settings..."
    
    CONFIG_FILE="$NODE_HOME/config/config.toml"
    APP_FILE="$NODE_HOME/config/app.toml"
    
    # Configure P2P settings
    sed -i "s/seeds = \"\"/seeds = \"$SEEDS\"/" "$CONFIG_FILE"
    sed -i 's/max_num_inbound_peers = 40/max_num_inbound_peers = 100/' "$CONFIG_FILE"
    sed -i 's/max_num_outbound_peers = 10/max_num_outbound_peers = 50/' "$CONFIG_FILE"
    
    # Configure consensus settings
    sed -i 's/timeout_commit = "5s"/timeout_commit = "3s"/' "$CONFIG_FILE"
    sed -i 's/timeout_propose = "3s"/timeout_propose = "2s"/' "$CONFIG_FILE"
    
    # Configure pruning (keep last 100,000 blocks)
    sed -i 's/pruning = "default"/pruning = "custom"/' "$APP_FILE"
    sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "100000"/' "$APP_FILE"
    sed -i 's/pruning-interval = "0"/pruning-interval = "10"/' "$APP_FILE"
    
    # Configure state sync (if enabled)
    if [ "$ENABLE_STATE_SYNC" = "true" ]; then
        log_info "Configuring state sync..."
        # State sync configuration would go here
    fi
    
    log_success "Node configuration completed"
}

setup_systemd() {
    log_info "Setting up systemd service..."
    
    # Create systemd service file
    sudo tee /etc/systemd/system/volnixd.service > /dev/null <<EOF
[Unit]
Description=Volnix Protocol Node
After=network-online.target

[Service]
User=$USER
ExecStart=/usr/local/bin/volnixd start --home $NODE_HOME
Restart=on-failure
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
    
    # Reload systemd and enable service
    sudo systemctl daemon-reload
    sudo systemctl enable volnixd
    
    log_success "Systemd service configured"
}

setup_monitoring() {
    log_info "Setting up monitoring..."
    
    # Create monitoring configuration
    mkdir -p "$NODE_HOME/monitoring"
    
    # Prometheus configuration
    tee "$NODE_HOME/monitoring/prometheus.yml" > /dev/null <<EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'volnix-node'
    static_configs:
      - targets: ['localhost:26660']
  - job_name: 'volnix-app'
    static_configs:
      - targets: ['localhost:8080']
EOF
    
    # Create monitoring script
    tee "$NODE_HOME/monitoring/monitor.sh" > /dev/null <<EOF
#!/bin/bash
# Simple monitoring script for Volnix Protocol node

NODE_STATUS=\$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.catching_up')
LATEST_BLOCK=\$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')

echo "Node Status: \$NODE_STATUS"
echo "Latest Block: \$LATEST_BLOCK"

# Check if node is running
if pgrep -x "volnixd" > /dev/null; then
    echo "Node Process: Running"
else
    echo "Node Process: Not Running"
    # Restart if not running
    sudo systemctl restart volnixd
fi
EOF
    
    chmod +x "$NODE_HOME/monitoring/monitor.sh"
    
    # Setup cron job for monitoring
    (crontab -l 2>/dev/null; echo "*/5 * * * * $NODE_HOME/monitoring/monitor.sh >> $NODE_HOME/monitoring/monitor.log 2>&1") | crontab -
    
    log_success "Monitoring setup completed"
}

create_validator() {
    log_info "Setting up validator..."
    
    # Check if validator key exists
    if [ ! -f "$NODE_HOME/config/priv_validator_key.json" ]; then
        log_error "Validator key not found. Node initialization may have failed."
        return 1
    fi
    
    # Get validator public key
    VALIDATOR_PUBKEY=$($BINARY_NAME tendermint show-validator --home "$NODE_HOME")
    
    log_info "Validator public key: $VALIDATOR_PUBKEY"
    
    # Create validator transaction template
    tee "$NODE_HOME/create-validator.json" > /dev/null <<EOF
{
  "pubkey": $VALIDATOR_PUBKEY,
  "amount": "1000000ant",
  "moniker": "$MONIKER",
  "identity": "",
  "website": "",
  "security_contact": "",
  "details": "Volnix Protocol Validator",
  "commission-rate": "0.10",
  "commission-max-rate": "0.20",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
EOF
    
    log_success "Validator setup template created at $NODE_HOME/create-validator.json"
    log_info "To create validator, run: $BINARY_NAME tx staking create-validator $NODE_HOME/create-validator.json --from <key-name> --chain-id $CHAIN_ID"
}

setup_firewall() {
    log_info "Configuring firewall..."
    
    # Check if ufw is available
    if command -v ufw &> /dev/null; then
        # Allow SSH
        sudo ufw allow 22/tcp
        
        # Allow P2P port
        sudo ufw allow 26656/tcp
        
        # Allow RPC port (optional, for API access)
        if [ "$ENABLE_RPC" = "true" ]; then
            sudo ufw allow 26657/tcp
        fi
        
        # Allow monitoring port
        if [ "$ENABLE_MONITORING" = "true" ]; then
            sudo ufw allow 8080/tcp
        fi
        
        # Enable firewall
        sudo ufw --force enable
        
        log_success "Firewall configured"
    else
        log_warning "UFW not available, skipping firewall configuration"
    fi
}

start_node() {
    log_info "Starting Volnix Protocol node..."
    
    # Start the service
    sudo systemctl start volnixd
    
    # Wait a moment for startup
    sleep 5
    
    # Check status
    if sudo systemctl is-active --quiet volnixd; then
        log_success "Node started successfully"
        
        # Show status
        log_info "Node status:"
        sudo systemctl status volnixd --no-pager -l
        
        # Show sync status
        sleep 10
        SYNC_STATUS=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.catching_up' 2>/dev/null || echo "unknown")
        if [ "$SYNC_STATUS" = "false" ]; then
            log_success "Node is synced"
        elif [ "$SYNC_STATUS" = "true" ]; then
            log_info "Node is syncing..."
        else
            log_warning "Unable to determine sync status"
        fi
    else
        log_error "Failed to start node"
        sudo journalctl -u volnixd --no-pager -l
        return 1
    fi
}

show_summary() {
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                    Deployment Completed!                    ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    echo "📋 Deployment Summary:"
    echo "  🏠 Node Home: $NODE_HOME"
    echo "  🔗 Chain ID: $CHAIN_ID"
    echo "  🏷️  Moniker: $MONIKER"
    echo "  📊 Version: $VOLNIX_VERSION"
    echo ""
    echo "🔧 Useful Commands:"
    echo "  📊 Check status: sudo systemctl status volnixd"
    echo "  📜 View logs: sudo journalctl -u volnixd -f"
    echo "  🔄 Restart node: sudo systemctl restart volnixd"
    echo "  ⏹️  Stop node: sudo systemctl stop volnixd"
    echo ""
    echo "🌐 Endpoints:"
    echo "  🔗 RPC: http://localhost:26657"
    echo "  📡 API: http://localhost:1317"
    echo "  📊 Monitoring: http://localhost:8080"
    echo ""
    echo "📁 Important Files:"
    echo "  ⚙️  Config: $NODE_HOME/config/config.toml"
    echo "  🌱 Genesis: $NODE_HOME/config/genesis.json"
    echo "  🔑 Validator Key: $NODE_HOME/config/priv_validator_key.json"
    echo "  📊 Monitoring: $NODE_HOME/monitoring/"
    echo ""
    echo "🚀 Next Steps:"
    echo "  1. Wait for node to sync (check with: curl -s localhost:26657/status | jq .result.sync_info.catching_up)"
    echo "  2. Create a wallet: $BINARY_NAME keys add <wallet-name>"
    echo "  3. Get tokens from faucet or exchange"
    echo "  4. Create validator: $BINARY_NAME tx staking create-validator ..."
    echo ""
    log_success "Volnix Protocol node deployment completed successfully!"
}

# Main deployment function
main() {
    print_banner
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --moniker)
                MONIKER="$2"
                shift 2
                ;;
            --chain-id)
                CHAIN_ID="$2"
                shift 2
                ;;
            --enable-state-sync)
                ENABLE_STATE_SYNC="true"
                shift
                ;;
            --enable-rpc)
                ENABLE_RPC="true"
                shift
                ;;
            --enable-monitoring)
                ENABLE_MONITORING="true"
                shift
                ;;
            --skip-build)
                SKIP_BUILD="true"
                shift
                ;;
            --help)
                echo "Volnix Protocol Deployment Script"
                echo ""
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --moniker <name>        Set node moniker"
                echo "  --chain-id <id>         Set chain ID (default: volnix-1)"
                echo "  --enable-state-sync     Enable state sync"
                echo "  --enable-rpc            Enable RPC access"
                echo "  --enable-monitoring     Enable monitoring"
                echo "  --skip-build            Skip binary build"
                echo "  --help                  Show this help"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Run deployment steps
    check_requirements
    
    if [ "$SKIP_BUILD" != "true" ]; then
        install_binary
    fi
    
    initialize_node
    download_genesis
    configure_node
    setup_systemd
    
    if [ "$ENABLE_MONITORING" = "true" ]; then
        setup_monitoring
    fi
    
    setup_firewall
    create_validator
    start_node
    show_summary
}

# Run main function with all arguments
main "$@"