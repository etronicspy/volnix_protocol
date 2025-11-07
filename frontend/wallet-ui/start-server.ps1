# Simple PowerShell HTTP Server for Wallet Demo
$port = 3000
$url = "http://localhost:$port/"

Write-Host "🚀 Starting Volnix Wallet Demo Server..." -ForegroundColor Green
Write-Host "📱 Open your browser and go to: $url" -ForegroundColor Cyan
Write-Host "📁 Serving files from: $(Get-Location)" -ForegroundColor Yellow
Write-Host "⏹️  Press Ctrl+C to stop the server" -ForegroundColor Gray
Write-Host ""

# Create HTTP listener
$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add($url)
$listener.Start()

Write-Host "✅ Server started successfully!" -ForegroundColor Green
Write-Host "🌐 Wallet interface available at: $url" -ForegroundColor Magenta

try {
    while ($listener.IsListening) {
        $context = $listener.GetContext()
        $request = $context.Request
        $response = $context.Response
        
        $path = $request.Url.AbsolutePath
        if ($path -eq "/") {
            $path = "/demo.html"
        }
        
        $filePath = Join-Path (Get-Location) $path.TrimStart('/')
        
        if (Test-Path $filePath) {
            $content = Get-Content $filePath -Raw -Encoding UTF8
            $buffer = [System.Text.Encoding]::UTF8.GetBytes($content)
            
            # Set content type
            if ($filePath.EndsWith(".html")) {
                $response.ContentType = "text/html; charset=utf-8"
            } elseif ($filePath.EndsWith(".css")) {
                $response.ContentType = "text/css"
            } elseif ($filePath.EndsWith(".js")) {
                $response.ContentType = "application/javascript"
            }
            
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
        
        Write-Host "$(Get-Date -Format 'HH:mm:ss') - $($request.HttpMethod) $($request.Url.AbsolutePath)" -ForegroundColor Gray
    }
} finally {
    $listener.Stop()
    Write-Host "🛑 Server stopped." -ForegroundColor Red
}