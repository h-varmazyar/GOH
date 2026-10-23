# Gatec Deployment

Run Compose from this directory:

```bash
docker compose build
docker compose up -d
```

The build context is the repository root. Build TDLib once as a reusable base
image, then build the application or worker image from `deploy/Dockerfile`:

```bash
docker build -f deploy/tdlib.Dockerfile -t gatec-tdlib:local .
docker build --build-arg TDLIB_IMAGE=gatec-tdlib:local -f deploy/Dockerfile -t gatec:local .
docker build --build-arg TDLIB_IMAGE=gatec-tdlib:local -f deploy/Dockerfile -t gate-worker:local .
```

The application and worker builds reuse the prebuilt TDLib libraries and do not
run the TDLib source build again.

Compose runs the airdrop Telegram consumer in its own `airdrop-catcher`
container. It loads Telegram accounts and active subscription filters from
Postgres, then processes matching messages independently of the API process.
The container shares the `tdlib-data` volume with the API so both use the same
stored Telegram sessions. Keep `SERVICE_TELEGRAM_DATABASE_DIR` set to
`/app/.tdlib2` in the environment file. Start it with:

```bash
docker compose build airdrop-catcher
docker compose up -d airdrop-catcher
docker compose logs -f airdrop-catcher
```

The Docker daemon must be able to pull the two base images:

- `golang:1.26-bookworm`
- `debian:bookworm-slim`

If the configured registry mirror cannot serve them, set reachable images in
the root `.env` before building:

```env
GO_IMAGE=<reachable-registry>/golang:1.26-bookworm
RUNTIME_IMAGE=<reachable-registry>/debian:bookworm-slim
```

When those values are stored in the repository root `.env`, build with:

```bash
docker compose --env-file ../.env build
```

The current daemon mirror reports `EOF` while resolving both official images;
that is a registry connectivity problem, not a Go source or Docker copy-path
error.
