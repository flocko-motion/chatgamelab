#!/bin/sh
set -eu

HTML_DIR="/usr/share/nginx/html"
ENV_JS="$HTML_DIR/env.js"
INDEX_HTML="$HTML_DIR/index.html"
NGINX_TEMPLATE="/etc/nginx/nginx.conf.template"
NGINX_CONF="/etc/nginx/conf.d/default.conf"

# Fail fast for required env vars
: "${API_BASE_URL:?Must set API_BASE_URL}"
: "${AUTH0_DOMAIN:?Must set AUTH0_DOMAIN}"
: "${AUTH0_CLIENT_ID:?Must set AUTH0_CLIENT_ID}"
: "${AUTH0_AUDIENCE:?Must set AUTH0_AUDIENCE}"
: "${PUBLIC_URL:?Must set PUBLIC_URL}"

# Optional
AUTH0_REDIRECT_URI="${AUTH0_REDIRECT_URI:-}"

# Extract path from PUBLIC_URL (e.g., "https://example.com/foo" -> "/foo")
# Remove protocol and domain, keep only the path
PUBLIC_URL_PATH=$(echo "$PUBLIC_URL" | sed -E 's|^https?://[^/]*||' | sed 's|/$||')

# If PUBLIC_URL_PATH is empty (root deployment), use a special nginx config
if [ -z "$PUBLIC_URL_PATH" ]; then
  # Root deployment - use simple config
  cat > "$NGINX_CONF" <<'NGINX_ROOT'
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;

    location = /env.js {
        add_header Cache-Control "no-store";
        try_files $uri =404;
    }

    location = /index.html {
        add_header Cache-Control "no-cache";
        try_files $uri =404;
    }

    location /assets/ {
        add_header Cache-Control "public, max-age=31536000, immutable";
        try_files $uri =404;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
NGINX_ROOT
else
  # Subpath deployment - generate config from template
  export PUBLIC_URL_PATH
  envsubst '${PUBLIC_URL_PATH}' < "$NGINX_TEMPLATE" > "$NGINX_CONF"
fi

# Generate env.js (public values only - no secrets!)
cat > "$ENV_JS" <<EOF
window.__APP_CONFIG__ = {
  API_BASE_URL: "$(printf '%s' "$API_BASE_URL" | sed 's/"/\\"/g')",
  AUTH0_DOMAIN: "$(printf '%s' "$AUTH0_DOMAIN" | sed 's/"/\\"/g')",
  AUTH0_CLIENT_ID: "$(printf '%s' "$AUTH0_CLIENT_ID" | sed 's/"/\\"/g')",
  AUTH0_AUDIENCE: "$(printf '%s' "$AUTH0_AUDIENCE" | sed 's/"/\\"/g')",
  AUTH0_REDIRECT_URI: "$(printf '%s' "$AUTH0_REDIRECT_URI" | sed 's/"/\\"/g')",
  PUBLIC_URL: "$(printf '%s' "$PUBLIC_URL" | sed 's/"/\\"/g')"
};
EOF

# Update index.html to use PUBLIC_URL_PATH for all asset paths
if [ -n "$PUBLIC_URL_PATH" ]; then
  # Patch env.js path
  sed -i "s|src=\"/env.js\"|src=\"${PUBLIC_URL_PATH}/env.js\"|g" "$INDEX_HTML"
  # Patch all /assets/ references to use the base path
  sed -i "s|src=\"/assets/|src=\"${PUBLIC_URL_PATH}/assets/|g" "$INDEX_HTML"
  sed -i "s|href=\"/assets/|href=\"${PUBLIC_URL_PATH}/assets/|g" "$INDEX_HTML"
  # Patch favicon/logo path
  sed -i "s|href=\"/logo.png\"|href=\"${PUBLIC_URL_PATH}/logo.png\"|g" "$INDEX_HTML"
fi

exec "$@"
