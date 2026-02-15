#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Generates an MDM enrollment profile with the specified public URL.

.DESCRIPTION
    This script creates an enrollment .mobileconfig file for macOS devices by replacing
    localhost URLs with the actual public URL (ngrok/cloudflare tunnel).

.PARAMETER PublicURL
    The public URL where your MDM server is accessible (without https:// prefix)
    Example: chose-academy-builds-cottage.trycloudflare.com

.PARAMETER TenantID
    The tenant ID for this enrollment profile
    Default: 65871431-6d9a-4adc-83f7-53a37c35a82f

.PARAMETER OutputFile
    Path where the generated profile should be saved
    Default: enrollment.mobileconfig

.EXAMPLE
    .\generate-enrollment-profile.ps1 -PublicURL "chose-academy-builds-cottage.trycloudflare.com"

.EXAMPLE
    .\generate-enrollment-profile.ps1 -PublicURL "abc123.ngrok.io" -TenantID "my-tenant-id" -OutputFile "my-profile.mobileconfig"
#>

param(
    [Parameter(Mandatory=$true, HelpMessage="The public URL where your MDM server is accessible")]
    [string]$PublicURL,
    
    [Parameter(Mandatory=$false)]
    [string]$TenantID = "65871431-6d9a-4adc-83f7-53a37c35a82f",
    
    [Parameter(Mandatory=$false)]
    [string]$OutputFile = "enrollment.mobileconfig"
)

# Template path
$templatePath = Join-Path $PSScriptRoot "internal\scep\-enroll.mobileconfig"

# Check if template exists
if (-not (Test-Path $templatePath)) {
    Write-Error "Template file not found at: $templatePath"
    exit 1
}

Write-Host "Generating enrollment profile..." -ForegroundColor Cyan
Write-Host "  Public URL: https://$PublicURL" -ForegroundColor Gray
Write-Host "  Tenant ID: $TenantID" -ForegroundColor Gray

# Read the template
$content = Get-Content -Path $templatePath -Raw

# Replace localhost:8080 with the public URL
$content = $content -replace 'http://localhost:8080', "https://$PublicURL"

# Update tenant ID if different from template
$content = $content -replace '65871431-6d9a-4adc-83f7-53a37c35a82f', $TenantID

# Ensure we're using https (not http) for all URLs
$content = $content -replace 'http://', 'https://'

# Write the output file
$outputPath = Join-Path $PSScriptRoot $OutputFile
$content | Set-Content -Path $outputPath -NoNewline

Write-Host ""
Write-Host "Enrollment profile generated successfully!" -ForegroundColor Green
Write-Host "  Output: $outputPath" -ForegroundColor Gray
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Transfer this file to your Mac device" -ForegroundColor White
Write-Host "  2. Double-click the .mobileconfig file to install" -ForegroundColor White
Write-Host "  3. Follow the installation prompts in System Settings" -  ForegroundColor White
Write-Host ""
Write-Host "Profile URLs configured:" -ForegroundColor Cyan
Write-Host "  SCEP:     https://$PublicURL/scep/$TenantID" -ForegroundColor Gray
Write-Host "  CheckIn:  https://$PublicURL/mdm/checkin" -ForegroundColor Gray
Write-Host "  MDM:      https://$PublicURL/mdm/connect" -ForegroundColor Gray
Write-Host ""
