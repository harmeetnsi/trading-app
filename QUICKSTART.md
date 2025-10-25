# Quick Start Guide

Get your AI Trading App up and running in 5 minutes!

## For Novice Users (Copy-Paste Commands)

### Step 1: Navigate to the project directory

```bash
cd /root/trading-app
```

### Step 2: Deploy the application

```bash
./deploy.sh
```

When prompted:
- Enter your **OpenAlgo API key** (from https://openalgo.mywire.org)
- Enter your **Abacus.AI API key** (from https://abacus.ai)

Wait for the deployment to complete (~5-10 minutes).

### Step 3: Access the application

Open your browser and go to:
```
http://YOUR_VPS_IP:3000
```

Replace `YOUR_VPS_IP` with your actual VPS IP address (e.g., 67.211.219.94).

### Step 4: Login

- Username: `admin`
- Password: `admin123`

⚠️ **Change this password after first login!**

## Daily Usage

### Starting the app
```bash
cd /root/trading-app
./start.sh
```

### Stopping the app
```bash
cd /root/trading-app
./stop.sh
```

### Checking if it's running
```bash
cd /root/trading-app
./status.sh
```

### Viewing logs (if something goes wrong)
```bash
cd /root/trading-app
./logs.sh backend    # For backend issues
./logs.sh frontend   # For frontend issues
```

## Features Overview

### 📊 Dashboard
- View your portfolio value
- See today's profit/loss
- Check open positions
- View recent trades

### 💬 AI Chat
- Ask questions about trading
- Upload files for analysis:
  - Pine Scripts (trading strategies)
  - CSV files (trade data)
  - Images (charts)
  - PDFs (reports)

### 🎯 Strategies
- View your trading strategies
- Activate or pause strategies
- Run backtests
- See performance metrics

## Using on Mobile

### Install as App (Android)
1. Open the trading app in Chrome
2. Tap the menu (three dots)
3. Tap "Install app" or "Add to Home screen"

### Install as App (iPhone)
1. Open the trading app in Safari
2. Tap the Share button
3. Tap "Add to Home Screen"

## Common Issues

### "Cannot connect to server"
```bash
cd /root/trading-app
./restart.sh
```

### "Login not working"
Make sure you're using:
- Username: `admin`
- Password: `admin123`

### "Nothing loads"
Check if services are running:
```bash
cd /root/trading-app
./status.sh
```

Both services should show "active (running)".

## Getting Help

If something doesn't work:
1. Try restarting: `./restart.sh`
2. Check logs: `./logs.sh backend` or `./logs.sh frontend`
3. Check if services are running: `./status.sh`

## Important Notes

- **Always use the scripts** in `/root/trading-app` directory
- **Don't delete** the `data` folder (contains your database)
- **Backup** your database regularly (it's in `backend/data/trading.db`)
- **Change the default password** after first login

## What to Do First

1. ✅ Login to the app
2. ✅ Change your password
3. ✅ Go to Dashboard to see overview
4. ✅ Try the AI Chat - ask "What can you help me with?"
5. ✅ Upload a Pine Script or CSV file in Chat
6. ✅ Check Strategies page

---

**That's it! You're ready to start using your AI Trading App! 🎉**
