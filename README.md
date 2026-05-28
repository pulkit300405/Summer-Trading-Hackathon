# IICPC Summer Hackathon 2026: Distributed Benchmarking Platform

A high-performance distributed platform for benchmarking contestant-submitted trading infrastructure under extreme load. Tests order matching engines with 5K+ concurrent bots, measures latency (p50/p90/p99), throughput, and validates correctness.

## Quick Start

```bash
git clone https://github.com/pulkit300405/Summer-Trading-Hackathon
cd Summer-Trading-Hackathon

docker-compose up

# Services ready at:
# - Submission: http://localhost:8080
# - Leaderboard: http://localhost:3000
# - Metrics: PostgreSQL on :5432
```

## Problem

Trading hackathons need fair, automated benchmarking. Contestants submit orderbook engines in any language (C++, Rust, Go). Manual testing is slow and biased. We need:
- Accept submissions safely (containerize in isolation)
- Generate realistic load (5K+ concurrent bots)
- Measure latency accurately (p50/p90/p99)
- Validate correctness (FIFO, no double-fills)
- Show results in real-time leaderboard

## Solution

Microservices architecture with 4 independent services:
- **Submission Handler** — accepts code, containerizes, manages lifecycle
- **Bot Fleet** — Go goroutines generating 5K+ concurrent connections with realistic order patterns
- **Telemetry Ingester** — low-latency metrics collection, correctness validation, p-percentile calculation
- **Leaderboard UI** — real-time React dashboard with WebSocket updates
- **PostgreSQL + TimescaleDB** — time-series metrics store

## System Architecture

```
Contestant Code (C++/Rust/Go)
    ↓
Submission Handler (containerize + isolate)
    ↓
┌───────────────────────────────────────────┐
│ Bot Fleet (5K concurrent bots)            │
│ Telemetry Ingester (validate correctness) │
│ Leaderboard UI (real-time metrics)        │
└───────────────────────────────────────────┘
    ↓
PostgreSQL + TimescaleDB (time-series)
```

## Features

**Load Generation**
- 5K+ concurrent bots using Go goroutines (minimal memory footprint)
- Realistic order patterns (limit orders, cancellations, partial fills)
- Multiple protocol support (FIX, REST, WebSocket)
- Configurable order rate and market dynamics

**Benchmarking Metrics**
- Latency: p50, p90, p99 (milliseconds)
- Throughput: Orders/fills per second
- Correctness: FIFO validation, fill accuracy, no double-fills
- Per-submission metrics tracking

**Isolation & Security**
- Each submission runs in isolated Docker container
- CPU/memory limits enforced per submission
- Network isolation (unique port per submission)
- No inter-submission data access

**Real-time Monitoring**
- Live leaderboard (WebSocket updates every 500ms)
- Metrics dashboard (latency curves, throughput graphs)
- Per-round metrics visualization

## Tech Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Load Generator | Go + Goroutines | 5K+ concurrent connections, minimal memory overhead |
| Submission Handler | Go + HTTP | Fast, single binary, easy containerization |
| Telemetry | Go + PostgreSQL | Low-latency metrics, time-series queries |
| Frontend | React + WebSocket | Real-time leaderboard, modern UI |
| Storage | PostgreSQL + TimescaleDB | Time-series optimized, easy scaling |
| Orchestration | Docker Compose (dev) + Kubernetes (prod) | Simple local dev, production-ready |

## Project Structure

```
Summer-Trading-Hackathon/
├── README.md                          # This file
├── ARCHITECTURE.md                    # Detailed system design (for judges)
├── SETUP.md                           # Development setup guide
├── docker-compose.yml                 # Local development environment
├── docker-compose.prod.yml            # Production configuration
│
├── submission-handler/                # Service: Accept & containerize submissions
│   ├── main.go
│   ├── handlers/
│   │   ├── upload.go
│   │   └── status.go
│   ├── sandbox/
│   │   └── containerizer.go
│   ├── Dockerfile
│   └── go.mod
│
├── bot-fleet/                         # Service: Generate massive load
│   ├── main.go
│   ├── bot/
│   │   ├── ordergen.go                # Generate realistic orders
│   │   ├── sender.go                  # Send orders (FIX/REST/WebSocket)
│   │   └── states.go                  # Bot state machine
│   ├── Dockerfile
│   └── go.mod
│
├── telemetry-ingester/                # Service: Collect & validate metrics
│   ├── main.go
│   ├── metrics/
│   │   ├── collector.go               # Low-latency metrics collection
│   │   ├── validator.go               # Correctness validation (FIFO, fills)
│   │   └── aggregator.go              # p50/p90/p99 calculation
│   ├── Dockerfile
│   └── go.mod
│
├── leaderboard/                       # Service: Real-time UI
│   ├── src/
│   │   ├── App.jsx
│   │   ├── components/
│   │   │   ├── ScoreBoard.jsx
│   │   │   └── MetricsGraph.jsx
│   │   └── hooks/
│   │       └── useWebSocket.js
│   ├── package.json
│   ├── Dockerfile
│   └── .dockerignore
│
├── infrastructure/                    # Deployment & IaC
│   ├── kubernetes/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── pvc.yaml
│   └── terraform/
│       ├── main.tf
│       ├── variables.tf
│       └── outputs.tf
│
└── docs/
    ├── DECISIONS.md                   # Tech stack rationale
    ├── API_SPEC.md                    # API endpoints (submission, bots, metrics)
    └── TROUBLESHOOTING.md
```

## Development Roadmap

**Week 1: Core Components**
- Submission handler accepts uploads & creates containers
- Bot fleet spawns 100 concurrent bots
- Telemetry ingester captures latency data
- Basic metrics storage (PostgreSQL)

**Week 2: Scale & Correctness**
- Scale bot fleet to 5K concurrent bots
- Implement correctness validation (FIFO, fill accuracy)
- Calculate p50/p90/p99 latencies
- Leaderboard UI (basic version)

**Week 3: Polish & IaC**
- Optimize latency (connection pooling, batching)
- Write Kubernetes manifests
- Comprehensive error handling
- Production docker-compose

**Week 4: Testing & Documentation**
- Stress test the platform itself
- Write ARCHITECTURE.md (judges read this)
- Demo script
- Final cleanup

## Setup (Local Development)

**Prerequisites**
- Docker & Docker Compose
- Go 1.21+ (for local development)
- Node.js 18+ (for leaderboard)
- PostgreSQL client tools (optional, for debugging)

**Installation**

```bash
# 1. Clone
git clone https://github.com/pulkit300405/Summer-Trading-Hackathon
cd Summer-Trading-Hackathon

# 2. Start all services
docker-compose up

# 3. Verify services are running
curl http://localhost:8080/health          # Submission handler
curl http://localhost:8081/health          # Bot fleet
curl http://localhost:8082/health          # Telemetry ingester
open http://localhost:3000                 # Leaderboard

# 4. Upload a test submission
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"language":"go", "code":"..."}'
```

## Key Metrics (What Judges Evaluate)

1. **Architecture Quality** (40%)
   - Clean microservices design
   - Clear separation of concerns
   - Resilience & error handling

2. **Correctness** (30%)
   - Valid order matching (FIFO, price-time priority)
   - Accurate latency measurements
   - No double-fills or lost orders

3. **Scale** (20%)
   - Handles 5K+ concurrent bots
   - Measures throughput accurately
   - Real-time leaderboard performance

4. **Documentation & IaC** (10%)
   - Clear ARCHITECTURE.md
   - Kubernetes manifests or Terraform
   - Easy to understand design decisions

## Security Notes

- Submissions run in isolated Docker containers
- CPU/memory limits enforced per submission
- Network isolation (each submission on unique port)
- No network access from submission containers (by default)

## Documentation

- **[ARCHITECTURE.md](./ARCHITECTURE.md)** — Detailed system design (start here if new)
- **[SETUP.md](./SETUP.md)** — Step-by-step development setup
- **[docs/DECISIONS.md](./docs/DECISIONS.md)** — Why we chose Go, Docker, PostgreSQL, etc.
- **[docs/API_SPEC.md](./docs/API_SPEC.md)** — API endpoints and contracts
- **[docs/TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md)** — Common issues and solutions

## Submission Checklist (June 9th)

- [ ] GitHub repo public
- [ ] `docker-compose up` works (all 4 services start)
- [ ] Can upload submission, spin up container, run bots
- [ ] Leaderboard shows real-time metrics
- [ ] ARCHITECTURE.md explains design decisions
- [ ] Kubernetes manifests or Terraform configs included
- [ ] README has clear setup instructions

## Learning Resources

- Go Concurrency: https://go.dev/blog/pipelines
- TimescaleDB: https://docs.timescale.com/
- Docker Best Practices: https://docs.docker.com/develop/dev-best-practices/
- Kubernetes: https://kubernetes.io/docs/concepts/overview/

## Author

**Pulkit Singh** — Full-stack architecture, bot fleet (goroutine concurrency), telemetry validation, Kubernetes setup, system testing.

GitHub: [@pulkit300405](https://github.com/pulkit300405)

## Timeline

Hackathon: May 9 - June 10, 2026  
Submission Deadline: June 9, 2026  
Status: In Development
