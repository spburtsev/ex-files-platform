# ex-files-platform

## Deploy to a VPS (Docker + Caddy)

### Prerequisites

- Docker + Docker Compose on the VPS.
- A Caddy container already running and attached to an **external** Docker network named `web`.
  Create it once if needed: `docker network create web`
- DNS: an `A` record for `exfiles.your-domain.xyz`

### 1. Caddy entry

Ensure your Caddyfile has this block, then reload Caddy. Caddy reaches the frontend
container directly over the shared `web` network, so no host ports are published.

```caddy
exfiles.your-domain.xyz {
    reverse_proxy ex-files-frontend:3003
}
```

### 2. Configure secrets

```sh
git clone <repo-url> ex-files-platform && cd ex-files-platform
cp .env.prod.example .env.prod
# Edit .env.prod
```

### 3. Launch

```sh
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

On first start the backend auto-migrates the database and seeds the root user from
`SEED_ROOT_EMAIL` / `SEED_ROOT_PASSWORD`

### 4. Verify

```sh
docker compose -f docker-compose.prod.yml ps
```

**Grafana** (logs + traces) is localhost-only; reach it via SSH tunnel:
  `ssh -L 3200:127.0.0.1:3200 <vps>` <http://localhost:3200>

## Common operations

```sh
# Redeploy after pulling changes
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build

# Tail logs
docker compose -f docker-compose.prod.yml logs -f ex-files-backend

# Stop (add -v to also delete data volumes)
docker compose -f docker-compose.prod.yml down
```
