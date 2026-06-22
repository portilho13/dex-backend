#!/bin/bash
set -e

DOMAIN="dex-project.duckdns.org"
BACKEND_DIR="$HOME/dex-backend"
FRONTEND_DIR="$HOME/dex-frontend"

export PATH=$PATH:/usr/local/go/bin

echo "=== Building backend ==="
cd "$BACKEND_DIR"
go build -o /opt/dex/dex-backend .

echo "=== Setting up .env ==="
if [ ! -f /opt/dex/.env ]; then
    cp "$BACKEND_DIR/deploy/env.example" /opt/dex/.env
    echo "!!! Edit /opt/dex/.env with your API keys !!!"
fi

echo "=== Building frontend ==="
cd "$FRONTEND_DIR"
npm install
npm run build
rm -rf /var/www/dex/frontend/*
cp -r dist/* /var/www/dex/frontend/

echo "=== Setting up Nginx ==="
cp "$BACKEND_DIR/deploy/nginx.conf" /etc/nginx/sites-available/dex
ln -sf /etc/nginx/sites-available/dex /etc/nginx/sites-enabled/dex
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx

echo "=== Setting up SSL ==="
if [ ! -d "/etc/letsencrypt/live/$DOMAIN" ]; then
    certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos --email admin@$DOMAIN --redirect
fi

echo "=== Setting up systemd service ==="
cp "$BACKEND_DIR/deploy/dex-backend.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable dex-backend
systemctl restart dex-backend

echo "=== Fixing permissions ==="
chown -R www-data:www-data /opt/dex
chown -R www-data:www-data /var/www/dex

echo ""
echo "=== Deploy complete ==="
echo "Site: https://$DOMAIN"
echo ""
echo "Useful commands:"
echo "  systemctl status dex-backend"
echo "  journalctl -u dex-backend -f"
echo "  systemctl restart dex-backend"
