#!/bin/bash

echo "=== Pokemon Monitor Auto-start Setup ==="
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "📁 Script location: $SCRIPT_DIR"
echo ""

sed -i "s|/opt/data/pokemon-monitor-standalone|$SCRIPT_DIR|g" pokemon-monitor.service

echo "📋 Installing systemd service..."
sudo cp pokemon-monitor.service /etc/systemd/system/

echo "🔄 Reloading systemd..."
sudo systemctl daemon-reload

echo "✅ Enabling auto-start on boot..."
sudo systemctl enable pokemon-monitor.service

echo "🚀 Starting service..."
sudo systemctl start pokemon-monitor.service

echo ""
echo "=== ✅ Setup Complete ==="
echo ""
echo "Useful commands:"
echo "  sudo systemctl status pokemon-monitor"
echo "  sudo systemctl stop pokemon-monitor"
echo "  sudo systemctl restart pokemon-monitor"
echo "  sudo systemctl disable pokemon-monitor"
echo "  tail -f monitor.log"
echo ""
