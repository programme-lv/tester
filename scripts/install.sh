#! /bin/bash

set -ex # exit on error, print commands

SCRIPT_DIR=$(dirname "$0")
cd "$SCRIPT_DIR/.."

# Install the Go binary
go install ./cmd/tester

# Install systemd service
SERVICE_FILE="tester.service"
SERVICE_PATH="/etc/systemd/system/$SERVICE_FILE"

echo "Installing systemd service..."

# Copy service file to systemd directory
sudo cp "$SCRIPT_DIR/$SERVICE_FILE" "$SERVICE_PATH"

# Reload systemd to recognize the new service
sudo systemctl daemon-reload

# Enable the service to start on boot
sudo systemctl enable tester.service

# Start the service now
sudo systemctl start tester.service

echo "Tester service installed and started successfully!"
echo "Use 'sudo systemctl status tester.service' to check status"
echo "Use 'sudo systemctl stop tester.service' to stop the service"
echo "Use 'sudo journalctl -u tester.service -f' to view logs"
