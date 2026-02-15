# Automated Profile Generation

## Overview

The `start.ps1` script now **automatically generates** a fresh enrollment profile every time you start the server.

## What Happens Automatically

When you run `.\start.ps1`:

1. **Stops** any existing processes
2. **Builds** the MDM server
3. **Starts** the Cloudflare tunnel
4. **Generates** `enrollment.mobileconfig` with the tunnel URL ⭐ **NEW**
5. **Starts** the MDM server

## No Manual Steps Required

You no longer need to:
- ❌ Manually run `generate-enrollment-profile.ps1`
- ❌ Remember to update the URL in the profile
- ❌ Worry about stale tunnel URLs

## Fresh Profile Every Time

Each time the server starts:
- ✅ New tunnel URL is obtained
- ✅ New profile is generated automatically
- ✅ Profile is always up-to-date
- ✅ Ready to transfer to your Mac

## Usage

Simply start the server:

```powershell
.\start.ps1
```

You'll see:

```
[1/5] Stopping existing processes...
[2/5] Building MDM server...
[3/5] Starting Cloudflare tunnel...
  Tunnel URL: https://chose-academy-builds-cottage.trycloudflare.com
[4/5] Generating enrollment profile...
  Profile generated: enrollment.mobileconfig
[5/5] Starting MDM server...

========================================
  MDM Server is running!
  Tunnel:    https://...
  Admin:     https://.../admin/
  Health:    https://.../health
----------------------------------------
  Enrollment Profile: enrollment.mobileconfig
  Transfer to Mac and install to enroll
========================================
```

## Finding the Profile

The generated profile is in the `mdm-server` directory:

```
Device-Agent/
└── mdm-server/
    ├── start.ps1
    ├── generate-enrollment-profile.ps1
    └── enrollment.mobileconfig  ⭐ Fresh for this session!
```

## Transfer to Mac

**Option 1: Email**
- Attach `enrollment.mobileconfig` and email to yourself
- Open on Mac and double-click to install

**Option 2: AirDrop** (if Mac is nearby)
- Right-click `enrollment.mobileconfig`
- Share via AirDrop

**Option 3: Cloud/USB**
- Copy to OneDrive, Google Drive, or USB
- Open on Mac

## Customization

If you need to change the tenant ID, edit `start.ps1` around line 78:

```powershell
$tenantID = "your-custom-tenant-id"
```

## Troubleshooting

### Profile Not Generated

If you see:
```
Warning: generate-enrollment-profile.ps1 not found
```

Make sure `generate-enrollment-profile.ps1` exists in the `mdm-server` directory.

### Want to Regenerate Manually

You can still manually generate profiles if needed:

```powershell
.\generate-enrollment-profile.ps1 -PublicURL "your-url.ngrok.io"
```

## Benefits

🔄 **Always Fresh**: New tunnel = new profile automatically  
⚡ **Zero Effort**: No manual steps needed  
🚀 **Faster Workflow**: Start server → transfer profile → enroll  
✅ **No Mistakes**: Can't forget to update URLs
