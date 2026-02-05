#!/bin/bash

# BIOS/UEFI Password Setup Script
# This script attempts to set a BIOS password and provides manual instructions

set -e

echo "========================================="
echo "BIOS/UEFI Password Setup for Device Agent"
echo "========================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "ERROR: This script must be run as root"
    exit 1
fi

# Generate or retrieve BIOS password
BIOS_PASSWORD_FILE="/var/lib/device-agent-linux/bios.pwd"

if [ -f "$BIOS_PASSWORD_FILE" ]; then
    BIOS_PASSWORD=$(cat "$BIOS_PASSWORD_FILE")
    echo "Using existing BIOS password from: $BIOS_PASSWORD_FILE"
else
    echo "Generating new BIOS password..."
    mkdir -p /var/lib/device-agent-linux
    BIOS_PASSWORD=$(openssl rand -hex 16)
    echo "$BIOS_PASSWORD" > "$BIOS_PASSWORD_FILE"
    chmod 400 "$BIOS_PASSWORD_FILE"
    echo "BIOS password saved to: $BIOS_PASSWORD_FILE"
fi

echo ""
echo "BIOS Password: $BIOS_PASSWORD"
echo ""
echo "IMPORTANT: Save this password securely!"
echo ""

# Detect hardware vendor
VENDOR=$(dmidecode -s system-manufacturer 2>/dev/null | tr '[:upper:]' '[:lower:]' || echo "unknown")
echo "Detected system manufacturer: $VENDOR"
echo ""

# Attempt automatic BIOS password setting based on vendor
SUCCESS=false

if [[ "$VENDOR" == *"dell"* ]]; then
    echo "Dell system detected. Attempting to set BIOS password with CCTK..."
    
    if command -v cctk &> /dev/null; then
        if cctk --setuppwd="$BIOS_PASSWORD" 2>/dev/null; then
            echo "SUCCESS: BIOS password set using Dell CCTK"
            SUCCESS=true
        else
            echo "FAILED: Dell CCTK command failed"
        fi
    else
        echo "Dell CCTK not found. Install from: https://www.dell.com/support/kbdoc/en-us/000178000/dell-command-configure"
    fi

elif [[ "$VENDOR" == *"hp"* ]] || [[ "$VENDOR" == *"hewlett"* ]]; then
    echo "HP system detected. Attempting to set BIOS password..."
    
    if command -v hpsetup &> /dev/null; then
        if hpsetup -s -a"SetupPassword=$BIOS_PASSWORD" 2>/dev/null; then
            echo "SUCCESS: BIOS password set using HP Setup"
            SUCCESS=true
        else
            echo "FAILED: HP Setup command failed"
        fi
    else
        echo "HP Setup utility not found. Install HP BIOS Configuration Utility (BCU)"
    fi

elif [[ "$VENDOR" == *"lenovo"* ]]; then
    echo "Lenovo system detected. Attempting to set BIOS password..."
    
    if command -v thinkvantage &> /dev/null; then
        echo "ThinkVantage found, attempting to set password..."
        # Lenovo-specific commands would go here
        echo "FAILED: Automatic setting not fully implemented for Lenovo"
    else
        echo "Lenovo BIOS tools not found"
    fi

else
    echo "Unknown or unsupported vendor: $VENDOR"
fi

# If automatic setting failed, provide manual instructions
if [ "$SUCCESS" = false ]; then
    echo ""
    echo "========================================="
    echo "MANUAL BIOS PASSWORD SETUP REQUIRED"
    echo "========================================="
    echo ""
    echo "Automatic BIOS password setting is not supported on this hardware."
    echo "Please follow these steps to manually set the BIOS password:"
    echo ""
    echo "1. SAVE THIS PASSWORD: $BIOS_PASSWORD"
    echo ""
    echo "2. Reboot the system"
    echo ""
    echo "3. During boot, press the BIOS key (usually one of these):"
    echo "   - F2 (most common)"
    echo "   - DEL (Delete key)"
    echo "   - F10 (HP systems)"
    echo "   - F1 (some Lenovo systems)"
    echo "   - ESC (some systems)"
    echo ""
    echo "4. Navigate to the Security tab/section"
    echo ""
    echo "5. Set the following passwords:"
    echo "   - Supervisor Password: $BIOS_PASSWORD"
    echo "   - Administrator Password: $BIOS_PASSWORD"
    echo "   - Setup Password: $BIOS_PASSWORD"
    echo ""
    echo "6. Configure Boot Security:"
    echo "   - Disable USB Boot"
    echo "   - Disable Network/PXE Boot"
    echo "   - Set boot order to: Hard Drive only"
    echo "   - Enable Secure Boot (if available)"
    echo ""
    echo "7. Save settings and exit (usually F10)"
    echo ""
    echo "8. Verify the password is set by trying to enter BIOS again"
    echo ""
    echo "========================================="
    echo ""
    
    # Save instructions to file
    INSTRUCTIONS_FILE="/var/lib/device-agent-linux/bios-setup-instructions.txt"
    cat > "$INSTRUCTIONS_FILE" << EOF
BIOS Password Setup Instructions
Generated: $(date)

BIOS Password: $BIOS_PASSWORD

System Manufacturer: $VENDOR

Manual Setup Steps:
1. Reboot the system
2. Press BIOS key during boot (F2, DEL, F10, F1, or ESC)
3. Navigate to Security settings
4. Set Supervisor/Administrator Password to: $BIOS_PASSWORD
5. Disable USB Boot and Network/PXE Boot
6. Enable Secure Boot if available
7. Save and exit (F10)

Recovery:
If you forget the BIOS password, you may need to:
- Contact the manufacturer for a master password
- Remove the CMOS battery (hardware reset)
- Use manufacturer-specific reset procedures

Password File Location: $BIOS_PASSWORD_FILE
EOF
    
    chmod 400 "$INSTRUCTIONS_FILE"
    echo "Instructions saved to: $INSTRUCTIONS_FILE"
    echo ""
fi

# Additional security recommendations
echo "========================================="
echo "ADDITIONAL SECURITY RECOMMENDATIONS"
echo "========================================="
echo ""
echo "After setting the BIOS password, also configure:"
echo ""
echo "1. Disable Boot Devices:"
echo "   - USB devices"
echo "   - Network/PXE boot"
echo "   - CD/DVD drives"
echo "   - External drives"
echo ""
echo "2. Set Boot Order:"
echo "   - Only allow booting from internal hard drive"
echo ""
echo "3. Enable Security Features:"
echo "   - Secure Boot (if available)"
echo "   - TPM (Trusted Platform Module)"
echo "   - UEFI Boot Mode (disable Legacy/CSM)"
echo ""
echo "4. Lock BIOS Settings:"
echo "   - Some systems allow locking individual settings"
echo "   - Prevent changes without supervisor password"
echo ""
echo "========================================="
echo ""

# Test BIOS password retrieval
echo "Testing BIOS password retrieval..."
if [ -f "$BIOS_PASSWORD_FILE" ]; then
    RETRIEVED_PASSWORD=$(cat "$BIOS_PASSWORD_FILE")
    if [ "$RETRIEVED_PASSWORD" = "$BIOS_PASSWORD" ]; then
        echo "SUCCESS: BIOS password can be retrieved from $BIOS_PASSWORD_FILE"
    else
        echo "ERROR: Password mismatch!"
    fi
else
    echo "ERROR: Password file not found!"
fi

echo ""
echo "Setup complete!"
echo ""
