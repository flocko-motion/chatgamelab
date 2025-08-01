[![CI to GHCR](https://github.com/flocko-motion/chatgamelab/actions/workflows/docker-image.yml/badge.svg)](https://github.com/flocko-motion/chatgamelab/actions/workflows/docker-image.yml)

# chatgamelab

Educational GPT-Chat based text adventure lab. 

Create your own text adventure games and play them with your friends.

- Learn, how GPT can be used to create interactive stories.
- Use debug-mode to see the raw requests and responses of the GPT model 

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

## Installation

The project is deployed as a docker. Start by pulling the image:

```bash
# make sure to login to ghcr.io first..
docker pull ghcr.io/flocko-motion/chatgamelab:latest
```

The docker requires a few
runtime parameters to be set:

```bash 
DATA_PATH=/path/to/data/dir
AUTH0_DOMAIN=your.auth0.domain
AUTH0_AUDIENCE=your.auth0.audience
PUBLIC_URL=your.public.url

docker run -p 3000:3000
-v ${DATA_PATH}:/app/var
-e AUTH0_DOMAIN=${AUTH0_DOMAIN}
-e AUTH0_AUDIENCE=${AUTH0_AUDIENCE}
-e CORS_ALLOWED_ORIGIN=${PUBLIC_URL}
chatgamelab
```