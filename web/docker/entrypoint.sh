#!/bin/sh
set -eu

HTML_DIR="/usr/share/nginx/html"
ENV_JS="$HTML_DIR/env.js"

# Fail fast for required env vars
: "${API_BASE_URL:?Must set API_BASE_URL}"
: "${AUTH0_DOMAIN:?Must set AUTH0_DOMAIN}"
: "${AUTH0_CLIENT_ID:?Must set AUTH0_CLIENT_ID}"
: "${AUTH0_AUDIENCE:?Must set AUTH0_AUDIENCE}"

# Optional
AUTH0_REDIRECT_URI="${AUTH0_REDIRECT_URI:-}"

# Required - runtime config for the app
: "${PUBLIC_URL:?Must set PUBLIC_URL}"

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

exec "$@"
