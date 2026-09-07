#!/bin/bash
# Pokemon Monitor - Auto-start Setup

echo "=== Pokemon Monitor Auto-start Setup ==="
echo ""

# Get current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "📁 Script location: $SCRIPT_DIR"
echo ""

# Update service file with actual path
sed -i "s|/opt/data/pokemon-monitor-standalone|$SCRIPT_DIR|g" pokemon-monitor.service

# Copy service file
echo "📋 Installing systemd service..."
sudo cp pokemon-monitor.service /etc/systemd/system/

# Reload systemd
echo "🔄 Reloading systemd..."
sudo systemctl daemon-reload

# Enable service (auto-start on boot)
echo "✅ Enabling auto-start on boot..."
sudo systemctl enable pokemon-monitor.service

# Start service now
echo "🚀 Starting service..."
sudo systemctl start pokemon-monitor.service

echo ""
echo "=== ✅ Setup Complete ==="
echo ""
echo "Service will now:"
echo "  ✅ Start automatically on boot"
echo "  ✅ Restart automatically if crashed"
echo "  ✅ Log to monitor.log"
echo ""
echo "Useful commands:"
echo "  sudo systemctl status pokemon-monitor   # Check status"
echo "  sudo systemctl stop pokemon-monitor     # Stop service"
echo "  sudo systemctl restart pokemon-monitor  # Restart service"
echo "  sudo systemctl disable pokemon-monitor  # Disable auto-start"
echo "  tail -f monitor.log                     # Watch logs"
echo ""
