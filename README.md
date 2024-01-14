# chatgamelab

Educational GPT-Chat based text adventure lab. 

Create your own text adventure games and play them with your friends.

- Learn, how GPT can be used to create interactive stories.
- Use debug-mode to see the raw requests and responses of the GPT model 

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