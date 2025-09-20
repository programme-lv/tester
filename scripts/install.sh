#! /bin/bash

set -e # exit on error, print commands

SCRIPT_PATH="$(readlink -f "$0")"
SCRIPT_DIR="$(dirname "$SCRIPT_PATH")"
cd "$SCRIPT_DIR/.."
DEFAULTS_DIR="$SCRIPT_DIR/defaults"
echo "Script directory: $SCRIPT_DIR"
echo "Defaults directory: $DEFAULTS_DIR"
echo ""

#########################
# INSTALL TESTER BINARY
#########################
echo "## Compile and install tester binary..."
go build -o tester ./cmd/tester
sudo cp tester /usr/local/bin/
echo "Tester binary installed in /usr/local/bin/tester"
rm tester
echo ""

#########################
# COPY CONFIG FILES
#########################
echo "## Copying config files if not present..."
echo "Creating /usr/local/etc/tester directory..."
TESTER_ETC_DIR="/usr/local/etc/tester"
echo "Tester etc directory: $TESTER_ETC_DIR"
sudo mkdir -p "$TESTER_ETC_DIR"
CONFIG_FILES=("tester.env" "system.txt" "testlib.h")
for file in "${CONFIG_FILES[@]}"; do
    if [ ! -f "$TESTER_ETC_DIR/$file" ]; then
        sudo cp "$DEFAULTS_DIR/$file" "$TESTER_ETC_DIR/$file"
        echo "Copied $file to $TESTER_ETC_DIR/$file"
    else 
        echo "$file already exists in $TESTER_ETC_DIR/$file"
    fi
done
echo ""

#########################
# INSTALL SYSTEMD SERVICE
#########################
echo "## Installing systemd service..."
SERVICE_PATH="/etc/systemd/system/tester.service"
sudo cp "$DEFAULTS_DIR/tester.service" "$SERVICE_PATH"
echo "Service path: $SERVICE_PATH"
echo "Reloading systemd to recognize the new service..."
sudo systemctl daemon-reload
echo ""

#########################
# PRINT SUCCESS MESSAGE
#########################
cat << 'EOF'
Tester service installed successfully!

IMPORTANT: before starting the service, please:
1. Configure environment variables in /usr/local/etc/tester/tester.env

After configuration, you can, e.g.:

systemctl enable tester.service
systemctl status tester.service
journalctl -u tester.service -f

etc.
EOF
