```bash
# Install and start the service
./scripts/install.sh

# Check service status
sudo systemctl status tester.service

# View logs
sudo journalctl -u tester.service -f

# Stop the service
sudo systemctl stop tester.service

# Disable auto-start
sudo systemctl disable tester.service
```