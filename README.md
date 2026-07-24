# Dockzilla

A self-hosted PaaS — think [Dokploy](https://dokploy.com), [Coolify](https://coolify.io), or [Zane-ops](https://zaneops.dev/), with the developer experience of Vercel/Heroku.

Point it at a git repo or a Docker image, and your app goes live on a domain with HTTPS — no SSH, no hand-written `docker run`. See [`ONBOARDING.md`](https://github.com/dockzaurus/internal-shared-docs/blob/master/docs/ONBOARDING.MD) for the full pitch, the concepts involved, and the v1 scope.

## Stack

- **Backend** (`packages/backend`) — Go, config via [`goloader`](https://github.com/zixyos/goloader) (TOML + env), logging via [`glog`](https://github.com/zixyos/glog). Postgres and Redis storage adapters.
- **Frontend** (`packages/frontend`) — [TanStack Start](https://tanstack.com/start) (React), Vite, Tailwind CSS, Biome, Storybook.

## Prerequisites

- [Go](https://go.dev) 1.25+
- [Node.js](https://nodejs.org) 22+ and [pnpm](https://pnpm.io)
- [Task](https://taskfile.dev) (`brew install go-task`)

## Getting started

Install frontend dependencies once:

```bash
cd packages/frontend && pnpm install
```

Then, from the repo root:

```bash
task dev
```

This runs the backend (`APP_ENV=local`, reading `packages/backend/cmd/config.local.toml`) and the frontend dev server concurrently.

## Tasks

| Task | Description |
|---|---|
| `task dev` | Run backend + frontend dev servers concurrently |
| `task build` | Build backend + frontend |
| `task backend:dev` | Run only the backend (`go run ./cmd`) |
| `task backend:build` | Build the backend binary to `bin/backend` |
| `task frontend:dev` | Run only the frontend dev server |
| `task frontend:build` | Build the frontend for production |

Run `task --list` to see all available tasks.

## Project structure

```
packages/
  backend/    Go control-plane API
    cmd/      Entrypoint + config (config.<env>.toml)
    internal/ Infra adapters (postgres, redis, http transport)
    pkg/      Shared domain types
  frontend/   TanStack Start dashboard
```
