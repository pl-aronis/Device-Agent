# MDM Quick Start Card

## 🚀 First Time Setup (5 minutes)

### 1. Get APNs Certificate
Go to [mdmcert.download](https://mdmcert.download) → Follow wizard → Download certificate

### 2. Upload to Dashboard
Open `http://your-server/admin/` → Select tenant → Upload certificate

### 3. Enroll a Mac
On the Mac: Open `http://your-server/enroll/TENANT_ID` → Download & install profile

---

## 📱 Managing Devices

| Click This | To Do This |
|------------|------------|
| 🔒 **Lock** | Lock device with PIN |
| 📍 **Locate** | Find device location |
| 🚨 **Lost Mode** | Show message on screen |
| 💀 **Wipe** | Erase everything |

---

## ⚠️ Remember

- APNs certificate expires **yearly** - renew before expiry!
- Commands may take up to 5 minutes if device is sleeping
- Wipe is **permanent** - cannot be undone

---

## 🆘 Need Help?

- Check `/health` endpoint for server status
- Look at server logs for errors
- Contact IT administrator
