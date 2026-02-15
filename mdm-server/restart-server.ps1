# Restart MDM Server (without touching cloudflared tunnel)
# Use this during development to quickly rebuild and restart the server

Write-Host "=== Restarting MDM Server ===" -ForegroundColor Cyan

# Get script directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir

# Stop the MDM server (but NOT cloudflared)
Write-Host "[1/3] Stopping MDM server..." -ForegroundColor Yellow
Stop-Process -Name mdmserver -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1
Write-Host "  Stopped" -ForegroundColor Green

# Rebuild the server
Write-Host "[2/3] Building MDM server..." -ForegroundColor Yellow
go build -o mdmserver.exe ./cmd/mdmserver
if (-not $?) {
    Write-Host "ERROR: Build failed!" -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "  Build OK" -ForegroundColor Green

# Check if MDM_SERVER_URL is set (from the original start.ps1)
if (-not $env:MDM_SERVER_URL) {
    Write-Host "WARNING: MDM_SERVER_URL not set. Trying to detect tunnel URL..." -ForegroundColor Yellow
    
    # Try to get tunnel URL from cloudflared metrics
    $tunnelURL = $null
    foreach ($port in 20241, 43727, 45363) {
        try {
            $tunnelInfo = Invoke-RestMethod -Uri "http://127.0.0.1:${port}/quicktunnel" -ErrorAction Stop
            $tunnelURL = "https://$($tunnelInfo.hostname)"
            break
        }
        catch { continue }
    }
    
    if ($tunnelURL) {
        $env:MDM_SERVER_URL = $tunnelURL
        Write-Host "  Detected tunnel URL: $tunnelURL" -ForegroundColor Green
    } else {
        Write-Host "  Could not detect tunnel URL. Server will use default settings." -ForegroundColor Yellow
        Write-Host "  Make sure to start with .\start.ps1 first to create the tunnel!" -ForegroundColor Yellow
    }
}

# Start the server
Write-Host "[3/3] Starting MDM server..." -ForegroundColor Yellow
if ($env:MDM_SERVER_URL) {
    Write-Host "  Using URL: $env:MDM_SERVER_URL" -ForegroundColor White
}
Write-Host ""
Write-Host "--- Server Logs (Ctrl+C to stop) ---" -ForegroundColor Cyan
Write-Host ""

# Start server in foreground
try {
    $process = Start-Process -FilePath ".\mdmserver.exe" -NoNewWindow -PassThru -Wait
    
    if ($process.ExitCode -ne 0 -and $process.ExitCode -ne $null) {
        Write-Host "`nERROR: Server exited with code $($process.ExitCode)" -ForegroundColor Red
    }
}
catch {
    Write-Host "`nERROR: Server crashed: $_" -ForegroundColor Red
}
finally {
    Pop-Location
    Write-Host "Server stopped." -ForegroundColor Yellow
}
