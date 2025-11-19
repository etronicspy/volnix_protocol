#!/usr/bin/env python3

"""
Генерация validator и node keys для мультинод сети
"""

import json
import os
import sys
import base64
import hashlib
from pathlib import Path

try:
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
    from cryptography.hazmat.primitives import serialization
except ImportError:
    print("❌ Установите cryptography: pip3 install cryptography")
    sys.exit(1)

def generate_node_key():
    """Генерирует node_key.json"""
    # Генерируем Ed25519 приватный ключ
    private_key = Ed25519PrivateKey.generate()
    
    # Получаем байты приватного и публичного ключей
    private_bytes = private_key.private_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PrivateFormat.Raw,
        encryption_algorithm=serialization.NoEncryption()
    )
    
    public_bytes = private_key.public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw
    )
    
    # Ed25519: 64 байта (32 private + 32 public)
    full_key = private_bytes + public_bytes
    
    return {
        "priv_key": {
            "type": "tendermint/PrivKeyEd25519",
            "value": base64.b64encode(full_key).decode('utf-8')
        }
    }

def generate_priv_validator_key(address_prefix=""):
    """Генерирует priv_validator_key.json"""
    private_key = Ed25519PrivateKey.generate()
    
    private_bytes = private_key.private_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PrivateFormat.Raw,
        encryption_algorithm=serialization.NoEncryption()
    )
    
    public_bytes = private_key.public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw
    )
    
    # Вычисляем address
    address_bytes = hashlib.sha256(public_bytes).digest()[:20]
    address = address_bytes.hex().upper()
    
    full_key = private_bytes + public_bytes
    
    return {
        "address": address,
        "pub_key": {
            "type": "tendermint/PubKeyEd25519",
            "value": base64.b64encode(public_bytes).decode('utf-8')
        },
        "priv_key": {
            "type": "tendermint/PrivKeyEd25519",
            "value": base64.b64encode(full_key).decode('utf-8')
        }
    }

def main():
    testnet_dir = sys.argv[1] if len(sys.argv) > 1 else "testnet-proper"
    num_nodes = int(sys.argv[2]) if len(sys.argv) > 2 else 3
    
    print(f"🔑 Генерация ключей для {num_nodes} узлов")
    print(f"📁 Директория: {testnet_dir}")
    print()
    
    for i in range(num_nodes):
        node_name = f"node{i}"
        node_dir = Path(testnet_dir) / node_name / ".volnix" / "config"
        node_dir.mkdir(parents=True, exist_ok=True)
        
        print(f"Генерация ключей для {node_name}...")
        
        # Генерируем node_key
        node_key = generate_node_key()
        node_key_file = node_dir / "node_key.json"
        with open(node_key_file, 'w') as f:
            json.dump(node_key, f, indent=2)
        
        # Генерируем priv_validator_key
        val_key = generate_priv_validator_key()
        val_key_file = node_dir / "priv_validator_key.json"
        with open(val_key_file, 'w') as f:
            json.dump(val_key, f, indent=2)
        
        # Вычисляем node ID для вывода
        priv_key_bytes = base64.b64decode(node_key['priv_key']['value'])
        pub_key_bytes = priv_key_bytes[32:]
        node_id = hashlib.sha256(pub_key_bytes).hexdigest()[:40]
        
        print(f"  ✅ Node ID: {node_id}")
        print(f"  ✅ Validator: {val_key['address']}")
        print()
    
    print("✅ Все ключи сгенерированы!")

if __name__ == "__main__":
    main()

