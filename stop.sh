
#!/bin/bash

# Stop both backend and frontend services

echo "Stopping AI Trading App..."
echo ""

sudo systemctl stop trading-backend
sudo systemctl stop trading-frontend

echo "Services stopped!"
