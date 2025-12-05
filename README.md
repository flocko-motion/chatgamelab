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

## Quick Start for Designers (Mock Mode)

If you're a designer wanting to explore the React frontend without setting up the full backend, you can run the app in mock mode:

**Prerequisites**: Make sure you have [Node.js and npm](https://nodejs.org/) installed on your Mac.

1. **Launch the frontend** (on Mac):
   ```bash
   ./run-frontend.sh
   ```

2. **Open your browser** and go to:
   ```
   http://localhost:3000?mock=true
   ```

3. **Explore the app**:
    - The React app will be running at `http://localhost:3000`
    - Mock mode provides fake data so you can test all features
    - No Auth0 or backend setup required
    - Login/logout buttons work with fake authentication

Perfect for UI/UX design work and frontend development!

## Quick Start for Designers (Mock Mode)

If you're a designer wanting to explore the React frontend without setting up the full backend, you can run the app in mock mode:

**Prerequisites**: Make sure you have [Node.js and npm](https://nodejs.org/) installed on your Mac.

1. **Launch the frontend** (on Mac):
   ```bash
   ./run-frontend.sh
   ```

2. **Open your browser** and go to:
   ```
   http://localhost:3000?mock=true
   ```

3. **Explore the app**:
   - The React app will be running at `http://localhost:3000`
   - Mock mode provides fake data so you can test all features
   - No Auth0 or backend setup required
   - Login/logout buttons work with fake authentication

Perfect for UI/UX design work and frontend development!

## Requirements

To install and run the project you will need the following:
- Auth0 account
- Docker

To use the project you will need the following:
- OpenAI API key

## Running with Docker Compose (recommended)

The repository now contains a multi-container setup using **Docker Compose**:

- `client/` – React SPA, built and served via Node/Express
- `server/` – Go backend API
- `db` – PostgreSQL database

### 1. Prerequisites

- Docker
- docker compose (v2+)
- Auth0 account & OpenAI API key

### 2. Configure environment

For Docker Compose, copy the example file at the **repo root**:

```bash
cp .env.example .env
```

Then adjust the values as needed. A typical setup:

```bash
AUTH0_DOMAIN=your.auth0.domain
AUTH0_AUDIENCE=your.auth0.audience
PUBLIC_URL=http://localhost:3000
```

You may also want to adjust the `DATABASE_URL` in `docker-compose.yml` if you change DB credentials.

### 3. Build and start the stack

From the project root:

```bash
docker compose build
docker compose up
```

This will start three services:

- `db`       → PostgreSQL at `localhost:5432`
- `server`   → Go API at `http://localhost:3001`
- `client`   → React frontend at `http://localhost:3000`

Inside the Docker network, the frontend talks to the API at `http://server:3000` (see `REACT_APP_API_BASE_URL` in `docker-compose.yml`).

To stop the stack:

```bash
docker compose down
```
