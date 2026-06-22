#!/bin/bash
set -e

echo "=== Installing dependencies ==="
apt update
apt install -y nginx certbot python3-certbot-nginx curl git

echo "=== Installing Go ==="
if ! command -v go &> /dev/null; then
    curl -LO https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
    rm go1.23.4.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
fi

echo "=== Installing Node.js ==="
if ! command -v node &> /dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    apt install -y nodejs
fi

echo "=== Creating directories ==="
mkdir -p /opt/dex
mkdir -p /var/www/dex/frontend

echo "=== Done ==="
echo ""
echo "Next steps:"
echo "  1. git clone your repos into ~/dex-backend and ~/dex-frontend"
echo "  2. bash ~/dex-backend/deploy/deploy.sh"
