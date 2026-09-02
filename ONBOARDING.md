# Dockzilla — What We're Building

Welcome. This doc exists so you don't have to guess what you signed up for. Read it once fully, then keep it around as a reference — we'll update it as the project evolves.

If any assumption here is wrong (stack, scope, priorities), say so. This is a living doc, not a spec carved in stone.

---

## 1. The elevator pitch

We're building a **self-hosted PaaS** — think **Dokploy**, **Zane-ops**, or **Coolify**, with the developer experience of **Vercel/Heroku**.

Concretely: you `git push`, or point it at a Docker image, and a few seconds later your app is live on a domain, with HTTPS, without you ever SSHing into a server or writing a `docker run` command by hand.

The difference between us and Vercel: Vercel is a company running that platform *for you*, on *their* servers, and billing you for it. We're building the software that lets **anyone run that same platform on their own server** (a $5 VPS, a home server, whatever). That's what Dokploy/Coolify/Zane-ops already do — we're building our own version, which means we need to understand every layer they abstract away.

---

## 2. Concepts you need before anything else

Don't skip this section even if some of it feels obvious — we'll use this vocabulary constantly.

### Client / server, and what a "host" actually is

Every website works the same way at the core: your browser (**client**) sends a request over the network to a machine somewhere (**server**), which sends back a response. That machine has to be turned on, connected to the internet, and running a program that's listening for requests. That machine is called a **host** — it "hosts" the application. "Hosting" a website just means: keeping a machine alive, reachable, and running your code 24/7.

A host is identified on the network by an **IP address** (e.g. `142.250.75.14`). Nobody types IPs though — that's what **DNS** (Domain Name System) is for: it's the phonebook that maps a domain name (`dockzilla.dev`) to an IP address, via a **DNS record** (an "A record" points a domain straight at an IPv4 address).

### IaaS vs PaaS vs SaaS

This is the axis of "how much do you manage yourself":

| Layer | You manage | Example |
|---|---|---|
| **IaaS** (Infrastructure as a Service) | OS, runtime, everything above raw hardware | AWS EC2, a bare DigitalOcean droplet |
| **PaaS** (Platform as a Service) | Just your code | Heroku, Vercel, Render, Railway, **Dockzilla** |
| **SaaS** (Software as a Service) | Nothing — you just use it | Notion, Gmail |

A PaaS sits in the middle: it takes a raw server (IaaS) and wraps it with automation so the developer only ever thinks about their code, never about the machine. **That automation layer is the entire product we're building.**

### Ports, and why one server can run many apps

A host doesn't run "a website" — it runs *processes*, and each process that wants to talk over the network binds to a **port** (a number from 1 to 65535) on that host. `example.com:443` means "host at this IP, port 443." This is why a single server can run dozens of apps simultaneously: each one just listens on a different port internally.

### Reverse proxy

The public internet only expects two ports to matter: 80 (HTTP) and 443 (HTTPS). But you might have 20 apps running on 20 different internal ports on the same host. The piece that sits in front of everything, looks at the incoming request (specifically the `Host` header — literally the domain name the browser asked for), and forwards it to the *correct* internal app/port is called a **reverse proxy**. Nginx, Caddy, and Traefik are the well-known ones. **This is one of the core pieces we'll build or wire up.** Without it, a PaaS can't route `app-a.dockzilla.dev` and `app-b.dockzilla.dev` to two different containers on the same box.

### TLS/SSL and Let's Encrypt

HTTPS = HTTP encrypted with TLS, which requires a **certificate**. Getting one manually is a pain, which is why **Let's Encrypt** exists: a free, automated certificate authority. The protocol it speaks is called **ACME**. "Automatic HTTPS for every app you deploy, zero config" is a signature PaaS feature — under the hood it's just the reverse proxy talking ACME to Let's Encrypt on your behalf.

### Containers (Docker)

Instead of installing your app's dependencies directly on the host (and fighting version conflicts between App A needing Node 16 and App B needing Node 20), you package the app into a **container**: a lightweight, isolated bundle with its own filesystem and dependencies, built from an **image**. Docker is the tool that builds images (from a `Dockerfile`) and runs containers from them.

Important nuance for later: a container is *not* a lightweight VM. It's a regular Linux process, isolated using two kernel features — **namespaces** (it can't see other processes/network/filesystem outside itself) and **cgroups** (its CPU/memory usage is capped). Docker didn't invent isolation, it just made it easy to use. Understanding this will matter a lot once we start talking to containers programmatically instead of via the `docker` CLI.

### Orchestration

Once you have many containers (maybe across many hosts), something needs to decide *where* each one runs, restart it if it crashes, and update the reverse proxy when it moves. That's **orchestration**. Kubernetes is the industrial-strength version; Docker Compose is the toy single-host version. Dokploy/Coolify/Zane-ops (and us, at first) sit closer to Compose: single-host, simple scheduling, no need to reinvent Kubernetes.

---

## 3. How a PaaS actually works, end to end

This is the pipeline that happens every time someone deploys, stitched from the concepts above:

1. **Trigger** — user pushes to a git branch (webhook fires) or hits "deploy" in a dashboard / CLI.
2. **Fetch** — the platform clones the repo (or pulls the specified Docker image) on the host that will run the build.
3. **Build** — either a `Dockerfile` in the repo is built, or the platform auto-detects the language/framework and builds a sane default image (this is what "buildpacks" do). Output: a Docker image, tagged and pushed to a registry.
4. **Run** — a container is started from that image, given env vars/secrets, and attached to an internal network.
5. **Route** — the reverse proxy config is updated so the app's domain now points at that new container. Ideally with **zero downtime**: the old container keeps serving traffic until the new one passes a health check, then traffic switches over (this pattern is called blue-green or rolling deploy).
6. **Expose** — DNS + reverse proxy + TLS certificate = the app is now live on `https://yourapp.com`.
7. **Observe** — logs and basic metrics (CPU/memory) get captured and shown back to the user, because "it's running, trust me" isn't good enough.

Every box in this list is a subsystem we'll build. None of it is magic — Vercel/Heroku/Dokploy just hide the wiring.

---

## 4. The engineering fields this project will touch

This is the part that matters most for you as a junior wanting to grind: this project is basically a tour through most of backend/infra engineering. Here's the map.

| Field | Why we need it here | What you'll actually learn |
|---|---|---|
| **Linux/systems fundamentals** | Containers *are* Linux features (namespaces, cgroups); you can't fake understanding this | Processes, file descriptors, signals, namespaces, cgroups |
| **Containers (Docker)** | The unit of deployment | Images, layers, Dockerfile builds, talking to the Docker daemon programmatically (not just the CLI) |
| **Networking** | Nothing is reachable without it | TCP/IP basics, HTTP, reverse proxying, load balancing, DNS |
| **TLS/PKI** | "It just has HTTPS" is a feature people expect for free | Certificates, ACME/Let's Encrypt automation |
| **Backend engineering (Go)** | This is the language of the whole control plane | Concurrency (goroutines per deployment/watcher), REST/gRPC API design, job queues, error handling at scale |
| **Git internals** | Deploys are triggered by git activity | Webhooks, cloning/checking out server-side, detecting commits/branches |
| **CI/CD concepts** | "Build → deploy" is a pipeline, and pipelines fail in interesting ways | Build isolation, artifact/image registries, rollout strategies (rolling/blue-green), rollback |
| **Databases** | Someone has to remember which app is which | Modeling apps, deployments, domains, env vars, users in a relational DB |
| **Security** | We're literally running strangers'/our own arbitrary code on a shared host | Secrets management, per-app isolation, auth (sessions/tokens), least-privilege |
| **Frontend/dashboard** | Nobody wants to manage this via raw API calls | Building a UI over the control-plane API: app list, deploy history, logs viewer |
| **Observability** | "Is it actually working" needs an answer | Log streaming/aggregation, basic metrics, health checks |
| **Distributed systems** *(later, if we go multi-host)* | Single host doesn't scale forever | Service discovery, scheduling, consensus — this is the deep end, we're not starting here |

You don't need to know all of this on day one. Nobody does. The point of the table is so you can see the shape of the whole thing and recognize, as we build, which piece of the map you're currently standing on.

---

## 5. What the MVP (v1) actually needs to do

Concrete and small, on purpose:

- [ ] Take a git repo URL + a `Dockerfile` (skip auto-buildpacks for v1 — that's its own project).
- [ ] Build the image on the host.
- [ ] Run it as a container.
- [ ] Reverse-proxy a subdomain to it (e.g. `<app>.dockzilla.dev`).
- [ ] Auto-provision a TLS cert for it.
- [ ] Redeploy on new commit (webhook), with zero-downtime swap.
- [ ] Store app/deployment state in a database, not in memory.
- [ ] A minimal dashboard: list apps, trigger deploy, view logs.

Everything else (multi-host, auto-detected buildpacks, teams/billing, previews per PR like Vercel does) is v2+ and deliberately out of scope until v1 works end to end.

---

## 6. Suggested order to actually learn/build this

If you want to ramp up productively rather than reading docs for two weeks first:

1. **Use Docker manually first.** Build an image, run it, expose a port, use Compose to run two containers and have one talk to the other. Don't write a line of our code until you've done this by hand — you need the "obvious" version before you build the automated one.
2. **Talk to Docker from Go**, not the CLI. Use the Docker Engine SDK to start/stop/inspect containers programmatically. This is the first real building block of the platform.
3. **Build a minimal reverse proxy.** Either wire up Caddy/Traefik and drive their config via API, or write a tiny HTTP proxy in Go that forwards based on `Host` header. Either is a legitimate learning path — we'll decide which one we actually ship with.
4. **Wire up a webhook receiver** that clones a repo and runs `docker build` + `docker run` on push. Congratulations, that's a (very rough) PaaS.
5. **Put a database behind it** so state survives a restart.
6. **Add TLS automation.**
7. **Add a dashboard.**
8. **Add logs/metrics.**

Each step is independently useful and demoable — good for keeping motivation up and for showing concrete progress.

---

## 7. Questions worth asking early (and not guessing on)

- Final language/framework for the dashboard frontend — not decided in this doc.
- Single-host only for v1, or design for multi-host from day one? (Strong recommendation: single-host first. Multi-host is a different, much harder problem — don't pay that complexity tax before v1 exists.)
- Which reverse proxy strategy: drive an existing one (Caddy/Traefik) vs write our own? Writing our own teaches more; using an existing one ships faster.

Bring these back to the team before locking them in — they shape a lot of what follows.
