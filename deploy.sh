
#!/bin/bash

# AI Trading App Deployment Script
# This script deploys the application on Ubuntu VPS

set -e

echo "======================================"
echo "AI Trading App - Deployment Script"
echo "======================================"
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then 
   echo "Please do not run this script as root"
   exit 1
fi

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "Step 1: Installing dependencies..."
echo ""

# Install tesseract-ocr for image processing (optional, for OCR)
echo "Installing system dependencies..."
sudo apt-get update
sudo apt-get install -y tesseract-ocr

echo ""
echo "Step 2: Setting up environment variables..."
echo ""

# Create .env file if it doesn't exist
if [ ! -f "$SCRIPT_DIR/backend/.env" ]; then
    echo "Creating backend .env file..."
    cp "$SCRIPT_DIR/backend/.env.example" "$SCRIPT_DIR/backend/.env"
    
    echo ""
    echo "Please configure the following in backend/.env:"
    echo "1. OPENALGO_API_KEY - Your OpenAlgo API key"
    echo "2. ABACUS_API_KEY - Your Abacus.AI API key"
    echo ""
    read -p "Enter your OpenAlgo API key: " OPENALGO_KEY
    read -p "Enter your Abacus.AI API key: " ABACUS_KEY
    
    sed -i "s/your_openalgo_api_key_here/$OPENALGO_KEY/" "$SCRIPT_DIR/backend/.env"
    sed -i "s/your_abacus_api_key_here/$ABACUS_KEY/" "$SCRIPT_DIR/backend/.env"
fi

# Create frontend .env.local
if [ ! -f "$SCRIPT_DIR/frontend/.env.local" ]; then
    echo "Creating frontend .env.local..."
    SERVER_IP=$(hostname -I | awk '{print $1}')
    cat > "$SCRIPT_DIR/frontend/.env.local" << EOF
NEXT_PUBLIC_API_URL=http://$SERVER_IP:8080
NEXT_PUBLIC_WS_URL=ws://$SERVER_IP:8080/ws
EOF
    echo "Frontend configured for IP: $SERVER_IP"
fi

echo ""
echo "Step 3: Building backend..."
echo ""

cd "$SCRIPT_DIR/backend"
go mod download
go build -o trading-server ./cmd/main.go

echo ""
echo "Step 4: Building frontend..."
echo ""

cd "$SCRIPT_DIR/frontend"
npm install --legacy-peer-deps
npm run build

echo ""
echo "Step 5: Setting up systemd services..."
echo ""

# Create systemd service for backend
sudo tee /etc/systemd/system/trading-backend.service > /dev/null << EOF
[Unit]
Description=AI Trading Backend
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$SCRIPT_DIR/backend
Environment="PATH=/usr/local/go/bin:/usr/bin:/bin"
EnvironmentFile=$SCRIPT_DIR/backend/.env
ExecStart=$SCRIPT_DIR/backend/trading-server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Create systemd service for frontend
sudo tee /etc/systemd/system/trading-frontend.service > /dev/null << EOF
[Unit]
Description=AI Trading Frontend
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$SCRIPT_DIR/frontend
Environment="PATH=/usr/bin:/bin:$HOME/.nvm/versions/node/v18.20.8/bin"
Environment="NODE_ENV=production"
ExecStart=$HOME/.nvm/versions/node/v18.20.8/bin/npm start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
sudo systemctl daemon-reload

# Enable services
sudo systemctl enable trading-backend
sudo systemctl enable trading-frontend

echo ""
echo "Step 6: Starting services..."
echo ""

sudo systemctl restart trading-backend
sudo systemctl restart trading-frontend

echo ""
echo "======================================"
echo "Deployment completed successfully!"
echo "======================================"
echo ""
echo "Service Status:"
sudo systemctl status trading-backend --no-pager
echo ""
sudo systemctl status trading-frontend --no-pager
echo ""
echo "Access the application at:"
echo "  http://$(hostname -I | awk '{print $1}'):3000"
echo ""
echo "To view logs:"
echo "  Backend:  sudo journalctl -u trading-backend -f"
echo "  Frontend: sudo journalctl -u trading-frontend -f"
echo ""
echo "To manage services:"
echo "  sudo systemctl start|stop|restart trading-backend"
echo "  sudo systemctl start|stop|restart trading-frontend"
echo ""
