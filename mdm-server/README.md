# MDM Server - Deployment & Usage Guide

This guide details how to set up, configure, and use the MDM server for macOS device management.

## Prerequisites

1. **Go 1.21+**: Required to build the server.
2. **Cloudflared**: Required for public HTTPS tunnel (needed for MDM callbacks).
   ```powershell
   winget install cloudflare.cloudflared
   ```
   *Note: Close and reopen your terminal after installing to refresh environment variables.*

## 1. APNs Certificate Setup

Apple Push Notification service (APNs) is required for MDM. We use `mdmcert.download` (free service) to obtain certificates.

1. **Build the APNs Tool**:
   ```powershell
   go build -o apnstool.exe ./cmd/apnstool
   ```

2. **Generate & Submit CSR**:
   ```powershell
   .\apnstool.exe generate your-email@example.com
   ```
   This generates encryption keys and submits a request to mdmcert.download.

3. **Decrypt the Response**:
   - Check your email for a `.p7` file from mdmcert.download.
   - Save it as `mdmcert.p7` in the `apns_certs` directory.
   - Run:
     ```powershell
     .\apnstool.exe decrypt apns_certs/mdmcert.p7
     ```
   This creates `signed_csr.pem`.

4. **Upload to Apple**:
   - Go to [identity.apple.com/pushcert](https://identity.apple.com/pushcert).
   - Sign in with your Apple ID.
   - Create a Certificate > Upload `signed_csr.pem`.
   - Download the final certificate (`MDM_....pem`).

## 2. Starting the Server

We use a helper script that builds the server and starts a Cloudflare tunnel automatically.

```powershell
.\start.ps1
```

This will:
1. Build `mdmserver.exe`.
2. Start a `cloudflared` tunnel.
3. Automatically configure `MDM_SERVER_URL` with the tunnel URL.
4. Start the server.

**Look for the output:**
```
  Tunnel:    https://[random-name].trycloudflare.com
  Admin:     https://[random-name].trycloudflare.com/admin/
```

## 3. Configuring a Tenant

1. Open the Admin Dashboard URL in your browser.
2. Go to **Tenants** > **Create Tenant**.
3. In the Tenant details:
   - **Upload APNs Certificate**: Upload the `MDM_....pem` file you got from Apple.
   - **APNs Topic**: Copy the "UID" or "Subject" topic from the certificate (e.g., `com.apple.mgmt.External...`). You can find this by running `openssl x509 -in MDM_Cert.pem -noout -subject` or checking the certificate details on Mac.
   - **SCEP CA**: Click **"Generate CA"** to create a root CA for device identity issuance.

## 4. Device Enrollment

1. On the Mac you want to enroll, open Safari.
2. Go to the enrollment URL shown in the Tenant details (e.g., `/enroll/[tenant-id]`).
3. **Download Profile**: Click the download button.
4. **Install Profile**:
   - Go to **System Settings** > **Privacy & Security** > **Profiles**.
   - Double-click the downloaded "MDM Enrollment" profile.
   - Click **Install**.
   - Enter your Mac password when prompted.

The device should now be enrolled and visible in the Admin Dashboard under **Devices**.

## Troubleshooting

- **NSCocoaErrorDomain:4097**: Connection error. Ensure the server URL in the profile matches the current ngrok/Cloudflare tunnel URL. If using ngrok free tier, you must visit the URL in a browser first to bypass the interstitial page (or use Cloudflare Tunnel as recommended).
- **Profile Installation Failed**: Check that the SCEP CA is generated in the tenant settings.
- **APNs Topic Error**: Ensure the APNs topic matches exactly what is in the certificate.
