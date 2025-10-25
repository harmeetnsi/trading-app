
#!/bin/bash

# View logs for backend and frontend

if [ "$1" = "backend" ]; then
    echo "Viewing backend logs (Ctrl+C to exit)..."
    sudo journalctl -u trading-backend -f
elif [ "$1" = "frontend" ]; then
    echo "Viewing frontend logs (Ctrl+C to exit)..."
    sudo journalctl -u trading-frontend -f
else
    echo "Usage: ./logs.sh [backend|frontend]"
    echo ""
    echo "Examples:"
    echo "  ./logs.sh backend   - View backend logs"
    echo "  ./logs.sh frontend  - View frontend logs"
fi
