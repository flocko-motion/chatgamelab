# Developing ChatGameLab

How to run ChatGameLab on your own machine and work on it. Before your first
pull request, read [CONTRIBUTING.md](../CONTRIBUTING.md): it walks through the
branches, the checks, what review is for, and the commands (`make rebase`,
`make pr`) that drive each step.

Working on ChatGameLab in any role, web design included, takes Git basics.
Friendly clients for Mac are [GitHub Desktop](https://desktop.github.com/) and
[Sourcetree](https://www.sourcetreeapp.com/).

## Requirements

- **Go** (version in `go/go.mod`)
- **Node.js and npm**
- **podman** (rootless) and **docker-compose**, or Docker with `SCHEMAF_RUNTIME=docker`; see [schemaf's container runtime](https://github.com/flocko-motion/schemaf/blob/main/INSTALL.md#container-runtime)
- **An Auth0 account**, for signing in
- **An API key of an AI platform** (OpenAI or Mistral), to play games

## Setup

The platform is a [schemaf](https://github.com/flocko-motion/schemaf) app: the
Go server in `go/` is the single gateway, serving `/api/*` and the frontend in
`frontend/`. The game engine lives in `go/engine/`, its own Go module.

The dev environment's values live outside the repository, in
`~/.chatgamelab/dev/etc/env`. It needs at least the database password:

```bash
mkdir -p ~/.chatgamelab/dev/etc
echo "DB_PASS=changeme" > ~/.chatgamelab/dev/etc/env
```

## Running it

`./schemaf.sh` is the entry point for everything:

```bash
./schemaf.sh codegen               # regenerate glue code after changing migrations, queries or endpoints
./schemaf.sh dev                   # list the services
./schemaf.sh dev db,backend        # Postgres + the Go server
./schemaf.sh dev all               # Postgres + the Go server + the Vite dev server
./schemaf.sh test                  # codegen, then all tests
```

The app runs at **http://localhost:7100**. The ports come from `port` in
`schemaf.toml`: the server on 7100, the Vite dev server on 7102, Postgres on
7103. In dev the server passes every request outside `/api/*` to Vite.

To work on the web app's look, `./schemaf.sh dev all` is all it takes; open
http://localhost:7100.

A development token for any user:

```bash
./schemaf.sh auth token <user-id>
```

`make help` lists the project's other commands: translations, the roadmap, the
homepage preview, releases and more.

## Running an instance

A ChatGameLab instance runs on its own server and installs released versions;
its admin upgrades it from the app. [deploy/README.md](../deploy/README.md)
sets up a server, [workflows/releasing.md](workflows/releasing.md) cuts a
release.
