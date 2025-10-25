
#!/bin/bash

# Start both backend and frontend services

echo "Starting AI Trading App..."
echo ""

sudo systemctl start trading-backend
sudo systemctl start trading-frontend

echo "Services started!"
echo ""
echo "Status:"
sudo systemctl status trading-backend --no-pager | head -n 5
echo ""
sudo systemctl status trading-frontend --no-pager | head -n 5
echo ""
echo "Access the application at:"
echo "  http://$(hostname -I | awk '{print $1}'):3000"
