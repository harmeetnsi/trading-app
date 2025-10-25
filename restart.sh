
#!/bin/bash

# Restart both backend and frontend services

echo "Restarting AI Trading App..."
echo ""

sudo systemctl restart trading-backend
sudo systemctl restart trading-frontend

echo "Services restarted!"
echo ""
echo "Status:"
sudo systemctl status trading-backend --no-pager | head -n 5
echo ""
sudo systemctl status trading-frontend --no-pager | head -n 5
