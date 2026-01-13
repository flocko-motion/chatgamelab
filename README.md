[![CI to GHCR](https://github.com/flocko-motion/chatgamelab/actions/workflows/docker-image.yml/badge.svg)](https://github.com/flocko-motion/chatgamelab/actions/workflows/docker-image.yml)

# chatgamelab

Educational GPT-Chat based text adventure lab.

Create your own text adventure games and play them with your friends.

- Learn, how GPT can be used to create interactive stories.
- Use debug-mode to see the raw requests and responses of the GPT model

## Rules for working on this project

If you're working on ChatGameLab (including web designers), you **must learn Git basics first**. There are user-friendly Git clients for Mac like [GitHub Desktop](https://desktop.github.com/) or [Sourcetree](https://www.sourcetreeapp.com/) that make this easier.

**Essential Rules:**

1. **`main` branch** = Published/live website (don't touch!)
2. **`development` branch** = Current development work
3. **Your workflow**: Fork `development` → make changes → create Pull Request to `development`→ wait for review
4. **Never** change `main` directly or make PRs to `main` alone
5. **Work in small chunks** - when you finish a feature, make a PR right away. Don't work alone for weeks!

This keeps everyone in sync and prevents the project from breaking apart.


## Requirements

- **Docker** and **docker compose** (v2+)
- **Node.js and npm** (for local frontend development)
- **Go 1.21+** (for local backend development)
- **Auth0 account** (for authentication)
- **OpenAI API key** (to play games)

## Setup

### 1. Configure environment

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` with your values. See `.env.example` for all available options organized by service (Database, Backend, Frontend).

## Development

Development uses Docker for services you're not actively working on, while you run your code locally with full debugger support.

### Development Modes

| Command | Docker runs | You develop locally |
|---------|-------------|---------------------|
| `./run-dev.sh frontend` | db + backend | Frontend (`cd web && npm run dev`) |
| `./run-dev.sh backend` | db + web | Backend (`cd server && go run . server`) |
| `./run-dev.sh all` | db only | Both frontend and backend |
| `./run-dev.sh` | all services | None (everything in Docker) |

### Develop Frontend

```bash
# Terminal 1 - Start database only
./run-dev.sh all

# Terminal 2 - Start frontend locally
cd web && npm run dev

# Terminal 3 - Start backend locally
cd server && go run . server
```

Open **http://localhost:5173** in your browser.

### Develop Backend

```bash
# Terminal 1 - Start database and frontend in Docker
./run-dev.sh backend

# Terminal 2 - Start backend locally (with debugger)
cd server && go run . server
```

Open **http://localhost** in your browser (served by Docker).

### Start All Services in Docker

```bash
# Start everything (db + backend + web) in Docker
./run-dev.sh
```

Open **http://localhost** in your browser.

### Options

```bash
./run-dev.sh frontend --reset-db          # Reset database before starting
./run-dev.sh all --reset-db               # Reset database before starting
./run-dev.sh --reset-db                   # Reset database before starting all services
./run-dev.sh frontend --port-backend 8080 # Custom backend port
./run-dev.sh --help                       # Show all options
```

### Dev Mode Features

When `DEV_MODE=true` in `.env`, additional development features are enabled:

**JWT Token Generation** - Generate JWT tokens for any user without Auth0:
```bash
cd server
go run . user jwt                    # Generate token for dev user
go run . user jwt <user-uuid>        # Generate token for specific user
```

**Dev User** - A default dev user is seeded on startup with UUID `00000000-0000-0000-0000-000000000000`.

### Reset Database

```bash
./reset-dev-db.sh
```

Then restart with `./run-dev.sh`.

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

**Production service:**
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
Environment="DB_PASSWORD=secure_password"
Environment="AUTH0_DOMAIN=your.auth0.domain"
Environment="AUTH0_AUDIENCE=your.auth0.audience"
Environment="AUTH0_CLIENT_ID=your_client_id"
Environment="API_BASE_URL=https://yourdomain.com"
Environment="PUBLIC_URL=https://yourdomain.com"
Environment="PORT_EXPOSED=80"

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

If you're a designer wanting to explore the React frontend without the full backend:

```bash
cd web && npm run dev
```

Then open **http://localhost:5173**
