#!/bin/sh
set -e

BACKEND_URL=${BACKEND_URL:-"/api"}
echo "Setting BACKEND_URL to: $BACKEND_URL"
sed -i "s|PLACEHOLDER_BACKEND_URL|${BACKEND_URL}|g" /usr/share/nginx/html/config.js

exec nginx -g 'daemon off;'
