[![CI to GHCR](https://github.com/flocko-motion/chatgamelab/actions/workflows/docker-image.yml/badge.svg)](https://github.com/flocko-motion/chatgamelab/actions/workflows/docker-image.yml)

# chatgamelab

Educational GPT-Chat based text adventure lab.

Create your own text adventure games and play them with your friends.

- Learn, how GPT can be used to create interactive stories.
- Use debug-mode to see the raw requests and responses of the GPT model

For a detailed explanation of how the AI game engine works, see [TECHNOLOGY.md](TECHNOLOGY.md).

**[Roadmap](https://flocko-motion.github.io/chatgamelab/):** what's next and in which order, generated from the issues; [epics only](https://flocko-motion.github.io/chatgamelab/#epics) for the overview.

## Contributing

Work happens on `development`; `main` is what the public uses. You branch off
`development`, open a pull request against it, automated checks run, a human
reviews, and once it is merged the development server deploys itself from it.
Releasing to `main` — and so to production — is a separate step performed by
the project lead.

**[CONTRIBUTING.md](CONTRIBUTING.md) is the full path**, written for someone
doing it here for the first time. Read it before your first pull request; it
covers the branches, the checks, what review is for, and the commands
(`make rebase`, `make pr`, `make release`) that drive each step.

If you are working on ChatGameLab in any capacity, including as a web designer,
you need Git basics first. There are friendly clients for Mac such as
[GitHub Desktop](https://desktop.github.com/) and
[Sourcetree](https://www.sourcetreeapp.com/).

## Requirements

- **Go** (version in `go/go.mod`)
- **Node.js and npm**
- **podman** (rootless) and **docker-compose**, or Docker with `SCHEMAF_RUNTIME=docker` — see [schemaf's container runtime](https://github.com/flocko-motion/schemaf/blob/main/INSTALL.md#container-runtime)
- **Auth0 account** (for authentication)
- **OpenAI API key** (to play games)

## Setup

The platform is a [schemaf](https://github.com/flocko-motion/schemaf) app: the Go server in `go/` is the single gateway, serving `/api/*` and the frontend in `frontend/`. The game engine lives in `go/engine/`, its own Go module.

The dev environment's values live outside the repository, in `~/.chatgamelab/dev/etc/env`. It needs at least the database password:

```bash
mkdir -p ~/.chatgamelab/dev/etc
echo "DB_PASS=changeme" > ~/.chatgamelab/dev/etc/env
```

## Development

`./schemaf.sh` is the entry point for everything:

```bash
./schemaf.sh codegen               # regenerate glue code after changing migrations, queries or endpoints
./schemaf.sh dev                   # list the services
./schemaf.sh dev db,backend        # Postgres + the Go server
./schemaf.sh dev all               # Postgres + the Go server + the Vite dev server
./schemaf.sh test                  # codegen, then all tests
```

The app runs at **http://localhost:7100**. Ports derive from `port` in `schemaf.toml`: the server on 7100, the Vite dev server on 7102, Postgres on 7103. In dev the server passes every request outside `/api/*` to Vite.

Development tokens for any user:

```bash
./schemaf.sh auth token <user-id>
```

## Production

Production uses pre-built Docker images from GitHub Container Registry (GHCR). Images are automatically built by GitHub Actions when you push to `main` or `development` branches.

**Image tags:**
- `main` branch → `:latest` tag (production)
- `development` branch → `:dev` tag (staging)
- All commits also get SHA tags (e.g., `:abc1234`) for rollbacks

### Deployment Steps

1. **Authenticate to GitHub Container Registry** on your server:
   ```bash
   # Create a GitHub Personal Access Token with read:packages scope
   # https://github.com/settings/tokens/new
   
   echo "YOUR_GITHUB_TOKEN" | docker login ghcr.io -u YOUR_USERNAME --password-stdin
   ```

2. **Copy deployment files** to your server:
   - `docker-compose.yml` - Container orchestration
   - Create systemd service file (see below)

3. **Configure and start** via systemd (recommended) or manually

### Required Environment Variables

**Do NOT use `.env` files in production.** Environment variables must be injected externally via your hosting provider or systemd.

```bash
# Image tag (controls which branch's images to use)
IMAGE_TAG=latest  # or 'dev' for development branch

# Database
DB_PASSWORD=secure_password_here

# Backend
PUBLIC_URL=https://yourdomain.com

# Auth0 (used by both backend and frontend)
AUTH0_DOMAIN=your.auth0.domain
AUTH0_AUDIENCE=your.auth0.audience
AUTH0_CLIENT_ID=your_client_id

# Frontend (runtime config - injected at container startup)
API_BASE_URL=https://yourdomain.com

# Ports
PORT_EXPOSED=80
```

> **Security:** Frontend config is PUBLIC (readable by browser). Never put secrets here. Auth0 SPA values are safe - security relies on Auth0's allowed origins and backend JWT validation.

### Option 1: Hosting Provider (Render, Railway, etc.)

Set environment variables in your provider's dashboard, then deploy.

### Option 2: Self-hosted with systemd

Create a systemd service for automatic startup and management:

```bash
sudo nano /etc/systemd/system/chatgamelab.service
```
With the following content:

```ini
[Unit]
Description=ChatGameLab Production
After=docker.service network-online.target
Requires=docker.service
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/path/to/chatgamelab
User=youruser
Group=youruser

# Environment variables
Environment="IMAGE_TAG=latest" # production: latest, development: dev
Environment="DB_PASSWORD=secure_password" # change this to a random password
Environment="AUTH0_DOMAIN=your.auth0.domain" # configure this according to your Auth0 account
Environment="AUTH0_AUDIENCE=your.auth0.audience" # configure this according to your Auth0 account
Environment="AUTH0_CLIENT_ID=your_client_id" # configure this according to your Auth0 account
Environment="API_BASE_URL=https://yourdomain.com/api" # configure this according to your domain
Environment="PUBLIC_URL=https://yourdomain.com" # configure this according to your domain
Environment="PORT_FRONTEND=8000" # port internal to webserver for exposing webapp
Environment="PORT_BACKEND=8001" # port internal to webserver for exposing backend
Environment="SESSION_RETENTION_HOURS=48" # game sessions are deleted this long after their last activity (0 = keep forever)

# Commands
ExecStartPre=/usr/bin/docker compose pull
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down

# Restart policy
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
```

**Enable and start:**
```bash
sudo systemctl daemon-reload
sudo systemctl enable chatgamelab.service
sudo systemctl start chatgamelab.service
sudo systemctl status chatgamelab.service
```

### Updating to Latest Images

When new code is pushed to GitHub, images are automatically built. To update your server:

```bash
# Via systemd (pulls latest images automatically)
sudo systemctl restart chatgamelab.service

# Or manually
IMAGE_TAG=latest docker compose pull
IMAGE_TAG=latest docker compose up -d
```

### Rollback to Previous Version

```bash
# Use the short commit SHA from GitHub Actions
IMAGE_TAG=abc1234 docker compose pull
IMAGE_TAG=abc1234 docker compose up -d
```

### Useful Commands

```bash
# View logs
docker compose logs -f
docker compose logs -f backend

# Check status
docker compose ps
sudo systemctl status chatgamelab.service

# Stop/start
docker compose down
IMAGE_TAG=latest docker compose up -d

# Or via systemd
sudo systemctl stop chatgamelab.service
sudo systemctl start chatgamelab.service
```

## Quick Start for Designers

To work on the React frontend:

```bash
./schemaf.sh dev all
```

Then open **http://localhost:7100**
