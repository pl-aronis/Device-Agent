# MDM Server - Getting Started Guide

This guide helps you set up device management for your organization's Mac computers.

---

## What You'll Achieve

After completing this guide, you'll be able to:
- ✅ Lock, locate, or wipe lost/stolen Macs remotely
- ✅ See all enrolled devices in a dashboard
- ✅ Manage multiple organizations (tenants) from one server

---

## Prerequisites

Before starting, you need:

| Item | Where to Get It | Cost |
|------|-----------------|------|
| Apple Business Manager account | [business.apple.com](https://business.apple.com) | Free |
| APNs Push Certificate | [mdmcert.download](https://mdmcert.download) | Free |
| A Mac computer to manage | Your organization | - |
| Server with public HTTPS URL | Your IT team | Varies |

---

## Step 1: Start the Server

1. **Download and build** the MDM server (your IT team should help with this)
2. **Run the server**:
   ```
   ./mdmserver
   ```
3. **Open the dashboard** in your browser:
   ```
   http://localhost:8080/admin/
   ```

You should see the MDM Dashboard with a default tenant created.

---

## Step 2: Get Your APNs Push Certificate

Apple requires a push certificate to send commands to devices.

1. Go to [mdmcert.download](https://mdmcert.download)
2. Follow their wizard to generate a CSR
3. Upload the CSR to Apple Push Portal
4. Download the signed certificate

> ⚠️ **Important**: This certificate expires yearly. Set a calendar reminder to renew it!

---

## Step 3: Configure Your Tenant

1. Open the **Admin Dashboard** at `/admin/`
2. Click on your tenant (e.g., "Default Organization")
3. Click **Upload APNs Certificate**
4. Upload your `.pem` certificate from Step 2
5. Enter your APNs Topic (from the certificate)

---

## Step 4: Enroll Your First Mac

### Option A: Manual Enrollment (Testing)

1. On the Mac you want to manage, open Safari
2. Go to: `https://your-server.com/enroll/YOUR_TENANT_ID`
3. Click **Download Profile**
4. Open System Settings → Privacy & Security → Profiles
5. Click **Install** on the MDM profile
6. Enter your Mac password when prompted

### Option B: Zero-Touch Enrollment (Production)

For new Macs purchased through Apple:

1. Log into [Apple Business Manager](https://business.apple.com)
2. Go to **Devices** → **MDM Servers**
3. Add your MDM server URL
4. Assign devices to your MDM server
5. When users turn on new Macs, they auto-enroll!

---

## Step 5: Manage Your Devices

From the Admin Dashboard, you can:

| Action | What It Does |
|--------|--------------|
| **Lock** | Locks the device with a PIN |
| **Locate** | Shows device location (if enabled) |
| **Lost Mode** | Displays a message on screen |
| **Wipe** | Erases all data (⚠️ use carefully!) |
| **Device Info** | Refreshes device details |

---

## Common Questions

### Q: How long until a device responds to commands?
**A:** Usually within seconds, but can take up to 5 minutes if the device is asleep.

### Q: Can I manage Windows PCs or phones?
**A:** This MDM server is specifically for macOS devices only.

### Q: What if my certificate expires?
**A:** Devices will stop responding to commands. Renew at mdmcert.download and re-upload.

### Q: Is the data secure?
**A:** Yes, all communication uses encryption. Set up HTTPS for production use.

---

## Getting Help

- Check the server logs for errors
- Visit the Admin Dashboard at `/admin/`
- Contact your IT administrator

---

## Quick Reference

| URL | Purpose |
|-----|---------|
| `/admin/` | Admin dashboard |
| `/enroll/{tenant}` | Device enrollment page |
| `/enroll/{tenant}/profile` | Download enrollment profile |
| `/health` | Server health check |
