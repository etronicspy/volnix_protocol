# Volnix Wallet Web Server
# Веб-сервер для интерфейса кошелька с API

param(
    [int]$Port = 3000
)

Write-Host "🌐 Starting Volnix Wallet Web Server..." -ForegroundColor Green
Write-Host "Port: $Port" -ForegroundColor Cyan
Write-Host "URL: http://localhost:$Port" -ForegroundColor Cyan
Write-Host ""

# Создание HTTP listener
$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add("http://localhost:$Port/")
$listener.Start()

Write-Host "✅ Wallet server started successfully!" -ForegroundColor Green
Write-Host "🌐 Open your browser: http://localhost:$Port" -ForegroundColor Magenta
Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
Write-Host ""

# Функция для получения MIME типа
function Get-MimeType($extension) {
    switch ($extension.ToLower()) {
        ".html" { return "text/html; charset=utf-8" }
        ".css" { return "text/css" }
        ".js" { return "application/javascript" }
        ".json" { return "application/json" }
        ".png" { return "image/png" }
        ".jpg" { return "image/jpeg" }
        ".ico" { return "image/x-icon" }
        default { return "text/plain" }
    }
}

# Функция для обработки API запросов
function Handle-ApiRequest($request, $response) {
    $path = $request.Url.AbsolutePath
    $method = $request.HttpMethod
    
    try {
        switch -Regex ($path) {
            "^/api/wallets$" {
                if ($method -eq "GET") {
                    # Получение списка кошельков
                    $wallets = @()
                    $walletDir = "../.volnix/wallets"
                    
                    if (Test-Path $walletDir) {
                        $walletFiles = Get-ChildItem -Path $walletDir -Filter "*.json"
                        foreach ($file in $walletFiles) {
                            $wallet = Get-Content $file.FullName | ConvertFrom-Json
                            $wallets += $wallet
                        }
                    }
                    
                    $json = $wallets | ConvertTo-Json -Depth 3
                    $buffer = [System.Text.Encoding]::UTF8.GetBytes($json)
                    $response.ContentType = "application/json"
                    $response.ContentLength64 = $buffer.Length
                    $response.OutputStream.Write($buffer, 0, $buffer.Length)
                    return $true
                }
            }
            
            "^/api/wallet/([^/]+)$" {
                $walletName = $matches[1]
                
                if ($method -eq "GET") {
                    # Получение информации о кошельке
                    $walletFile = "../.volnix/wallets/$walletName.json"
                    
                    if (Test-Path $walletFile) {
                        $wallet = Get-Content $walletFile | ConvertFrom-Json
                        $json = $wallet | ConvertTo-Json -Depth 3
                        $buffer = [System.Text.Encoding]::UTF8.GetBytes($json)
                        $response.ContentType = "application/json"
                        $response.ContentLength64 = $buffer.Length
                        $response.OutputStream.Write($buffer, 0, $buffer.Length)
                        return $true
                    } else {
                        $response.StatusCode = 404
                        return $true
                    }
                }
            }
            
            "^/api/transactions$" {
                if ($method -eq "GET") {
                    # Получение истории транзакций
                    $transactions = @()
                    $txDir = "../.volnix/transactions"
                    
                    if (Test-Path $txDir) {
                        $txFiles = Get-ChildItem -Path $txDir -Filter "*.json" | Sort-Object LastWriteTime -Descending
                        foreach ($file in $txFiles) {
                            $tx = Get-Content $file.FullName | ConvertFrom-Json
                            $transactions += $tx
                        }
                    }
                    
                    $json = $transactions | ConvertTo-Json -Depth 3
                    $buffer = [System.Text.Encoding]::UTF8.GetBytes($json)
                    $response.ContentType = "application/json"
                    $response.ContentLength64 = $buffer.Length
                    $response.OutputStream.Write($buffer, 0, $buffer.Length)
                    return $true
                }
                
                if ($method -eq "POST") {
                    # Отправка транзакции
                    $reader = New-Object System.IO.StreamReader($request.InputStream)
                    $body = $reader.ReadToEnd()
                    $txData = $body | ConvertFrom-Json
                    
                    # Вызов transaction manager для отправки
                    $result = powershell -ExecutionPolicy Bypass -Command "
                        Set-Location '../'
                        .\scripts\transaction-manager.ps1 -Action send -From '$($txData.from)' -To '$($txData.to)' -Amount '$($txData.amount)'
                    "
                    
                    $response.ContentType = "application/json"
                    $json = @{ success = $true; message = "Transaction sent" } | ConvertTo-Json
                    $buffer = [System.Text.Encoding]::UTF8.GetBytes($json)
                    $response.ContentLength64 = $buffer.Length
                    $response.OutputStream.Write($buffer, 0, $buffer.Length)
                    return $true
                }
            }
            
            "^/api/wallet/create$" {
                if ($method -eq "POST") {
                    # Создание нового кошелька
                    $reader = New-Object System.IO.StreamReader($request.InputStream)
                    $body = $reader.ReadToEnd()
                    $walletData = $body | ConvertFrom-Json
                    
                    # Вызов transaction manager для создания кошелька
                    $result = powershell -ExecutionPolicy Bypass -Command "
                        Set-Location '../'
                        .\scripts\transaction-manager.ps1 -Action create -KeyName '$($walletData.name)'
                    "
                    
                    $response.ContentType = "application/json"
                    $json = @{ success = $true; message = "Wallet created" } | ConvertTo-Json
                    $buffer = [System.Text.Encoding]::UTF8.GetBytes($json)
                    $response.ContentLength64 = $buffer.Length
                    $response.OutputStream.Write($buffer, 0, $buffer.Length)
                    return $true
                }
            }
        }
    } catch {
        Write-Host "API Error: $($_.Exception.Message)" -ForegroundColor Red
        $response.StatusCode = 500
        return $true
    }
    
    return $false
}

try {
    while ($listener.IsListening) {
        $context = $listener.GetContext()
        $request = $context.Request
        $response = $context.Response
        
        $path = $request.Url.AbsolutePath
        $method = $request.HttpMethod
        
        Write-Host "$(Get-Date -Format 'HH:mm:ss') - $method $path" -ForegroundColor Gray
        
        # Обработка API запросов
        if ($path.StartsWith("/api/")) {
            if (Handle-ApiRequest $request $response) {
                $response.OutputStream.Close()
                continue
            }
        }
        
        # Обработка статических файлов
        if ($path -eq "/") {
            $path = "/index.html"
        }
        
        $filePath = Join-Path (Get-Location) $path.TrimStart('/')
        
        if (Test-Path $filePath) {
            $content = Get-Content $filePath -Raw -Encoding UTF8
            $buffer = [System.Text.Encoding]::UTF8.GetBytes($content)
            
            $extension = [System.IO.Path]::GetExtension($filePath)
            $response.ContentType = Get-MimeType $extension
            $response.ContentLength64 = $buffer.Length
            $response.OutputStream.Write($buffer, 0, $buffer.Length)
        } else {
            $response.StatusCode = 404
            $notFound = "404 - File Not Found"
            $buffer = [System.Text.Encoding]::UTF8.GetBytes($notFound)
            $response.ContentLength64 = $buffer.Length
            $response.OutputStream.Write($buffer, 0, $buffer.Length)
        }
        
        $response.OutputStream.Close()
    }
} catch {
    Write-Host "Server Error: $($_.Exception.Message)" -ForegroundColor Red
} finally {
    $listener.Stop()
    Write-Host "Wallet server stopped." -ForegroundColor Red
}