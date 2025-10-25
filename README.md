
# AI Trading Web Application

A complete AI-powered trading application with strategy management, backtesting, and real-time trading capabilities. Designed for easy deployment on a 2GB RAM VPS.

## 🚀 Features

### Backend (Go)
- **REST API Server**: Fast and efficient API endpoints
- **WebSocket Support**: Real-time communication for chat and updates
- **AI Chat Integration**: Powered by Abacus.AI for intelligent trading assistance
- **File Upload & Processing**:
  - Pine Scripts (.pine, .txt) - Strategy extraction and analysis
  - CSV/Excel (.csv, .xlsx) - Trade data analysis
  - Images (.jpg, .png) - Chart analysis support
  - PDFs (.pdf) - Document analysis
- **OpenAlgo Integration**: Connect to your OpenAlgo instance for live trading
- **Strategy Management**: Store, manage, and deploy trading strategies
- **Backtesting Engine**: Test strategies on historical data
- **SQLite Database**: Lightweight database optimized for low memory
- **Authentication**: Secure login with JWT tokens

### Frontend (Next.js)
- **Responsive Design**: Works on mobile and desktop browsers
- **WhatsApp-style Chat**: Real-time AI chat interface
- **File Upload UI**: Drag & drop support with progress indicators
- **Trading Dashboard**: Live P&L, positions, trades, and metrics
- **Strategy Management**: View, deploy, pause strategies with backtest results
- **PWA Support**: Install to home screen, offline support
- **Authentication Pages**: Login and registration

## 📋 Prerequisites

- **OS**: Ubuntu 20.04 or later
- **RAM**: Minimum 2GB (optimized for low memory)
- **Node.js**: v18.20.8 (already installed on your VPS)
- **Go**: 1.21.6 or later (already installed on your VPS)
- **OpenAlgo**: Running instance with API key
- **Abacus.AI**: API key for AI features

## 🔧 Quick Start (One Command Deployment)

### Step 1: Clone or upload the project to your VPS

```bash
cd /root
# If you're uploading, ensure the project is in /root/trading-app
```

### Step 2: Run the deployment script

```bash
cd /root/trading-app
chmod +x *.sh
./deploy.sh
```

The deployment script will:
1. Install system dependencies
2. Prompt you for API keys (OpenAlgo and Abacus.AI)
3. Build the backend and frontend
4. Set up systemd services for auto-start
5. Start the application

### Step 3: Access the application

Open your browser and go to:
```
http://YOUR_VPS_IP:3000
```

**Default Login Credentials:**
- Username: `admin`
- Password: `admin123`

⚠️ **Important**: Change the default password after first login!

## 📱 Mobile Access

### Install as PWA (Progressive Web App)

1. **On Android Chrome**:
   - Open the app in Chrome
   - Tap the menu (⋮) and select "Install app" or "Add to Home screen"
   - The app will appear on your home screen

2. **On iOS Safari**:
   - Open the app in Safari
   - Tap the Share button
   - Tap "Add to Home Screen"

## 🛠️ Management Scripts

All management scripts are in the project root:

```bash
# Start services
./start.sh

# Stop services
./stop.sh

# Restart services
./restart.sh

# Check status
./status.sh

# View logs
./logs.sh backend   # Backend logs
./logs.sh frontend  # Frontend logs
```

## 📊 Usage Guide

### 1. Dashboard
- View portfolio overview with live P&L
- See open positions and recent trades
- Quick actions for chat, strategies, and orders

### 2. AI Chat
- Ask questions about trading strategies
- Upload files for analysis:
  - Pine Scripts: AI will analyze the strategy
  - CSV files: Get trade data insights
  - Images: Chart analysis
  - PDFs: Extract and analyze reports
- Get real-time responses

### 3. Strategy Management
- View all uploaded strategies
- Activate/pause strategies
- Run backtests with custom parameters:
  - Date range
  - Initial capital
  - Symbol and exchange
- View backtest results with metrics:
  - Total return
  - Win rate
  - Max drawdown
  - Sharpe ratio

### 4. Creating Strategies
1. Go to Chat
2. Upload a Pine Script file
3. AI will analyze it and create a strategy
4. Go to Strategies to view and manage it

## 🔑 API Keys Configuration

### OpenAlgo API Key
1. Log in to your OpenAlgo instance at `https://openalgo.mywire.org`
2. Generate an API key from settings
3. Enter it during deployment or add to `/root/trading-app/backend/.env`

### Abacus.AI API Key
1. Sign up at [Abacus.AI](https://abacus.ai)
2. Generate an API key from your account
3. Enter it during deployment or add to `/root/trading-app/backend/.env`

## 🗂️ Project Structure

```
trading-app/
├── backend/              # Go backend
│   ├── cmd/             # Main application entry
│   ├── internal/        # Internal packages
│   │   ├── auth/        # Authentication
│   │   ├── database/    # Database layer
│   │   ├── models/      # Data models
│   │   ├── handlers/    # HTTP handlers
│   │   ├── websocket/   # WebSocket server
│   │   ├── ai/          # AI integration
│   │   ├── fileprocessor/ # File processing
│   │   ├── openalgo/    # OpenAlgo client
│   │   └── strategy/    # Strategy & backtesting
│   ├── pkg/             # Public packages
│   └── data/            # SQLite database & uploads
├── frontend/            # Next.js frontend
│   ├── app/             # Pages (App Router)
│   ├── components/      # React components
│   ├── lib/             # Utilities & API client
│   └── public/          # Static assets
├── deploy.sh            # Deployment script
├── start.sh             # Start services
├── stop.sh              # Stop services
├── restart.sh           # Restart services
├── logs.sh              # View logs
└── status.sh            # Check status
```

## 🔒 Security Notes

1. **Change Default Password**: After first login, create a new user and remove the default admin
2. **API Keys**: Keep your API keys secure, never commit them to version control
3. **Firewall**: Consider setting up a firewall to restrict access to ports 8080 and 3000
4. **HTTPS**: For production, use a reverse proxy (nginx) with SSL/TLS

## 🐛 Troubleshooting

### Services won't start
```bash
# Check service status
./status.sh

# View logs for errors
./logs.sh backend
./logs.sh frontend

# Restart services
./restart.sh
```

### Port already in use
```bash
# Find process using port 8080 (backend)
sudo lsof -i :8080

# Find process using port 3000 (frontend)
sudo lsof -i :3000

# Kill the process if needed
sudo kill -9 <PID>
```

### Cannot connect to OpenAlgo
- Verify your OpenAlgo instance is running
- Check the API key in `/root/trading-app/backend/.env`
- Ensure the URL is correct (default: https://openalgo.mywire.org)

### Database issues
```bash
# The database is created automatically at:
# /root/trading-app/backend/data/trading.db

# To reset the database, stop services and delete it:
./stop.sh
rm /root/trading-app/backend/data/trading.db
./start.sh
```

### Out of memory
If you experience memory issues on 2GB RAM:
```bash
# Monitor memory usage
free -h

# Check which service is using memory
top

# Restart services to free memory
./restart.sh
```

## 📈 Performance Optimization

The application is optimized for 2GB RAM:
- Database connection pooling (max 5 connections)
- Efficient data structures
- Lazy loading of components
- Pagination for large datasets
- Image and asset optimization

## 🔄 Updates

To update the application:

```bash
cd /root/trading-app
./stop.sh

# Pull latest changes or upload new files
# Then rebuild and restart:

cd backend
go build -o trading-server ./cmd/main.go

cd ../frontend
npm run build

cd ..
./start.sh
```

## 📞 Support

For issues or questions:
1. Check the troubleshooting section
2. View logs: `./logs.sh backend` or `./logs.sh frontend`
3. Check service status: `./status.sh`

## 📄 License

This project is for personal use. Ensure compliance with OpenAlgo and Abacus.AI terms of service.

## 🙏 Acknowledgments

- OpenAlgo for trading infrastructure
- Abacus.AI for AI capabilities
- Go and Next.js communities

---

**Made with ❤️ for traders who want AI-powered insights**
