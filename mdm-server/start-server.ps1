# Start MDM Server Only (tunnel must already be running)
# Use restart-server.ps1 to rebuild and restart without touching the tunnel

Write-Host "=== Starting MDM Server ===" -ForegroundColor Cyan

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir

# Stop any existing server
Write-Host "[1/3] Stopping existing server..." -ForegroundColor Yellow
Stop-Process -Name mdmserver -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

# Build the server
Write-Host "[2/3] Building MDM server..." -ForegroundColor Yellow
go build -o mdmserver.exe ./cmd/mdmserver
if (-not $?) {
    Write-Host "ERROR: Build failed!" -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "  Build OK" -ForegroundColor Green

# Detect tunnel URL if not set
if (-not $env:MDM_SERVER_URL) {
    foreach ($port in 20241, 43727, 45363) {
        try {
            $tunnelInfo = Invoke-RestMethod -Uri "http://127.0.0.1:${port}/quicktunnel" -ErrorAction Stop
            $env:MDM_SERVER_URL = "https://$($tunnelInfo.hostname)"
            break
        }
        catch { continue }
    }
}

if ($env:MDM_SERVER_URL) {
    Write-Host "  Server URL: $env:MDM_SERVER_URL" -ForegroundColor Green
} else {
    Write-Host "  WARNING: No tunnel detected. Using localhost." -ForegroundColor Yellow
    Write-Host "  Start tunnel first with: .\start-tunnel.ps1" -ForegroundColor Yellow
}

# Start server
Write-Host "[3/3] Starting MDM server..." -ForegroundColor Yellow
Write-Host ""
Write-Host "--- Server Logs (Ctrl+C to stop) ---" -ForegroundColor Cyan
Write-Host ""

try {
    # Run server directly so logs stream to this terminal and Ctrl+C works cleanly
    & .\mdmserver.exe
}
catch {
    Write-Host "`nServer stopped." -ForegroundColor Yellow
}
finally {
    Pop-Location
}
