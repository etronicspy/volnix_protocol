#!/bin/bash

# Volnix Protocol Testnet Startup Script
# Запускает 3 узла для тестирования сети

echo "🚀 Starting Volnix Protocol Testnet..."
echo "====================================="
echo ""

# Проверяем, что исполняемый файл существует
if [ ! -f "./volnixd-integrated" ]; then
    echo "❌ volnixd-integrated not found!"
    echo "Please run: go build -o volnixd-integrated ./cmd/volnixd"
    exit 1
fi

# Инициализируем testnet
echo "🔧 Initializing testnet..."
./volnixd-integrated network init-testnet 3

echo ""
echo "🌐 Starting network nodes..."

# Запускаем узлы в фоновых процессах
echo "🚀 Starting Node 0..."
./volnixd-integrated network start-node 0 > logs/node0.log 2>&1 &
NODE0_PID=$!

sleep 2

echo "🚀 Starting Node 1..."
./volnixd-integrated network start-node 1 > logs/node1.log 2>&1 &
NODE1_PID=$!

sleep 2

echo "🚀 Starting Node 2..."
./volnixd-integrated network start-node 2 > logs/node2.log 2>&1 &
NODE2_PID=$!

sleep 3

echo ""
echo "✅ All nodes started!"
echo "Node 0 PID: $NODE0_PID"
echo "Node 1 PID: $NODE1_PID" 
echo "Node 2 PID: $NODE2_PID"
echo ""

# Показываем статус сети
echo "📊 Network Status:"
./volnixd-integrated network status

echo ""
echo "🧪 Testing consensus..."
./volnixd-integrated network test-consensus

echo ""
echo "🔧 Testing modules..."
./volnixd-integrated network test-modules

echo ""
echo "🎉 Volnix Protocol Testnet is running!"
echo "======================================"
echo ""
echo "📋 Available commands:"
echo "  ./volnixd-integrated network status"
echo "  ./volnixd-integrated network test-consensus"
echo "  ./volnixd-integrated network test-modules"
echo ""
echo "🛑 To stop all nodes:"
echo "  kill $NODE0_PID $NODE1_PID $NODE2_PID"

# Создаем файл с PID для удобства остановки
echo "$NODE0_PID $NODE1_PID $NODE2_PID" > testnet_pids.txt
echo "📝 Node PIDs saved to testnet_pids.txt"