# 🎁 Trading App Deployment Package - README

Welcome! This package contains everything you need to deploy your trading application to your Ubuntu VPS.

---

## 📦 What's in This Package?

This directory contains:

1. **trading-app-deployment.tar.gz** - The complete application archive
2. **deployment_guide.md** - Step-by-step deployment guide (MAIN GUIDE)
3. **DEPLOYMENT_SUMMARY.md** - Overview of what you're deploying
4. **vps-quick-setup.sh** - Automated setup script for the VPS
5. **README_DEPLOYMENT_PACKAGE.md** - This file

---

## 🚀 Quick Start - 3 Steps

### Step 1: Transfer Files to VPS

Transfer the deployment package and quick setup script to your VPS:

```bash
cd /home/ubuntu/code_artifacts/

# Transfer the main application package
scp trading-app-deployment.tar.gz root@67.211.219.94:/root/

# Transfer the quick setup script
scp vps-quick-setup.sh root@67.211.219.94:/root/
```

### Step 2: Run the Quick Setup Script

SSH into your VPS and run the automated setup:

```bash
ssh root@67.211.219.94
cd /root/
bash vps-quick-setup.sh
```

This script will:
- Update your system
- Install Node.js and dependencies
- Extract the application
- Create configuration files
- Set up firewall rules

### Step 3: Follow the Detailed Deployment Guide

Open the **deployment_guide.md** and follow from Step 5 onwards (configuration and building).

---

## 📖 Which File Should You Read?

### For Complete Beginners - Start Here:
👉 **deployment_guide.md**

This is your main guide with:
- Every command you need to copy-paste
- Explanations for what each command does
- Verification steps to check if things worked
- Troubleshooting section
- Security tips

**Time needed**: 15-20 minutes if you follow step-by-step

---

### For Quick Overview:
👉 **DEPLOYMENT_SUMMARY.md**

This gives you:
- What's in the application
- System requirements
- Configuration needed
- Quick checklist

**Time to read**: 3-5 minutes

---

### For Automated Setup:
👉 **vps-quick-setup.sh**

Run this script on your VPS to automate:
- System updates
- Dependency installation
- File extraction
- Configuration file creation

**Time to run**: 3-5 minutes

---

## 🎯 Recommended Deployment Path

**Option A: Beginner-Friendly (Recommended)**
1. Read DEPLOYMENT_SUMMARY.md (to understand what you're deploying)
2. Transfer files to VPS (using commands from deployment_guide.md)
3. Run vps-quick-setup.sh on VPS (automates setup)
4. Follow deployment_guide.md from Step 5 onwards (build & start apps)

**Option B: Full Manual Control**
1. Read DEPLOYMENT_SUMMARY.md
2. Follow deployment_guide.md step-by-step from beginning to end
3. Manually run every command

**Option C: Experienced Users**
1. Transfer trading-app-deployment.tar.gz to VPS
2. Extract and review the included DEPLOYMENT.md and deploy.sh
3. Run deploy.sh with your API keys

---

## 📋 Pre-Deployment Checklist

Before you start, make sure you have:

- [ ] SSH access to your VPS (67.211.219.94)
- [ ] Root password for the VPS
- [ ] OpenAlgo API key (for trading functionality)
- [ ] Abacus.AI API key (for AI chat features)
- [ ] Basic terminal/command-line knowledge
- [ ] 20 minutes of uninterrupted time

---

## 🔑 API Keys You'll Need

### OpenAlgo API Key
**What it's for**: Connecting to OpenAlgo trading platform to execute trades

**Where to get it**: 
- Log into your OpenAlgo account
- Go to API settings
- Generate/copy your API key

### Abacus.AI API Key
**What it's for**: Powering the AI chat assistant in the application

**Where to get it**:
- Log into Abacus.AI
- Go to API settings/developer console
- Generate/copy your API key

⚠️ **Keep these keys secure!** Never share them or commit them to public repositories.

---

## 🖥️ VPS Information

**IP Address**: 67.211.219.94  
**Operating System**: Ubuntu  
**RAM**: 2GB  
**CPU**: 1 core  
**SSH User**: root  

**Ports that will be used**:
- Port 3000 - Frontend (Next.js web interface)
- Port 8080 - Backend (Go API server)
- Port 22 - SSH (for remote access)

---

## 📊 What Gets Deployed

### Backend (Go Server)
- **What**: API server handling all business logic
- **Port**: 8080
- **Features**: 
  - Trading operations
  - User authentication
  - File processing
  - WebSocket for real-time updates
  - Database management
  - AI integration

### Frontend (Next.js Web App)
- **What**: User interface for the application
- **Port**: 3000
- **Features**:
  - Responsive web interface
  - Real-time charts and dashboards
  - Strategy builder
  - Portfolio tracking
  - AI chat interface
  - File uploads/downloads

---

## 🛠️ Files Breakdown

### 1. trading-app-deployment.tar.gz (349 KB)

**What it is**: Compressed archive containing the complete application

**Contents**:
- Go backend source code
- Next.js frontend source code
- Configuration scripts
- Documentation
- Deployment automation scripts

**What it does NOT contain**:
- node_modules (will be installed during setup)
- Compiled binaries (will be built during setup)
- Your API keys (you'll add these during configuration)

---

### 2. deployment_guide.md (Your Main Guide)

**What it is**: Complete step-by-step deployment instructions

**Sections**:
1. Transfer files to VPS
2. Connect to VPS
3. Prepare VPS environment
4. Extract application
5. Configure backend and frontend
6. Build applications
7. Set up services
8. Start applications
9. Verify everything works
10. Configure firewall
11. Monitoring and maintenance
12. Troubleshooting

**Why it's detailed**: Written for beginners with explanations for every command

**Length**: ~450 lines (comprehensive!)

---

### 3. DEPLOYMENT_SUMMARY.md

**What it is**: High-level overview document

**Contains**:
- Application structure
- System requirements
- Configuration details
- Port information
- Quick verification checklist
- Common issues and solutions

**Best for**: Getting oriented before starting deployment

---

### 4. vps-quick-setup.sh

**What it is**: Bash script that automates VPS preparation

**What it does**:
1. Updates system packages
2. Installs Node.js 18.x
3. Installs system dependencies (tesseract, git, etc.)
4. Extracts the deployment package
5. Creates configuration file templates
6. Sets up firewall rules (but doesn't enable yet)

**How to use**:
```bash
# Transfer to VPS
scp vps-quick-setup.sh root@67.211.219.94:/root/

# SSH into VPS
ssh root@67.211.219.94

# Run the script
bash vps-quick-setup.sh
```

**Time to run**: 3-5 minutes

---

## 🎨 Visual Deployment Flow

```
Local Computer                    VPS (67.211.219.94)
─────────────                     ──────────────────

[Trading App]                     
    Files                         
      │                           
      │ scp transfer               
      ├─────────────────────────> [Deployment Package]
      │                                   │
      │                                   │ Extract
      │                                   ▼
      │                           [Trading App Files]
      │                                   │
      │                                   │ Install Dependencies
      │                                   ▼
      │                           [Node.js + Go Ready]
      │                                   │
      │                                   │ Configure
      │                                   ▼
      │                           [.env files with API keys]
      │                                   │
      │                                   │ Build
      │                                   ▼
      │                           [Backend Binary + Frontend Build]
      │                                   │
      │                                   │ Create Services
      │                                   ▼
      │                           [systemd services]
      │                                   │
      │                                   │ Start
      │                                   ▼
Web Browser <────────────────── [🚀 Running Application]
http://67.211.219.94:3000             Port 3000 (Frontend)
                                       Port 8080 (Backend)
```

---

## ⏱️ Time Estimates

| Task | Time Required |
|------|---------------|
| Read DEPLOYMENT_SUMMARY.md | 5 minutes |
| Transfer files to VPS | 1 minute |
| Run vps-quick-setup.sh | 3-5 minutes |
| Configure API keys | 2 minutes |
| Build backend | 2-3 minutes |
| Build frontend | 3-5 minutes |
| Set up and start services | 2 minutes |
| Configure firewall | 1 minute |
| Verify deployment | 2 minutes |
| **Total** | **20-25 minutes** |

*These are estimates for a smooth deployment. Add time for troubleshooting if needed.*

---

## ✅ Success Criteria

You'll know your deployment succeeded when:

✅ You can SSH into your VPS  
✅ Both services show "active (running)" status  
✅ Backend responds to curl test  
✅ You can access http://67.211.219.94:3000 in your browser  
✅ You can log in with default credentials (admin/admin123)  
✅ Dashboard loads without errors  
✅ Firewall is active with correct rules  

---

## 🆘 If You Get Stuck

### 1. Check the Troubleshooting Section
The deployment_guide.md has a comprehensive troubleshooting section covering:
- Backend won't start
- Frontend won't start
- Can't access from browser
- Out of memory issues
- Port conflicts

### 2. Check the Logs
Logs usually tell you what went wrong:
```bash
# Backend logs
sudo tail -f /var/log/trading-backend.log

# Frontend logs
sudo tail -f /var/log/trading-frontend.log

# System logs
sudo journalctl -u trading-backend -n 50
sudo journalctl -u trading-frontend -n 50
```

### 3. Verify Each Step
Go back through the deployment guide and verify you didn't skip any steps.

### 4. Check System Resources
```bash
free -h      # Check memory
df -h        # Check disk space
top          # Check CPU usage
```

### 5. Common Quick Fixes

**Port already in use**:
```bash
sudo lsof -i :8080  # Find what's using port 8080
sudo lsof -i :3000  # Find what's using port 3000
```

**Service won't start**:
```bash
sudo systemctl restart trading-backend
sudo systemctl restart trading-frontend
```

**Need to rebuild**:
```bash
cd /root/trading-app/backend && go build -o trading-server ./cmd/main.go
cd /root/trading-app/frontend && npm run build
```

---

## 🔒 Security Notes

### Immediately After Deployment:

1. **Change default password**: Login and change from admin123 to something secure
2. **Update JWT secret**: Edit backend/.env and change JWT_SECRET
3. **Enable firewall**: Run `ufw enable` after verifying everything works
4. **Keep API keys private**: Never share .env files or commit them to git

### Regular Maintenance:

- Update system packages monthly: `apt-get update && apt-get upgrade`
- Backup database weekly: `cp backend/data/trading.db backups/`
- Monitor logs for suspicious activity
- Keep API keys secure

---

## 📞 File Locations Reference

**After transfer to VPS, everything will be at**:

```
/root/
├── trading-app-deployment.tar.gz    # Original package
├── vps-quick-setup.sh               # Setup script
└── trading-app/                     # Extracted application
    ├── backend/
    │   ├── .env                     # Backend config (you create this)
    │   ├── trading-server           # Compiled Go binary (after build)
    │   └── data/                    # Database and uploads
    ├── frontend/
    │   ├── .env.local               # Frontend config (you create this)
    │   └── .next/                   # Build output (after npm build)
    ├── deploy.sh                    # Automated deploy script
    ├── start.sh                     # Start services
    ├── stop.sh                      # Stop services
    ├── DEPLOYMENT.md                # Included deployment guide
    └── README.md                    # Application readme
```

**Log files**:
- `/var/log/trading-backend.log` - Backend logs
- `/var/log/trading-frontend.log` - Frontend logs

**Service files**:
- `/etc/systemd/system/trading-backend.service` - Backend service
- `/etc/systemd/system/trading-frontend.service` - Frontend service

---

## 🎓 Learning Resources

### Understanding the Commands

If you want to learn more about the commands used:

- `scp` - Secure copy (file transfer): `man scp`
- `ssh` - Secure shell (remote login): `man ssh`
- `systemctl` - Service management: `man systemctl`
- `tar` - Archive extraction: `man tar`
- `npm` - Node package manager: `npm help`
- `go` - Go programming tools: `go help`

### Understanding the Stack

- **Next.js**: https://nextjs.org/docs
- **Go**: https://go.dev/doc/
- **systemd**: https://systemd.io/
- **Ubuntu**: https://ubuntu.com/server/docs

---

## 🎯 Your Next Action

**Right now, you should**:

1. ✅ Open **deployment_guide.md** in a text editor or PDF viewer
2. ✅ Skim through it to get familiar (5 minutes)
3. ✅ Gather your API keys (OpenAlgo and Abacus.AI)
4. ✅ Start with Step 1 of the deployment guide
5. ✅ Follow each step carefully, checking verification at each stage

---

## 📈 After Successful Deployment

Once everything is running, you can:

- Access your app at: http://67.211.219.94:3000
- Log in with: admin / admin123
- Explore the dashboard
- Upload trading data
- Test the AI chat
- Create trading strategies
- Run backtests
- Monitor your portfolio

**Don't forget to**:
- Change the default password
- Set up regular database backups
- Monitor resource usage
- Keep the system updated

---

## 🏁 Final Checklist

Before you begin:

- [ ] I have SSH access to 67.211.219.94
- [ ] I have root password for the VPS
- [ ] I have my OpenAlgo API key ready
- [ ] I have my Abacus.AI API key ready
- [ ] I have read the DEPLOYMENT_SUMMARY.md
- [ ] I have the deployment_guide.md open and ready
- [ ] I have 20 minutes of uninterrupted time
- [ ] I'm ready to follow instructions step-by-step

---

## 🎉 Ready to Deploy!

**You have everything you need!**

The deployment process is straightforward when you follow the guide. Each step has:
- The exact command to run
- An explanation of what it does
- A way to verify it worked
- Troubleshooting if it doesn't

**Start here**: Open `deployment_guide.md` and begin with Step 1!

**Good luck! 🚀**

---

## 📝 Quick Command Reference

### Transfer Files
```bash
cd /home/ubuntu/code_artifacts/
scp trading-app-deployment.tar.gz root@67.211.219.94:/root/
scp vps-quick-setup.sh root@67.211.219.94:/root/
```

### Connect to VPS
```bash
ssh root@67.211.219.94
```

### Run Quick Setup
```bash
bash vps-quick-setup.sh
```

### Check Service Status
```bash
sudo systemctl status trading-backend trading-frontend
```

### View Logs
```bash
sudo tail -f /var/log/trading-backend.log
sudo tail -f /var/log/trading-frontend.log
```

### Restart Services
```bash
sudo systemctl restart trading-backend trading-frontend
```

---

**Created for VPS**: 67.211.219.94  
**Date**: October 25, 2025  
**Application**: AI Trading Platform  
**Support**: See deployment_guide.md for troubleshooting  
