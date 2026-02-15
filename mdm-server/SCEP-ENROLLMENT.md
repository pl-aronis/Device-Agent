# SCEP Enrollment Profile Generator

## Quick Start

1. **Start your tunnel** (ngrok or cloudflare):
   ```powershell
   # Example with ngrok
   ngrok http 8080
   
   # Example with cloudflare
   cloudflared tunnel --url http://localhost:8080
   ```

2. **Generate the enrollment profile** with your tunnel URL:
   ```powershell
   .\generate-enrollment-profile.ps1 -PublicURL "your-tunnel-url.ngrok.io"
   ```

3. **Transfer the profile** to your Mac and install it

## Usage

### Basic Usage
```powershell
.\generate-enrollment-profile.ps1 -PublicURL "chose-academy-builds-cottage.trycloudflare.com"
```

### Custom Tenant ID
```powershell
.\generate-enrollment-profile.ps1 -PublicURL "abc123.ngrok.io" -TenantID "my-tenant-id"
```

### Custom Output File
```powershell
.\generate-enrollment-profile.ps1 -PublicURL "abc123.ngrok.io" -OutputFile "my-custom-profile.mobileconfig"
```

## What This Does

The script:
- Reads the template file from `internal/scep/-enroll.mobileconfig`
- Replaces all `localhost:8080` URLs with your public tunnel URL
- Ensures all URLs use HTTPS (required by macOS)
- Generates a ready-to-install `.mobileconfig` file

## Installation on Mac

1. Transfer the generated `enrollment.mobileconfig` file to your Mac
2. Double-click the file to open it
3. Go to **System Settings** > **Privacy & Security** > **Profiles**
4. Click **Install** and follow the prompts
5. You may need to enter your Mac password

## Troubleshooting

### Check SCEP Server Accessibility

Before installing the profile, verify your SCEP server is accessible:

```bash
# Test GetCACaps
curl "https://your-url.ngrok.io/scep/TENANT-ID?operation=GetCACaps"
# Should return: POSTPKIOperation, SHA-256, AES, SCEPStandard

# Test GetCACert (returns binary data)
curl -I "https://your-url.ngrok.io/scep/TENANT-ID?operation=GetCACert"
# Should return: Content-Type: application/x-x509-ca-ra-cert
```

### Monitor Server Logs

The server now includes enhanced logging for SCEP operations. Watch for:
- `SCEP request: tenant=... operation=GetCACaps`
- `SCEP GetCACert: returning CA certificate`
- `SCEP PKIOperation: received ... bytes`

### Common Issues

1. **"Unable to obtain certificate from SCEP server"** (MDM-SCEP:15002)
   - Verify the tunnel URL is correct and accessible
   - Check that the server is running
   - Ensure HTTPS is used (not HTTP)

2. **Connection refused**
   - Your tunnel may have expired (ngrok free tier has 2-hour limit)
   - Regenerate the profile with the new tunnel URL

3. **Certificate trust issues**
   - This shouldn't happen with SCEP enrollment, but if it does, check that the CA was generated correctly

## Architecture

```
Mac Device
    ↓
  HTTPS (via tunnel)
    ↓
ngrok/cloudflare → localhost:8080
    ↓
MDM Server → SCEP Handler → CA Certificate/Signing
```

The enrollment flow:
1. Mac requests SCEP capabilities (`GetCACaps`)
2. Mac requests CA certificate (`GetCACert`)
3. Mac generates CSR and sends it (`PKIOperation`)
4. Server signs CSR and returns device certificate
5. Mac installs certificate and completes MDM enrollment
