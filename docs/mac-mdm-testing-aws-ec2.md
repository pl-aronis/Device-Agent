# Testing macOS MDM on AWS EC2

## Overview

**Short Answer: Yes, you can test macOS MDM on AWS EC2**, but with important considerations and limitations.

AWS provides EC2 Mac instances that run on actual Apple hardware (Mac mini and Mac Studio), which makes them suitable for MDM testing. Unlike virtualized macOS environments, EC2 Mac instances are **bare metal instances** running on real Apple silicon or Intel Mac hardware.

---

## Can You Test MDM on EC2 Mac Instances?

### ✅ What Works

| Feature | Supported | Notes |
|---------|-----------|-------|
| MDM Profile Installation | ✅ Yes | Manual enrollment via `.mobileconfig` profiles |
| MDM Commands | ✅ Yes | Lock, wipe, query commands work |
| APNS Push Notifications | ✅ Yes | Requires valid APNS certificate |
| Configuration Profiles | ✅ Yes | Full support |
| Custom MDM Server Testing | ✅ Yes | Your own MDM server can manage the instance |
| DEP/ABM Enrollment | ⚠️ Limited | See limitations below |

### ⚠️ Limitations

| Limitation | Details |
|------------|---------|
| **No DEP/ABM Support** | EC2 Mac instances **cannot be enrolled in Apple Business Manager (ABM)** or Device Enrollment Program (DEP). The hardware is owned by AWS, not your organization. |
| **No Automated Enrollment** | You cannot get zero-touch MDM enrollment since ABM/DEP is not available |
| **Minimum 24-Hour Allocation** | You must allocate the Dedicated Host for at least 24 hours (even if you only use it for minutes) |
| **Cost** | Mac instances are expensive (~$25/day minimum for M1 instances) |
| **No Apple Intelligence** | EC2 Mac boots from external EBS volumes, so Apple Intelligence features are unavailable |

---

## AWS EC2 Mac Instance Types

| Instance Type | Hardware | CPU | Memory | GPU Cores | Best For |
|--------------|----------|-----|--------|-----------|----------|
| `mac1.metal` | 2018 Mac mini | Intel Core i7, 6 cores | 32 GiB | - | Legacy/Intel testing |
| `mac2.metal` | 2020 Mac mini | Apple M1, 8 cores | 16 GiB | 8 | General Apple Silicon testing |
| `mac2-m1ultra.metal` | 2022 Mac Studio | Apple M1 Ultra, 20 cores | 128 GiB | 64 | Performance-intensive testing |
| `mac2-m2.metal` | 2023 Mac mini | Apple M2, 8 cores | 24 GiB | 10 | Modern Apple Silicon testing |
| `mac2-m2pro.metal` | 2023 Mac mini | Apple M2 Pro, 12 cores | 32 GiB | 19 | Advanced testing |
| `mac-m4.metal` | 2024 Mac mini | Apple M4, 10 cores | 24 GiB | 10 | Latest hardware testing |
| `mac-m4pro.metal` | 2024 Mac mini | Apple M4 Pro, 14 cores | 48 GiB | 20 | Latest high-performance testing |

---

## Step-by-Step Guide: Setting Up MDM Testing on EC2 Mac

### Prerequisites

1. **AWS Account** with permissions to create EC2 instances and Dedicated Hosts
2. **Apple Developer Account** (for APNS certificates)
3. **Your MDM Server** running and accessible (you already have the mdm-server in this project)
4. **SSH Key Pair** for connecting to the Mac instance

---

### Step 1: Allocate a Dedicated Host

> [!IMPORTANT]
> Mac instances run ONLY on Dedicated Hosts with a **minimum 24-hour allocation period**.

#### Using AWS Console:

1. Go to **EC2 Dashboard** → **Dedicated Hosts** → **Allocate Dedicated Host**
2. Configure:
   - **Instance family**: `mac2` (or your preferred type)
   - **Instance type**: `mac2.metal`
   - **Availability Zone**: Choose one where Mac instances are available
   - **Quantity**: 1
3. Click **Allocate**

#### Using AWS CLI:

```bash
# Allocate a Dedicated Host for M2 Mac
aws ec2 allocate-hosts \
    --instance-type mac2.metal \
    --availability-zone us-east-1a \
    --quantity 1 \
    --auto-placement on

# Note the HostId from the response
```

---

### Step 2: Launch a Mac Instance

#### Using AWS Console:

1. Go to **EC2 Dashboard** → **Launch Instance**
2. Choose **macOS** AMI (e.g., macOS Sonoma 14.x)
3. Select instance type: `mac2.metal`
4. Configure:
   - **Network settings**: Ensure you have SSH (port 22) and VNC (port 5900) access
   - **Key pair**: Select or create one
   - **Tenancy**: Choose your Dedicated Host
5. Launch the instance

#### Using AWS CLI:

```bash
# Find macOS AMI
aws ec2 describe-images \
    --owners amazon \
    --filters "Name=name,Values=amzn-ec2-macos-14*" \
    --query 'Images[*].[ImageId,Name]' \
    --output table

# Launch instance
aws ec2 run-instances \
    --image-id ami-xxxxxxxxx \
    --instance-type mac2.metal \
    --key-name your-key-pair \
    --placement "HostId=h-xxxxxxxxx" \
    --block-device-mappings '[{"DeviceName":"/dev/sda1","Ebs":{"VolumeSize":200,"VolumeType":"gp3"}}]'
```

---

### Step 3: Connect to the Mac Instance

#### SSH Connection:

```bash
# Wait for instance to be ready (takes 5-10 minutes)
ssh -i your-key.pem ec2-user@<public-ip>
```

#### Enable VNC (for GUI access):

```bash
# SSH into the instance first, then run:
sudo /System/Library/CoreServices/RemoteManagement/ARDAgent.app/Contents/Resources/kickstart \
    -activate -configure -access -on \
    -configure -allowAccessFor -allUsers \
    -configure -restart -agent -privs -all

# Set a password for the ec2-user
sudo passwd ec2-user

# Enable Screen Sharing
sudo defaults write /var/db/launchd.db/com.apple.launchd/overrides.plist com.apple.screensharing -dict Disabled -bool false
sudo launchctl load -w /System/Library/LaunchDaemons/com.apple.screensharing.plist
```

Then connect using any VNC client to `<public-ip>:5900`

---

### Step 4: Prepare Your MDM Server

Ensure your MDM server is accessible from the EC2 Mac instance:

```bash
# From your MDM server machine, ensure it's publicly accessible
# or set up a VPN/tunnel to the EC2 instance

# Your MDM server should be running on a public IP or domain
# Example: https://your-mdm-server.example.com

# Verify your MDM server is accessible from the EC2 Mac:
ssh -i your-key.pem ec2-user@<mac-ip>
curl -v https://your-mdm-server.example.com/mdm/enroll
```

---

### Step 5: Manual MDM Enrollment

Since DEP/ABM is not available, you need to manually enroll the device:

#### Option A: Download and Install Enrollment Profile

1. **From the Mac instance**, open Safari
2. Navigate to your MDM server's enrollment URL
3. Download the `.mobileconfig` enrollment profile
4. Double-click to install
5. Go to **System Preferences** → **Profiles** → **Install**

#### Option B: Use `profiles` Command (CLI):

```bash
# Copy your enrollment profile to the Mac
scp -i your-key.pem enrollment.mobileconfig ec2-user@<mac-ip>:/tmp/

# SSH into the Mac and install
ssh -i your-key.pem ec2-user@<mac-ip>
sudo profiles install -path /tmp/enrollment.mobileconfig
```

#### Option C: Use Apple Configurator (if you have GUI access):

1. Download Apple Configurator 2 from the App Store
2. Create an enrollment profile
3. Install it on the device

---

### Step 6: Test MDM Commands

Once enrolled, test your MDM commands:

```bash
# From your MDM server, send test commands

# 1. Device Information Query
curl -X POST https://your-mdm-server/api/command \
    -H "Content-Type: application/json" \
    -d '{"udid": "<device-udid>", "command": "DeviceInformation"}'

# 2. Install Configuration Profile
curl -X POST https://your-mdm-server/api/command \
    -H "Content-Type: application/json" \
    -d '{"udid": "<device-udid>", "command": "InstallProfile", "payload": "<base64-profile>"}'

# 3. Device Lock (BE CAREFUL - this will lock the device!)
curl -X POST https://your-mdm-server/api/command \
    -H "Content-Type: application/json" \
    -d '{"udid": "<device-udid>", "command": "DeviceLock", "pin": "123456"}'
```

---

### Step 7: Clean Up (Important!)

> [!CAUTION]
> Remember you're billed for the Dedicated Host even if the instance is stopped. Release it when done testing.

```bash
# Terminate the instance
aws ec2 terminate-instances --instance-ids i-xxxxxxxxx

# Wait for termination (scrubbing takes ~1 hour for M1, ~3 hours for Intel)
# Then release the Dedicated Host
aws ec2 release-hosts --host-ids h-xxxxxxxxx
```

---

## Cost Estimation

| Resource | Pricing (US East) | 24-Hour Cost |
|----------|------------------|--------------|
| mac2.metal Dedicated Host | ~$1.083/hour | ~$26 |
| mac2-m2.metal Dedicated Host | ~$1.25/hour | ~$30 |
| mac-m4.metal Dedicated Host | ~$1.25/hour | ~$30 |
| EBS Storage (200GB gp3) | ~$0.08/GB/month | ~$0.50 |
| Data Transfer | Varies | Varies |

> [!TIP]
> For cost optimization, consider using **Savings Plans** if you plan regular testing.

---

## Alternative Testing Approaches

If DEP/ABM testing is critical, consider these alternatives:

### 1. Physical Mac Hardware

| Approach | Pros | Cons |
|----------|------|------|
| Buy a Mac mini | Full DEP/ABM support, one-time cost | Upfront investment, maintenance |
| Used/Refurbished Mac | Lower cost | May not support latest macOS |

### 2. MacStadium

- Cloud-hosted Mac infrastructure
- Some plans support DEP/ABM enrollment
- Visit: [macstadium.com](https://www.macstadium.com)

### 3. MacinCloud

- Another cloud Mac provider
- More flexible pricing than AWS
- Visit: [macincloud.com](https://www.macincloud.com)

### 4. Orka by MacStadium

- Kubernetes-based Mac virtualization
- Good for CI/CD and testing
- Visit: [orkadocs.macstadium.com](https://orkadocs.macstadium.com)

---

## Testing Checklist

Use this checklist to verify your MDM implementation:

### Enrollment Testing
- [ ] Manual enrollment via profile download
- [ ] Enrollment profile installation succeeds
- [ ] Device appears in MDM server dashboard
- [ ] APNS token is received and stored

### Command Testing
- [ ] `DeviceInformation` query works
- [ ] `SecurityInfo` query works
- [ ] `InstalledApplicationList` query works
- [ ] `CertificateList` query works
- [ ] `ProfileList` query works

### Profile Management
- [ ] Install configuration profile
- [ ] Remove configuration profile
- [ ] Profile restrictions apply correctly

### Security Commands
- [ ] `DeviceLock` command works
- [ ] `ClearPasscode` command works (if applicable)
- [ ] `EraseDevice` command works (test on disposable instance!)

### Push Notifications
- [ ] APNS push delivery works
- [ ] Device responds to push within expected time

---

## Troubleshooting

### Problem: Instance takes too long to start

**Solution**: Mac instances typically take 5-10 minutes to become available after launch. This is normal for bare metal instances.

### Problem: Cannot connect via SSH

**Solutions**:
1. Check security group allows port 22
2. Verify the instance has a public IP
3. Wait for the instance to fully boot
4. Check the instance status checks in EC2 console

### Problem: VNC connection fails

**Solutions**:
1. Ensure Screen Sharing is enabled
2. Check security group allows port 5900
3. Set a password for the ec2-user

### Problem: MDM enrollment fails

**Solutions**:
1. Verify MDM server is accessible from the Mac instance
2. Check APNS certificate is valid and not expired
3. Verify the enrollment profile is correctly signed
4. Check macOS version compatibility

### Problem: Push notifications not received

**Solutions**:
1. Verify APNS certificate in MDM server
2. Check device token is stored correctly
3. Verify network connectivity to Apple's APNS servers
4. Check for firewall rules blocking outbound traffic

---

## Security Considerations

> [!WARNING]
> When testing MDM on cloud infrastructure, keep these security practices in mind:

1. **Never use production APNS certificates** on test instances
2. **Use separate test Apple IDs** - don't sign in with personal accounts
3. **Encrypt sensitive data** - use EBS encryption for volumes
4. **Restrict network access** - use security groups to limit access
5. **Clean up properly** - ensure instances are terminated and hosts released
6. **Don't enable FileVault** - use EBS encryption instead

---

## Summary

| Question | Answer |
|----------|--------|
| Can I test MDM on AWS EC2 Mac? | **Yes** |
| Can I test DEP/ABM enrollment? | **No** - use physical Mac or specialized providers |
| Minimum cost for testing? | ~$26-30 for 24 hours |
| Best instance type for testing? | `mac2.metal` or `mac2-m2.metal` |
| Can I use my own MDM server? | **Yes** |

AWS EC2 Mac instances are excellent for:
- Testing MDM command handling
- Verifying profile installation/removal
- Testing APNS push notification flow
- Integration testing with your MDM server
- CI/CD for macOS MDM development

They are **not suitable** for:
- DEP/ABM zero-touch enrollment testing
- Apple Business Manager integration testing
- Any scenario requiring device ownership registration

---

## References

- [AWS EC2 Mac Instances Documentation](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-mac-instances.html)
- [Apple MDM Protocol Reference](https://developer.apple.com/documentation/devicemanagement)
- [Apple Business Manager User Guide](https://support.apple.com/guide/apple-business-manager)
- [APNS Documentation](https://developer.apple.com/documentation/usernotifications)
