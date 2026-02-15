# Restart MDM Server Only (does NOT touch cloudflared tunnel)
# Works with start.ps1 - signals it to auto-restart the server

Write-Host "=== Restarting MDM Server ===" -ForegroundColor Cyan

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir

# Rebuild the server first (before killing to minimize downtime)
Write-Host "[1/3] Building MDM server..." -ForegroundColor Yellow
go build -o mdmserver.exe ./cmd/mdmserver
if (-not $?) {
    Write-Host "ERROR: Build failed! Server NOT restarted." -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "  Build OK" -ForegroundColor Green

# Signal start.ps1 to restart (create signal file BEFORE killing)
Write-Host "[2/3] Signaling restart..." -ForegroundColor Yellow
New-Item -Path ".restart" -ItemType File -Force | Out-Null

# Kill the running server
Write-Host "[3/3] Stopping old server..." -ForegroundColor Yellow
Stop-Process -Name mdmserver -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Done! If start.ps1 is running, the server will restart automatically." -ForegroundColor Green
Write-Host "If not, run: .\start-server.ps1" -ForegroundColor Gray

Pop-Location
