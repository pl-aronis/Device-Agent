# Quick Test Guide - SCEP Enrollment

## Prerequisites
- [ ] MDM server is running (`.\start.ps1`)
- [ ] Tunnel is active (ngrok or cloudflare)
- [ ] You have the tunnel's public URL (e.g., `abc123.ngrok.io`)

## Step 1:  Generate Enrollment Profile

```powershell
# Replace with your actual tunnel URL (without https://)
.\generate-enrollment-profile.ps1 -PublicURL "chose-academy-builds-cottage.trycloudflare.com"
```

**Expected output:**
```
✓ Enrollment profile generated successfully!
  Output: C:\Users\bhata\...\enrollment.mobileconfig
```

## Step 2: Verify SCEP Endpoints

Replace `YOUR_URL` and `YOUR_TENANT_ID` below:

```bash
# Test GetCACaps
curl "https://YOUR_URL/scep/YOUR_TENANT_ID?operation=GetCACaps"

# Expected: POSTPKIOperation, SHA-256, AES, SCEPStandard
```

```bash
# Test GetCACert
curl -I "https://YOUR_URL/scep/YOUR_TENANT_ID?operation=GetCACert"

# Expected: Content-Type: application/x-x509-ca-ra-cert
```

**Default Tenant ID**: `65871431-6d9a-4adc-83f7-53a37c35a82f`

## Step 3: Install on Mac

1. Transfer `enrollment.mobileconfig` to your Mac
2. Double-click to open
3. System Settings > Privacy & Security > Profiles
4. Click Install
5. Enter Mac password when prompted

## Step 4: Monitor Server Logs

Look for these log messages in your MDM server console:

```
✓ SCEP request: tenant=... operation=GetCACaps ...
✓ SCEP GetCACaps: returning capabilities: [...]
✓ SCEP request: tenant=... operation=GetCACert ...
✓ SCEP GetCACert: returning CA certificate ...
✓ SCEP request: tenant=... operation=PKIOperation ...
✓ SCEP: issued certificate for ...
```

## Step 5: Verify Enrollment

**On Mac**:
- Open Keychain Access
- Look for "MDM Device Certificate"
- Signed by "testing MDM CA"

**On Server**:
- Device should appear in tenant's device list
- Check MDM dashboard for the new device

## Troubleshooting

### Error: "Unable to obtain certificate" (15002)
1. Check tunnel is running: `curl https://YOUR_URL`
2. Verify server is running on port 8080
3. Check server logs for SCEP requests
4. Regenerate profile if tunnel URL changed

### No logs appearing
- Device might not be reaching the server
- Check tunnel configuration
- Verify URL in profile matches tunnel URL

### Certificate not in Keychain
- Check for error messages in Console.app (filter "mdm" or "scep")
- Verify server logs show successful certificate issuance
- Try removing and reinstalling the profile

## Success Criteria

✅ SCEP endpoints respond correctly  
✅ Profile installs without errors  
✅ Server logs show all 3 SCEP operations  
✅ Certificate appears in Keychain  
✅ Device appears in tenant device list  

---

**Need more details?** See [SCEP-ENROLLMENT.md](file:///C:/Users/bhata/OneDrive/Desktop/development/projects/Device-Agent/mdm-server/SCEP-ENROLLMENT.md) and [walkthrough.md](file:///C:/Users/bhata/.gemini/antigravity/brain/72d2fad2-f851-4ba5-b417-cda9e4df8ad5/walkthrough.md)
