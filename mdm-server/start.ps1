# MDM Server Startup Script
# Starts cloudflared tunnel + MDM server with the correct URL

Write-Host "=== MDM Server Startup ===" -ForegroundColor Cyan

# Refresh PATH to pick up newly installed tools
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

# Stop any existing processes
Write-Host "[1/5] Stopping existing processes..." -ForegroundColor Yellow
Stop-Process -Name mdmserver -Force -ErrorAction SilentlyContinue
Stop-Process -Name cloudflared -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Build the server
Write-Host "[2/5] Building MDM server..." -ForegroundColor Yellow
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir
go build -o mdmserver.exe ./cmd/mdmserver
if (-not $?) {
    Write-Host "ERROR: Build failed!" -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "  OK" -ForegroundColor Green

# Start cloudflared tunnel in background
Write-Host "[3/5] Starting Cloudflare tunnel..." -ForegroundColor Yellow
$tunnelJob = Start-Process -FilePath $cloudflared -ArgumentList "tunnel", "--url", "http://localhost:8080" -PassThru -NoNewWindow
Start-Sleep -Seconds 5

# Get the tunnel URL
try {
    $metricsPort = 20241
    # Try to find the metrics port from cloudflared
    $tunnelInfo = Invoke-RestMethod -Uri "http://127.0.0.1:$metricsPort/quicktunnel" -ErrorAction Stop
    $tunnelURL = "https://$($tunnelInfo.hostname)"
}
catch {
    # Fallback: try common metrics ports
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
        Write-Host "ERROR: Could not get tunnel URL. Check cloudflared output." -ForegroundColor Red
        Stop-Process -Id $tunnelJob.Id -Force -ErrorAction SilentlyContinue
        Pop-Location
        exit 1
    }
}

Write-Host "  Tunnel URL: $tunnelURL" -ForegroundColor Green

# Generate enrollment profile automatically
Write-Host "[4/5] Generating enrollment profile..." -ForegroundColor Yellow
try {
    # Extract hostname from tunnel URL (remove https://)
    $hostname = $tunnelURL -replace '^https?://', ''
    
    # Get default tenant ID from template or use hardcoded one
    $tenantID = "65871431-6d9a-4adc-83f7-53a37c35a82f"
    
    # Call the profile generator script
    $generateScript = Join-Path $scriptDir "generate-enrollment-profile.ps1"
    if (Test-Path $generateScript) {
        & $generateScript -PublicURL $hostname -TenantID $tenantID -OutputFile "enrollment.mobileconfig" | Out-Null
        Write-Host "  Profile generated: enrollment.mobileconfig" -ForegroundColor Green
    } else {
        Write-Host "  Warning: generate-enrollment-profile.ps1 not found, skipping profile generation" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "  Warning: Failed to generate enrollment profile: $_" -ForegroundColor Yellow
    Write-Host "  You can manually generate it with: .\generate-enrollment-profile.ps1 -PublicURL <url>" -ForegroundColor Gray
}

# Start MDM server with the tunnel URL
Write-Host "[5/5] Starting MDM server..." -ForegroundColor Yellow
$env:MDM_SERVER_URL = $tunnelURL

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  MDM Server is running!" -ForegroundColor Green
Write-Host "  Tunnel:    $tunnelURL" -ForegroundColor White
Write-Host "  Admin:     $tunnelURL/admin/" -ForegroundColor White
Write-Host "  Health:    $tunnelURL/health" -ForegroundColor White
Write-Host "----------------------------------------" -ForegroundColor Cyan
Write-Host "  Enrollment Profile: enrollment.mobileconfig" -ForegroundColor Yellow
Write-Host "  Transfer to Mac and install to enroll" -ForegroundColor Gray
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""

# Run server in foreground (Ctrl+C stops everything)
# All logs (stdout and stderr) will be shown in real-time
Write-Host "--- Server Logs (Ctrl+C to stop) ---" -ForegroundColor Cyan
Write-Host ""
try {
    # Start server and capture all output streams
    $process = Start-Process -FilePath ".\mdmserver.exe" -NoNewWindow -PassThru -Wait
    
    # If process exits with error code
    if ($process.ExitCode -ne 0 -and $process.ExitCode -ne $null) {
        Write-Host "`nERROR: Server exited with code $($process.ExitCode)" -ForegroundColor Red
    }
}
catch {
    Write-Host "`nERROR: Server crashed: $_" -ForegroundColor Red
}
finally {
    Write-Host "`nStopping cloudflared tunnel..." -ForegroundColor Yellow
    Stop-Process -Id $tunnelJob.Id -Force -ErrorAction SilentlyContinue
    Pop-Location
    Write-Host "Done." -ForegroundColor Green
}
