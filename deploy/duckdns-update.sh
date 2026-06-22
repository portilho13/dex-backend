#!/bin/bash
# Updates DuckDNS with your VPS IP
# Usage: bash duckdns-update.sh YOUR_DUCKDNS_TOKEN
#
# To auto-update, add to crontab:
#   crontab -e
#   */5 * * * * /opt/dex/duckdns-update.sh YOUR_TOKEN >> /var/log/duckdns.log 2>&1

TOKEN="$1"

if [ -z "$TOKEN" ]; then
    echo "Usage: $0 <duckdns-token>"
    exit 1
fi

echo url="https://www.duckdns.org/update?domains=dex-project&token=${TOKEN}&ip=" | curl -k -o /dev/null -s -K -
echo " $(date)"
