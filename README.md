# Dockzilla

[![Backend CI](https://github.com/dockzaurus/dockzilla/actions/workflows/backend-ci.yml/badge.svg?branch=master)](https://github.com/dockzaurus/dockzilla/actions/workflows/backend-ci.yml)

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
| `task backend:lint` | Lint the backend (golangci-lint, Uber style rules) |
| `task backend:lint:fix` | Lint and apply the auto-fixable subset |
| `task backend:test` | Run the backend tests with the race detector |
| `task backend:check` | Everything CI runs: build + vet + lint + test |
| `task frontend:dev` | Run only the frontend dev server |
| `task frontend:build` | Build the frontend for production |

Run `task --list` to see all available tasks.

## Continuous integration

Every PR touching `packages/backend/**` runs [`.github/workflows/backend-ci.yml`](.github/workflows/backend-ci.yml),
which is `task backend:check` split into three parallel jobs — **Build** (`go build`, `go vet`,
`go mod tidy -diff`), **Lint** (golangci-lint) and **Test** (`go test -race`). Run
`task backend:check` locally to get the same answer before pushing. The frontend workflow comes later.

## Code style

The Go backend follows the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).
See [`GO_STYLE.md`](GO_STYLE.md) for the house rules explained with examples from this codebase —
start there if you're new to Go. The rules are enforced by `packages/backend/.golangci.yml`; run
`task backend:check` before opening a PR.

## Project structure

```
packages/
  backend/    Go control-plane API
    cmd/      Entrypoint + config (config.<env>.toml)
    internal/ Infra adapters (postgres, redis, http transport)
    pkg/      Shared domain types
  frontend/   TanStack Start dashboard
```
