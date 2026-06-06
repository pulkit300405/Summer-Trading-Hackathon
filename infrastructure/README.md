# Kubernetes Infrastructure

Production-ready Kubernetes manifests for IICPC Hackathon 2026.

## Quick Deploy

```bash
kubectl apply -k k8s/
```

## Services

- **Namespace**: iicpc
- **PostgreSQL**: postgres:5432 (TimescaleDB, 1 replica)
- **Submission Handler**: :8080 (2 replicas)
- **Bot Fleet**: :8081 (5000 concurrent bots)
- **Telemetry Ingester**: :8082 (2 replicas)
- **Leaderboard**: :3000 (2 replicas, LoadBalancer)

## Scaling

```bash
kubectl scale deployment submission-handler -n iicpc --replicas=5
kubectl scale deployment telemetry-ingester -n iicpc --replicas=10
```

## Access

```bash
kubectl port-forward -n iicpc svc/leaderboard 3000:3000
```

## Production Ready

- Health checks configured
- Resource limits set
- Persistent volumes for PostgreSQL
- Horizontal scaling support
