# Docker And Security

## Local Stack

The repository includes:

- `arpego/Dockerfile` for the Go API
- `libretto/Dockerfile` for the Vite frontend
- `docker-compose.yaml` with Traefik and CrowdSec in front of both services

Hostnames default to:

- `app.localhost`
- `api.localhost`
- `traefik.localhost`

Copy the local environment template and create the CrowdSec secret file:

```bash
cp .env.example .env
mkdir -p infra/secrets
cp infra/secrets/crowdsec_lapi_key.example.txt infra/secrets/crowdsec_lapi_key.txt
```

Use the same random value for:

- `CROWDSEC_LAPI_KEY` in `.env`
- `infra/secrets/crowdsec_lapi_key.txt`

Bring the stack up:

```bash
docker-compose -f docker-compose.yaml up --build
```

## Security Scanning

Local commands:

```bash
make scan-security
make scan-secrets
make scan-config
make scan-images
```

CI:

- `.github/workflows/security.yml` runs Trivy against the repository, IaC/config, and built images.
- SARIF results are uploaded to GitHub code scanning.
