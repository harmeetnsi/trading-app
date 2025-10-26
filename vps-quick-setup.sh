#!/bin/bash

################################################################################
# VPS Quick Setup Script for Trading Application
# This script automates the VPS preparation steps
# Run this on your VPS AFTER transferring the deployment package
################################################################################

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Print header
echo "=========================================================================="
echo "           Trading Application VPS Quick Setup Script"
echo "=========================================================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    print_error "Please run as root (use 'sudo bash vps-quick-setup.sh')"
    exit 1
fi

print_info "Starting VPS setup process..."
echo ""

# Step 1: Update system
print_info "Step 1/8: Updating system packages..."
apt-get update -qq && apt-get upgrade -y -qq
print_success "System packages updated"
echo ""

# Step 2: Install system dependencies
print_info "Step 2/8: Installing system dependencies..."
apt-get install -y -qq build-essential git curl wget tesseract-ocr net-tools
print_success "System dependencies installed"
echo ""

# Step 3: Check Go installation
print_info "Step 3/8: Checking Go installation..."
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go is already installed: $GO_VERSION"
else
    print_warning "Go is not installed! You mentioned it's already installed."
    print_warning "If you need to install Go, run:"
    echo "    wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz"
    echo "    tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz"
    echo "    export PATH=\$PATH:/usr/local/go/bin"
fi
echo ""

# Step 4: Install Node.js
print_info "Step 4/8: Installing Node.js 18.x..."
if command_exists node; then
    NODE_VERSION=$(node --version)
    print_warning "Node.js is already installed: $NODE_VERSION"
    print_info "Checking if we need to upgrade..."
fi

curl -fsSL https://deb.nodesource.com/setup_18.x | bash - 2>&1 | grep -v "^#" || true
apt-get install -y -qq nodejs
NODE_VERSION=$(node --version)
NPM_VERSION=$(npm --version)
print_success "Node.js installed: $NODE_VERSION"
print_success "npm installed: $NPM_VERSION"
echo ""

# Step 5: Extract deployment package
print_info "Step 5/8: Extracting deployment package..."
if [ -f "/root/trading-app-deployment.tar.gz" ]; then
    cd /root/
    tar -xzf trading-app-deployment.tar.gz
    print_success "Deployment package extracted to /root/trading-app/"
else
    print_error "Deployment package not found at /root/trading-app-deployment.tar.gz"
    print_error "Please transfer the file first using:"
    echo "    scp trading-app-deployment.tar.gz root@67.211.219.94:/root/"
    exit 1
fi
echo ""

# Step 6: Make scripts executable
print_info "Step 6/8: Making scripts executable..."
cd /root/trading-app/
chmod +x *.sh
print_success "Scripts are now executable"
echo ""

# Step 7: Create configuration files
print_info "Step 7/8: Creating configuration templates..."

# Backend .env
if [ ! -f "/root/trading-app/backend/.env" ]; then
    cat > /root/trading-app/backend/.env << 'EOF'
# Database configuration
DB_PATH=./data/trading.db

# Upload directory for files
UPLOAD_DIR=./data/uploads

# Server port (backend will run on this port)
PORT=8080

# OpenAlgo configuration (trading platform)
OPENALGO_URL=https://openalgo.mywire.org
OPENALGO_API_KEY=your_openalgo_api_key_here

# Abacus.AI API key (for AI chat features)
ABACUS_API_KEY=your_abacus_api_key_here

# JWT Secret for authentication (CHANGE THIS!)
JWT_SECRET=change_this_to_a_random_secure_string_123456

# Environment
ENVIRONMENT=production
EOF
    print_success "Backend configuration created at /root/trading-app/backend/.env"
    print_warning "IMPORTANT: You need to edit this file and add your API keys!"
else
    print_success "Backend configuration already exists"
fi

# Frontend .env.local
if [ ! -f "/root/trading-app/frontend/.env.local" ]; then
    cat > /root/trading-app/frontend/.env.local << 'EOF'
# Backend API URL (replace with your VPS IP)
NEXT_PUBLIC_API_URL=http://67.211.219.94:8080

# WebSocket URL for real-time updates
NEXT_PUBLIC_WS_URL=ws://67.211.219.94:8080/ws
EOF
    print_success "Frontend configuration created at /root/trading-app/frontend/.env.local"
else
    print_success "Frontend configuration already exists"
fi
echo ""

# Step 8: Install UFW firewall
print_info "Step 8/8: Installing and configuring firewall..."
apt-get install -y -qq ufw

print_info "Configuring firewall rules..."
ufw --force reset >/dev/null 2>&1
ufw default deny incoming >/dev/null 2>&1
ufw default allow outgoing >/dev/null 2>&1
ufw allow 22/tcp >/dev/null 2>&1
ufw allow 3000/tcp >/dev/null 2>&1
ufw allow 8080/tcp >/dev/null 2>&1

print_warning "Firewall configured but NOT enabled yet (to prevent lockout)"
print_info "To enable firewall, run: ufw enable"
print_success "Firewall setup complete"
echo ""

# Summary
echo "=========================================================================="
echo "                          Setup Complete! ✅"
echo "=========================================================================="
echo ""
print_success "VPS preparation is complete!"
echo ""
echo "Next steps:"
echo ""
echo "1. Edit backend configuration:"
echo "   nano /root/trading-app/backend/.env"
echo "   - Add your OpenAlgo API key"
echo "   - Add your Abacus.AI API key"
echo "   - Change the JWT_SECRET to a secure random string"
echo ""
echo "2. Build the backend:"
echo "   cd /root/trading-app/backend/"
echo "   go mod download"
echo "   go build -o trading-server ./cmd/main.go"
echo ""
echo "3. Build the frontend:"
echo "   cd /root/trading-app/frontend/"
echo "   npm install --legacy-peer-deps"
echo "   npm run build"
echo ""
echo "4. Set up and start services:"
echo "   Follow the deployment guide from Step 7 onwards"
echo ""
echo "5. Enable firewall (when ready):"
echo "   ufw enable"
echo ""
echo "📖 For detailed instructions, see: /root/trading-app/DEPLOYMENT.md"
echo ""
echo "=========================================================================="
echo "           🚀 Ready to build and deploy your application!"
echo "=========================================================================="
