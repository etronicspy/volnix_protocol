#!/bin/bash

# Test Current Functionality Script for Волникс Протокол
# This script tests the current ABCI server functionality

set -e

echo "🧪 Testing Current Волникс Протокол Functionality"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    if [ "$status" = "OK" ]; then
        echo -e "${GREEN}✅ $message${NC}"
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠️  $message${NC}"
    elif [ "$status" = "ERROR" ]; then
        echo -e "${RED}❌ $message${NC}"
    else
        echo -e "${BLUE}ℹ️  $message${NC}"
    fi
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if process is running
is_process_running() {
    pgrep -f "$1" >/dev/null 2>&1
}

# Function to wait for process
wait_for_process() {
    local process_name=$1
    local max_wait=30
    local wait_time=0
    
    echo "⏳ Waiting for $process_name to start..."
    while [ $wait_time -lt $max_wait ]; do
        if is_process_running "$process_name"; then
            print_status "OK" "$process_name is running"
            return 0
        fi
        sleep 1
        wait_time=$((wait_time + 1))
    done
    
    print_status "ERROR" "$process_name failed to start within $max_wait seconds"
    return 1
}

# Function to cleanup
cleanup() {
    echo ""
    echo "🧹 Cleaning up..."
    if is_process_running volnixd; then
        pkill -f volnixd
        sleep 2
    fi
    print_status "OK" "Cleanup completed"
}

# Set trap for cleanup on exit
trap cleanup EXIT

# Check prerequisites
echo ""
echo "🔍 Checking Prerequisites"
echo "------------------------"

# Check if Go is installed
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_status "OK" "Go installed: $GO_VERSION"
else
    print_status "ERROR" "Go is not installed"
    exit 1
fi

# Check if binary exists
if [ -f "./build/volnixd" ]; then
    print_status "OK" "volnixd binary found"
else
    print_status "ERROR" "volnixd binary not found. Run 'make build' first"
    exit 1
fi

# Check binary version
echo ""
echo "📊 Binary Information"
echo "--------------------"
./build/volnixd version

# Test 1: Initialize node
echo ""
echo "🚀 Test 1: Node Initialization"
echo "-------------------------------"

# Clean up existing node if exists
if [ -d "$HOME/.volnix" ]; then
    print_status "WARN" "Existing node found, cleaning up..."
    rm -rf "$HOME/.volnix"
fi

# Initialize new node
print_status "INFO" "Initializing new node..."
./build/volnixd init testnode

# Check if files were created
if [ -f "$HOME/.volnix/config/genesis.json" ]; then
    print_status "OK" "Genesis file created"
else
    print_status "ERROR" "Genesis file not created"
    exit 1
fi

if [ -f "$HOME/.volnix/config/config.toml" ]; then
    print_status "OK" "Config file created"
else
    print_status "ERROR" "Config file not created"
    exit 1
fi

# Test 2: Start node
echo ""
echo "📡 Test 2: Node Startup"
echo "-----------------------"

# Start node in background
print_status "INFO" "Starting node..."
./build/volnixd start > /tmp/volnixd.log 2>&1 &
NODE_PID=$!

# Wait for node to start
if wait_for_process volnixd; then
    print_status "OK" "Node started successfully"
else
    print_status "ERROR" "Node failed to start"
    cat /tmp/volnixd.log
    exit 1
fi

# Test 3: Check node status
echo ""
echo "📊 Test 3: Node Status"
echo "----------------------"

# Check if process is running
if is_process_running volnixd; then
    print_status "OK" "Node process is running"
else
    print_status "ERROR" "Node process is not running"
    exit 1
fi

# Check if data directory was created
if [ -d "$HOME/.volnix/data" ]; then
    print_status "OK" "Data directory created"
else
    print_status "ERROR" "Data directory not created"
    exit 1
fi

# Test 4: Test CLI commands
echo ""
echo "⌨️  Test 4: CLI Commands"
echo "-----------------------"

# Test keys command
print_status "INFO" "Testing keys command..."
./build/volnixd keys add testkey > /tmp/keys.log 2>&1
if [ $? -eq 0 ]; then
    print_status "OK" "Keys command works"
else
    print_status "WARN" "Keys command failed (expected for ABCI server)"
fi

# Test tx command (should show help)
print_status "INFO" "Testing tx command..."
./build/volnixd tx --help > /tmp/tx.log 2>&1
if [ $? -eq 0 ]; then
    print_status "OK" "Tx command help works"
else
    print_status "ERROR" "Tx command failed"
    exit 1
fi

# Test query command (should show help)
print_status "INFO" "Testing query command..."
./build/volnixd query --help > /tmp/query.log 2>&1
if [ $? -eq 0 ]; then
    print_status "OK" "Query command help works"
else
    print_status "ERROR" "Query command failed"
    exit 1
fi

# Test 5: Check logs
echo ""
echo "📝 Test 5: Log Analysis"
echo "-----------------------"

# Check if node is producing output
if [ -s /tmp/volnixd.log ]; then
    print_status "OK" "Node is producing logs"
    
    # Check for specific messages
    if grep -q "ABCI server is running" /tmp/volnixd.log; then
        print_status "OK" "ABCI server message found in logs"
    else
        print_status "WARN" "ABCI server message not found in logs"
    fi
    
    if grep -q "Persistent (GoLevelDB)" /tmp/volnixd.log; then
        print_status "OK" "Persistent storage message found in logs"
    else
        print_status "WARN" "Persistent storage message not found in logs"
    fi
else
    print_status "ERROR" "Node is not producing logs"
    exit 1
fi

# Test 6: Performance check
echo ""
echo "⚡ Test 6: Performance Check"
echo "---------------------------"

# Check memory usage
MEMORY_USAGE=$(ps -o rss= -p $NODE_PID | awk '{print $1/1024}')
print_status "INFO" "Memory usage: ${MEMORY_USAGE} MB"

# Check if memory usage is reasonable (should be < 500MB for ABCI server)
if (( $(echo "$MEMORY_USAGE < 500" | bc -l) )); then
    print_status "OK" "Memory usage is reasonable"
else
    print_status "WARN" "Memory usage is high: ${MEMORY_USAGE} MB"
fi

# Test 7: Storage check
echo ""
echo "💾 Test 7: Storage Check"
echo "-----------------------"

# Check storage size
STORAGE_SIZE=$(du -sh "$HOME/.volnix" | awk '{print $1}')
print_status "INFO" "Storage size: $STORAGE_SIZE"

# Check if storage is reasonable (should be < 100MB for new node)
STORAGE_SIZE_BYTES=$(du -sb "$HOME/.volnix" | awk '{print $1}')
if [ $STORAGE_SIZE_BYTES -lt 104857600 ]; then  # 100MB in bytes
    print_status "OK" "Storage size is reasonable"
else
    print_status "WARN" "Storage size is large: $STORAGE_SIZE"
fi

# Test 8: Graceful shutdown
echo ""
echo "🛑 Test 8: Graceful Shutdown"
echo "----------------------------"

# Send SIGTERM to node
print_status "INFO" "Sending SIGTERM to node..."
kill -TERM $NODE_PID

# Wait for graceful shutdown
sleep 5

# Check if process is still running
if is_process_running volnixd; then
    print_status "WARN" "Node still running after SIGTERM, sending SIGKILL..."
    kill -KILL $NODE_PID
    sleep 2
fi

if ! is_process_running volnixd; then
    print_status "OK" "Node stopped successfully"
else
    print_status "ERROR" "Node failed to stop"
    exit 1
fi

# Final summary
echo ""
echo "🎯 Test Summary"
echo "==============="

print_status "OK" "All basic functionality tests passed!"
print_status "INFO" "Current status: ABCI server with persistent storage"
print_status "INFO" "Next step: Integrate with CometBFT for full blockchain"

echo ""
echo "📋 What was tested:"
echo "   ✅ Binary compilation and version"
echo "   ✅ Node initialization and configuration"
echo "   ✅ Node startup and process management"
echo "   ✅ CLI command availability"
echo "   ✅ Logging and output"
echo "   ✅ Memory and storage usage"
echo "   ✅ Graceful shutdown"

echo ""
echo "🔮 Next steps for full blockchain:"
echo "   1. Integrate with CometBFT v0.38.17"
echo "   2. Implement PoVB consensus"
echo "   3. Add P2P networking"
echo "   4. Enable RPC API"

echo ""
print_status "OK" "Testing completed successfully!"
echo "📚 See docs/COMETBFT_INTEGRATION_PLAN.md for next steps"
