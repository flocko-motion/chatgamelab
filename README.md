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
| `./run-dev.sh db` | db only | Both frontend and backend |

### Develop Frontend

```bash
# Terminal 1 - Start database and backend in Docker
./run-dev.sh frontend

# Terminal 2 - Start frontend locally (with hot reload)
cd web && npm run dev
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

### Options

```bash
./run-dev.sh frontend --reset-db          # Reset database before starting
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

Production runs everything in Docker containers. **Do NOT use `.env` files in production.** Environment variables must be injected externally via your hosting provider or systemd.

### Required Environment Variables

```bash
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

Create a systemd service with environment variables:

```bash
# Create service file
sudo nano /etc/systemd/system/chatgamelab.service
```

```ini
[Unit]
Description=ChatGameLab
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/path/to/chatgamelab
Environment="DB_PASSWORD=secure_password"
Environment="AUTH0_DOMAIN=your.auth0.domain"
Environment="AUTH0_AUDIENCE=your.auth0.audience"
Environment="API_BASE_URL=https://yourdomain.com"
Environment="AUTH0_CLIENT_ID=your_client_id"
ExecStart=/usr/bin/docker compose up -d --build
ExecStop=/usr/bin/docker compose down

[Install]
WantedBy=multi-user.target
```

Or use `systemctl edit chatgamelab` to add environment overrides

### Useful Commands

```bash
docker compose logs -f      # View logs
docker compose down         # Stop all services
docker compose up -d --build  # Rebuild and restart
```

## Quick Start for Designers

If you're a designer wanting to explore the React frontend without the full backend:

```bash
cd web && npm run dev
```

Then open **http://localhost:5173**
