# MDM Server Full Startup Script
# Starts BOTH cloudflared tunnel AND MDM server
# Server auto-restarts when killed by restart-server.ps1

Write-Host "=== MDM Server Full Startup ===" -ForegroundColor Cyan

# Refresh PATH
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir

# Find cloudflared
$cloudflared = (Get-Command cloudflared -ErrorAction SilentlyContinue).Source
if (-not $cloudflared) {
    $cloudflared = Get-ChildItem "C:\Program Files*\cloudflared*\cloudflared.exe" -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
}
if (-not $cloudflared) {
    Write-Host "ERROR: cloudflared not found. Install with: winget install cloudflare.cloudflared" -ForegroundColor Red
    Pop-Location
    exit 1
}

# Stop any existing processes
Write-Host "[1/4] Stopping existing processes..." -ForegroundColor Yellow
Stop-Process -Name mdmserver -Force -ErrorAction SilentlyContinue
Stop-Process -Name cloudflared -Force -ErrorAction SilentlyContinue
# Clean up stale restart signal
Remove-Item -Path ".restart" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Build the server
Write-Host "[2/4] Building MDM server..." -ForegroundColor Yellow
go build -o mdmserver.exe ./cmd/mdmserver
if (-not $?) {
    Write-Host "ERROR: Build failed!" -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "  OK" -ForegroundColor Green

# Start cloudflared tunnel in background
Write-Host "[3/4] Starting Cloudflare tunnel..." -ForegroundColor Yellow
$tunnelProcess = Start-Process -FilePath $cloudflared -ArgumentList "tunnel", "--url", "http://localhost:8080" -PassThru -NoNewWindow
Start-Sleep -Seconds 5

# Get the tunnel URL
$tunnelURL = $null
foreach ($port in 20241, 43727, 45363) {
    try {
        $tunnelInfo = Invoke-RestMethod -Uri "http://127.0.0.1:${port}/quicktunnel" -ErrorAction Stop
        $tunnelURL = "https://$($tunnelInfo.hostname)"
        break
    }
    catch { continue }
}

if (-not $tunnelURL) {
    Write-Host "ERROR: Could not get tunnel URL." -ForegroundColor Red
    Stop-Process -Id $tunnelProcess.Id -Force -ErrorAction SilentlyContinue
    Pop-Location
    exit 1
}
Write-Host "  Tunnel URL: $tunnelURL" -ForegroundColor Green

# Start MDM server
Write-Host "[4/4] Starting MDM server..." -ForegroundColor Yellow
$env:MDM_SERVER_URL = $tunnelURL

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  MDM Server is running!" -ForegroundColor Green
Write-Host "  Tunnel:    $tunnelURL" -ForegroundColor White
Write-Host "  Admin:     $tunnelURL/admin/" -ForegroundColor White
Write-Host "  Health:    $tunnelURL/health" -ForegroundColor White
Write-Host "----------------------------------------" -ForegroundColor Cyan
Write-Host "  Enroll:    $tunnelURL/enroll/{tenantID}" -ForegroundColor Yellow
Write-Host "  Profiles are generated dynamically" -ForegroundColor Gray
Write-Host "----------------------------------------" -ForegroundColor Cyan
Write-Host "  To restart server only: .\restart-server.ps1" -ForegroundColor Gray
Write-Host "  Press Ctrl+C to stop everything" -ForegroundColor Gray
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "--- Server Logs ---" -ForegroundColor Cyan
Write-Host ""

# Run server in a supervised loop
# - restart-server.ps1 creates a ".restart" file before killing mdmserver
# - If we see ".restart" after server exits, we restart automatically
# - Ctrl+C breaks the loop and kills the tunnel
try {
    while ($true) {
        & .\mdmserver.exe

        # Server exited - check if restart was requested
        if (Test-Path ".restart") {
            Remove-Item -Path ".restart" -Force -ErrorAction SilentlyContinue
            Write-Host ""
            Write-Host ">>> Server restarting (triggered by restart-server.ps1)..." -ForegroundColor Cyan
            Start-Sleep -Seconds 1
            continue
        }

        # No restart signal - server crashed or Ctrl+C
        Write-Host ""
        Write-Host "Server exited." -ForegroundColor Yellow
        break
    }
}
finally {
    Write-Host "`nStopping cloudflared tunnel..." -ForegroundColor Yellow
    Stop-Process -Id $tunnelProcess.Id -Force -ErrorAction SilentlyContinue
    Remove-Item -Path ".restart" -Force -ErrorAction SilentlyContinue
    Pop-Location
    Write-Host "Done." -ForegroundColor Green
}
