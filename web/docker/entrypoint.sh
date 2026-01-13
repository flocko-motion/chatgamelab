#!/bin/sh
set -eu

HTML_DIR="/usr/share/nginx/html"
ENV_JS="$HTML_DIR/env.js"
INDEX_HTML="$HTML_DIR/index.html"

# Fail fast for required env vars
: "${API_BASE_URL:?Must set API_BASE_URL}"
: "${AUTH0_DOMAIN:?Must set AUTH0_DOMAIN}"
: "${AUTH0_CLIENT_ID:?Must set AUTH0_CLIENT_ID}"
: "${AUTH0_AUDIENCE:?Must set AUTH0_AUDIENCE}"
: "${PUBLIC_URL:?Must set PUBLIC_URL}"

# Optional
AUTH0_REDIRECT_URI="${AUTH0_REDIRECT_URI:-}"

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

# Update index.html to use PUBLIC_URL for env.js path
if [ -n "$PUBLIC_URL" ] && [ "$PUBLIC_URL" != "/" ]; then
  # Remove trailing slash from PUBLIC_URL for script src
  PUBLIC_URL_NO_SLASH="${PUBLIC_URL%/}"
  sed -i "s|src=\"./env.js\"|src=\"${PUBLIC_URL_NO_SLASH}/env.js\"|g" "$INDEX_HTML"
fi

exec "$@"
