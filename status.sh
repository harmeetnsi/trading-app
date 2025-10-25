
#!/bin/bash

# Check status of services

echo "======================================"
echo "AI Trading App - Service Status"
echo "======================================"
echo ""

echo "Backend Service:"
sudo systemctl status trading-backend --no-pager
echo ""
echo "======================================"
echo ""

echo "Frontend Service:"
sudo systemctl status trading-frontend --no-pager
echo ""
echo "======================================"
echo ""

echo "Access the application at:"
echo "  http://$(hostname -I | awk '{print $1}'):3000"
