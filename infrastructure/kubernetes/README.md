# Kubernetes manifests

Deployable manifests for the multi-tenant e-commerce platform, organized for `kubectl apply -k .` (kustomize).

## Layout

| File | Purpose |
|---|---|
| `00-namespace.yaml` | `ecommerce` namespace |
| `01-secrets.yaml` | Secret templates (replace `REPLACE_ME` before applying) |
| `02-shared-config.yaml` | ConfigMap consumed by every service (Kafka, Redis, Postgres hosts) |
| `10-` … `25-` | One file per backend service: Deployment + Service + HPA |
| `30-frontend.yaml` | Next.js storefront + admin |
| `40-ingress.yaml` | NGINX ingress fan-out for `saajan.com` and `api.saajan.com` |
| `kustomization.yaml` | Kustomize entry point |

## Prerequisites

This deploys application workloads only. The cluster must already provide:

- Postgres reachable as `postgres:5432`
- MongoDB reachable as `mongodb:27017`
- Redis reachable as `redis:6379`
- Elasticsearch reachable as `elasticsearch:9200`
- Kafka reachable as `kafka:9092`
- NGINX ingress controller and cert-manager (for `40-ingress.yaml`)

Use the cloud provider's managed services (RDS, ElastiCache, MSK, MongoDB Atlas) and write StatefulSets/Helm charts for them — those are intentionally not in this directory.

## Secrets

Every reference in `01-secrets.yaml` is a placeholder. **Do not commit real values.** Populate them via one of:

```bash
# Option A: kubectl create secret (out-of-band)
kubectl -n ecommerce create secret generic postgres-credentials \
  --from-literal=username=postgres \
  --from-literal=password='<real-password>'

# Option B: sealed-secrets or external-secrets-operator (recommended)
```

The 32-byte 2FA encryption key can be generated with:

```bash
openssl rand -hex 32
```

## Deploy

```bash
# Validate
kubectl kustomize infrastructure/kubernetes | kubectl apply --dry-run=client -f -

# Apply
kubectl apply -k infrastructure/kubernetes
```

## Image references

All Deployments reference `ghcr.io/saajan/<service>:latest`. Update the image
registry/tag via kustomize overlays (`overlays/staging/`, `overlays/production/`)
so the base manifests stay environment-agnostic.

## Bangladesh-first defaults

`02-shared-config.yaml` ships BD defaults (BDT currency, Asia/Dhaka timezone,
`bn` language, BD country). Override per-tenant via the tenant-service config
endpoints; override per-environment via kustomize patches.

## Health checks

Every service exposes `/health`; the frontend's liveness probe hits `/`.
Probes use a 30 s initial delay to absorb cold-start time.

## Autoscaling

Each service has an HPA scaling on CPU at 70%. Tune `minReplicas`/`maxReplicas`
per workload — order-service and frontend are pre-set higher because they're
on the request path.
