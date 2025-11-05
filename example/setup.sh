#!/bin/bash
set -euo pipefail

APP_NAME="testapp"
REGISTRY="ghcr.io/zeitlos/knockknock/testapp"
BASE_DIR="/opt/${APP_NAME}"
VERSIONS_DIR="${BASE_DIR}/versions"

# Check root
if [[ $EUID -ne 0 ]]; then
   echo "Error: This script must be run as root"
   exit 1
fi

# Check arguments
if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.0.1"
    exit 1
fi

VERSION="$1"

echo "Setting up ${APP_NAME} version ${VERSION}..."

# Check oras is installed
if ! command -v oras &> /dev/null; then
    echo "Error: oras command not found"
    echo "Install from: https://oras.land/docs/installation"
    exit 1
fi

# Stop the service if it's running
if systemctl is-active --quiet ${APP_NAME}.service 2>/dev/null; then
    echo "Stopping ${APP_NAME} service..."
    systemctl stop ${APP_NAME}.service
fi

# Create user if doesn't exist
if ! id -u ${APP_NAME} &>/dev/null; then
    echo "Creating ${APP_NAME} user..."
    useradd --system --no-create-home --shell /bin/false ${APP_NAME}
fi

# Create directory structure
echo "Creating directories..."
mkdir -p "${VERSIONS_DIR}/${VERSION}"

# Clean up old versions and symlinks
if [[ -d "${VERSIONS_DIR}" ]]; then
    echo "Removing old versions..."
    for dir in "${VERSIONS_DIR}"/*; do
        if [[ -d "$dir" && "$dir" != "${VERSIONS_DIR}/${VERSION}" ]]; then
            echo "  Removing $(basename "$dir")"
            rm -rf "$dir"
        fi
    done
fi

# Remove all previous-* symlinks and old current symlink
echo "Removing old symlinks..."
rm -f "${BASE_DIR}"/previous-*
rm -f "${BASE_DIR}/current"

# Pull binary from ORAS
echo "Pulling ${APP_NAME}:${VERSION}..."
cd "${VERSIONS_DIR}/${VERSION}"
oras pull "${REGISTRY}:${VERSION}"

# Make binary executable
chmod +x ${APP_NAME}

# Set ownership
chown -R ${APP_NAME}:${APP_NAME} "${BASE_DIR}"

# Create current symlink
echo "Creating symlink..."
ln -sfn "${VERSIONS_DIR}/${VERSION}" "${BASE_DIR}/current"

# Install systemd service
echo "Installing systemd service..."
cat > /etc/systemd/system/${APP_NAME}.service <<EOF
[Unit]
Description=TestApp Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${APP_NAME}
Group=${APP_NAME}
WorkingDirectory=${BASE_DIR}
ExecStart=${BASE_DIR}/current/${APP_NAME}
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
systemctl daemon-reload
systemctl enable ${APP_NAME}.service
systemctl restart ${APP_NAME}.service

echo "Done! Service is running."
echo ""
echo "Check status: systemctl status ${APP_NAME}"
echo "View logs:    journalctl -u ${APP_NAME} -f"