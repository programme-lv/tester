#! /bin/bash

set -ex # exit on error, print commands

SCRIPT_DIR=$(dirname "$0")
cd "$SCRIPT_DIR/.."

# Build and install the Go binary
go build -o tester ./cmd/tester
sudo cp tester /usr/local/bin/
rm tester

# Install systemd service
SERVICE_FILE="tester.service"
SERVICE_PATH="/etc/systemd/system/$SERVICE_FILE"

echo "Installing systemd service..."

# Create tester user if it doesn't exist
if ! id "tester" &>/dev/null; then
    echo "Creating tester user..."
    sudo useradd -m tester
fi

# Create configuration directory (prefer /usr/local/etc over /etc)
sudo mkdir -p /usr/local/etc/tester

# Copy service file to systemd directory
sudo cp "$SCRIPT_DIR/$SERVICE_FILE" "$SERVICE_PATH"

# Copy environment template if config doesn't exist
if [ ! -f "/usr/local/etc/tester/tester.env" ]; then
    echo "Creating environment configuration template..."
    sudo cp "$SCRIPT_DIR/tester.env.example" "/usr/local/etc/tester/tester.env"
    echo "Please edit /usr/local/etc/tester/tester.env to configure your environment variables"
fi

# Copy additional configuration assets
if [ -f "$SCRIPT_DIR/system.txt" ]; then
    sudo cp "$SCRIPT_DIR/system.txt" "/usr/local/etc/tester/system.txt"
fi
if [ -f "$SCRIPT_DIR/testlib.h" ]; then
    sudo cp "$SCRIPT_DIR/testlib.h" "/usr/local/etc/tester/testlib.h"
fi

# Reload systemd to recognize the new service
sudo systemctl daemon-reload

# Enable the service to start on boot
sudo systemctl enable tester.service

# Don't start the service automatically - let user configure it first
set +x # disable printing commands

echo ""
echo "Tester service installed successfully!"
echo ""
echo "IMPORTANT: Before starting the service, please:"
echo "1. Edit /usr/local/etc/tester/tester.env to configure your environment variables"
echo "2. Set at minimum the SUBM_REQ_QUEUE_URL variable"
echo ""
echo "After configuration, you can:"
echo "  sudo systemctl start tester.service    # Start the service"
echo "  sudo systemctl status tester.service   # Check status"
echo "  sudo systemctl stop tester.service     # Stop the service"
echo "  sudo journalctl -u tester.service -f   # View logs"
echo ""
echo "Service is enabled and will start automatically on boot."
