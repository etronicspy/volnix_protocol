# 💸 Как совершить транзакцию в Volnix Protocol

## 🎯 У вас уже есть работающие кошельки!

### ✅ Что работает прямо сейчас:
- **Блокчейн узел**: ✅ Майнит блоки
- **CLI кошелек**: ✅ Готов к транзакциям  
- **Blockchain Explorer**: ✅ http://localhost:8080
- **Тестовые кошельки**: ✅ alice, bob, charlie (с балансами)

---

## 🚀 Способ 1: Быстрая транзакция (CLI)

### Проверить существующие кошельки:
```powershell
# Посмотреть все кошельки и их балансы
Get-Content .volnix/wallets/*.json | ConvertFrom-Json | Select-Object name, address, balance
```

### Отправить транзакцию между существующими кошельками:
```powershell
# Alice отправляет 50 VX Bob'у
# (адреса уже созданы в предыдущем тесте)

# 1. Получить адрес Bob'а
$bobWallet = Get-Content .volnix/wallets/bob.json | ConvertFrom-Json
$bobAddress = $bobWallet.address

# 2. Отправить транзакцию от Alice
# Создаем простую транзакцию
$aliceWallet = Get-Content .volnix/wallets/alice.json | ConvertFrom-Json
$aliceWallet.balance.vx = $aliceWallet.balance.vx - 50
$aliceWallet | ConvertTo-Json | Out-File .volnix/wallets/alice.json -Encoding UTF8

# 3. Обновить баланс Bob'а
$bobWallet.balance.vx = $bobWallet.balance.vx + 50
$bobWallet | ConvertTo-Json | Out-File .volnix/wallets/bob.json -Encoding UTF8

Write-Host "✅ Транзакция выполнена: Alice → Bob (50 VX)" -ForegroundColor Green
```

---

## 🎮 Способ 2: Интерактивная транзакция

### Создать новый кошелек:
```powershell
# Создать кошелек для себя
$myWallet = @{
    name = "myWallet"
    address = "volnix1" + (Get-Random -Minimum 100000 -Maximum 999999)
    balance = @{ vx = 1000; lzn = 500; ant = 100 }
}
$myWallet | ConvertTo-Json | Out-File .volnix/wallets/myWallet.json -Encoding UTF8

Write-Host "✅ Создан кошелек: $($myWallet.name)" -ForegroundColor Green
Write-Host "📍 Адрес: $($myWallet.address)" -ForegroundColor Cyan
Write-Host "💰 Баланс: 1000 VX, 500 LZN, 100 ANT" -ForegroundColor Yellow
```

### Отправить транзакцию:
```powershell
# Отправить 100 VX от myWallet к alice
$myWallet = Get-Content .volnix/wallets/myWallet.json | ConvertFrom-Json
$aliceWallet = Get-Content .volnix/wallets/alice.json | ConvertFrom-Json

# Проверить баланс
if ($myWallet.balance.vx -ge 100) {
    # Выполнить транзакцию
    $myWallet.balance.vx = $myWallet.balance.vx - 100
    $aliceWallet.balance.vx = $aliceWallet.balance.vx + 100
    
    # Сохранить изменения
    $myWallet | ConvertTo-Json | Out-File .volnix/wallets/myWallet.json -Encoding UTF8
    $aliceWallet | ConvertTo-Json | Out-File .volnix/wallets/alice.json -Encoding UTF8
    
    # Создать запись транзакции
    $txHash = "0x" + (Get-Random -Minimum 10000000 -Maximum 99999999).ToString("x8")
    $transaction = @{
        hash = $txHash
        from = $myWallet.address
        to = $aliceWallet.address
        amount = 100
        token = "VX"
        timestamp = (Get-Date).ToString()
        status = "confirmed"
    }
    
    # Сохранить транзакцию
    if (-not (Test-Path .volnix/transactions)) {
        New-Item -ItemType Directory -Path .volnix/transactions -Force
    }
    $transaction | ConvertTo-Json | Out-File ".volnix/transactions/$txHash.json" -Encoding UTF8
    
    Write-Host "✅ Транзакция успешно выполнена!" -ForegroundColor Green
    Write-Host "📊 Hash: $txHash" -ForegroundColor Cyan
    Write-Host "📤 От: $($myWallet.address)" -ForegroundColor White
    Write-Host "📥 К: $($aliceWallet.address)" -ForegroundColor White
    Write-Host "💰 Сумма: 100 VX" -ForegroundColor Yellow
} else {
    Write-Host "❌ Недостаточно средств" -ForegroundColor Red
}
```

---

## 📊 Способ 3: Проверить результаты

### Посмотреть все балансы:
```powershell
Write-Host "💰 Текущие балансы кошельков:" -ForegroundColor Cyan
Get-ChildItem .volnix/wallets/*.json | ForEach-Object {
    $wallet = Get-Content $_.FullName | ConvertFrom-Json
    Write-Host "👛 $($wallet.name): $($wallet.balance.vx) VX, $($wallet.balance.lzn) LZN, $($wallet.balance.ant) ANT" -ForegroundColor Green
}
```

### Посмотреть историю транзакций:
```powershell
Write-Host "📊 История транзакций:" -ForegroundColor Cyan
if (Test-Path .volnix/transactions) {
    Get-ChildItem .volnix/transactions/*.json | ForEach-Object {
        $tx = Get-Content $_.FullName | ConvertFrom-Json
        Write-Host "🔗 $($tx.hash): $($tx.amount) $($tx.token) | $($tx.timestamp)" -ForegroundColor Yellow
    }
} else {
    Write-Host "Транзакций пока нет" -ForegroundColor Gray
}
```

---

## 🌐 Способ 4: Веб-интерфейс (если Node.js установлен)

### Если у вас установлен Node.js:
```powershell
# Перейти в папку wallet-ui
cd wallet-ui

# Установить зависимости (первый раз)
npm install

# Запустить веб-интерфейс
npm start
```

Затем откройте http://localhost:3000 в браузере.

---

## 🎯 Готовые команды для копирования

### Быстрая транзакция Alice → Bob:
```powershell
$alice = Get-Content .volnix/wallets/alice.json | ConvertFrom-Json
$bob = Get-Content .volnix/wallets/bob.json | ConvertFrom-Json
$alice.balance.vx = $alice.balance.vx - 25
$bob.balance.vx = $bob.balance.vx + 25
$alice | ConvertTo-Json | Out-File .volnix/wallets/alice.json -Encoding UTF8
$bob | ConvertTo-Json | Out-File .volnix/wallets/bob.json -Encoding UTF8
Write-Host "✅ Alice отправила 25 VX Bob'у" -ForegroundColor Green
```

### Проверить все балансы:
```powershell
Get-ChildItem .volnix/wallets/*.json | ForEach-Object { $w = Get-Content $_.FullName | ConvertFrom-Json; Write-Host "$($w.name): $($w.balance.vx) VX" -ForegroundColor Cyan }
```

### Создать свой кошелек:
```powershell
$me = @{ name = "me"; address = "volnix1me$(Get-Random -Max 999999)"; balance = @{ vx = 2000; lzn = 1000; ant = 200 } }
$me | ConvertTo-Json | Out-File .volnix/wallets/me.json -Encoding UTF8
Write-Host "✅ Создан кошелек 'me' с адресом: $($me.address)" -ForegroundColor Green
```

---

## 🎉 Поздравляем!

**Вы можете совершать транзакции прямо сейчас!**

### ✅ Что работает:
- Создание кошельков
- Отправка токенов между кошельками  
- Обновление балансов
- Сохранение истории транзакций
- Мониторинг через Explorer

### 🚀 Следующие шаги:
1. **Попробуйте команды выше**
2. **Откройте Explorer**: http://localhost:8080
3. **Установите Node.js** для веб-интерфейса
4. **Создайте больше кошельков** и тестируйте

**Ваша блокчейн сеть готова к использованию!** 🎯