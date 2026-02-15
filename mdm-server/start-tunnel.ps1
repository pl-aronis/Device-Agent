# Start Cloudflare Tunnel Only
# Run this first, then use start-server.ps1 separately

Write-Host "=== Starting Cloudflare Tunnel ===" -ForegroundColor Cyan

# Refresh PATH
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

# Find cloudflared
$cloudflared = (Get-Command cloudflared -ErrorAction SilentlyContinue).Source
if (-not $cloudflared) {
    $cloudflared = Get-ChildItem "C:\Program Files*\cloudflared*\cloudflared.exe" -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
}
if (-not $cloudflared) {
    Write-Host "ERROR: cloudflared not found. Install with: winget install cloudflare.cloudflared" -ForegroundColor Red
    exit 1
}
Write-Host "  Using: $cloudflared" -ForegroundColor Gray

# Stop any existing cloudflared
Write-Host "[1/2] Stopping existing tunnel..." -ForegroundColor Yellow
Stop-Process -Name cloudflared -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Start cloudflared tunnel
Write-Host "[2/2] Starting tunnel..." -ForegroundColor Yellow
$tunnelProcess = Start-Process -FilePath $cloudflared -ArgumentList "tunnel", "--url", "http://localhost:8080" -PassThru -NoNewWindow
Start-Sleep -Seconds 5

# Get tunnel URL
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
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Tunnel is running!" -ForegroundColor Green
Write-Host "  URL: $tunnelURL" -ForegroundColor White
Write-Host "  PID: $($tunnelProcess.Id)" -ForegroundColor Gray
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Press Ctrl+C to stop tunnel" -ForegroundColor Gray
Write-Host ""

# Wait for tunnel process (keeps script alive)
try {
    $tunnelProcess.WaitForExit()
}
catch {
    # Ctrl+C pressed
}
finally {
    Write-Host "`nStopping tunnel..." -ForegroundColor Yellow
    Stop-Process -Id $tunnelProcess.Id -Force -ErrorAction SilentlyContinue
    Write-Host "Tunnel stopped." -ForegroundColor Green
}
