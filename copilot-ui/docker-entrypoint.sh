#!/bin/sh
set -e

# For local dev, default to localhost:8000. For Docker, pass BACKEND_URL env var.
BACKEND_URL=${BACKEND_URL:-"http://localhost:8000"}
echo "Setting BACKEND_URL to: $BACKEND_URL"
# Replace whatever value is assigned to BACKEND_URL in config.js (placeholder or '/api')
sed -i "s|\(BACKEND_URL: *'\)[^']*\('\)|\1${BACKEND_URL}\2|g" /usr/share/nginx/html/config.js

exec nginx -g 'daemon off;'
