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

Edit `.env` and set your values:

```bash
# Database password (use something simple for dev, no special characters)
DB_PASSWORD='your_dev_password'

# Auth0 configuration
AUTH0_DOMAIN="your.auth0.domain"
AUTH0_AUDIENCE="your.auth0.audience"
PUBLIC_URL=http://localhost:3000
```

## Development Mode

Development mode runs the database and nginx proxy in Docker, while you run the Go server and React client locally with full debugger support.

### Start development environment

**Terminal 1** - Start database and proxy:
```bash
./run-dev.sh
```

**Terminal 2** - Start Go backend:
```bash
./run-dev-server.sh
```

**Terminal 3** - Start React frontend:
```bash
./run-dev-client.sh
```

Then open **http://localhost** in your browser (or the `PUBLIC_URL` from your `.env`).

### Dev Mode Features

When running the server with the `--dev` flag (as done in `run-dev-server.sh`), additional development features are enabled:

**JWT Token Generation** - Generate JWT tokens for any user without Auth0:
```bash
cd server
go run . user jwt                    # Generate token for dev user
go run . user jwt <user-uuid>        # Generate token for specific user
```

This saves the token to `~/.cgl/jwt` and outputs a URL you can open in your browser to log in automatically. The CLI tool will use this token for subsequent API calls.

**Dev User** - A default dev user is seeded on startup with UUID `00000000-0000-0000-0000-000000000000`.

### Reset the database

To wipe the database and recreate it from `schema.sql`:

```bash
./reset-dev-db.sh
```

Then restart with `./run-dev.sh`.

## Production Mode

Production mode builds and runs everything in Docker containers.

```bash
./run-prod.sh
```

This builds all images and starts the full stack in detached mode.

Useful commands:
```bash
docker compose logs -f    # View logs
docker compose down       # Stop all services
```

## Quick Start for Designers (Mock Mode)

If you're a designer wanting to explore the React frontend without setting up the full backend:

```bash
./run-dev-client.sh
```

Then open **http://localhost:3000?mock=true**

Mock mode provides fake data so you can test all UI features without Auth0 or backend setup.
