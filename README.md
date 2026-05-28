# IICPC Summer Hackathon 2026: Distributed Benchmarking Platform

A high-performance distributed platform for benchmarking contestant-submitted trading infrastructure under extreme load. Tests order matching engines with 5K+ concurrent bots, measures latency (p50/p90/p99), throughput, and correctness.

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

Trading hackathons need fair benchmarking. Contestants submit orderbook engines in any language (C++, Rust, Go). We need to:
- Accept submissions safely (containerize them)
- Generate realistic load (5K+ concurrent bots)
- Measure latency accurately (p50/p90/p99)
- Validate correctness (FIFO, no double-fills)
- Show results in real-time

## Solution

Microservices architecture:
- **Submission Handler** accepts code, containerizes it, runs it in isolation
- **Bot Fleet** (Go + goroutines) generates 5K+ concurrent connections
- **Telemetry Ingester** captures latency, throughput, validates correctness
- **Leaderboard UI** shows real-time metrics
- **PostgreSQL + TimescaleDB** stores time-series metrics

## System Architecture

```
Contestant Code (C++/Rust/Go)
    ↓
Submission Handler (containerize)
    ↓
┌─────────────────────────────┐
│ Bot Fleet (5K bots)         │
│ Telemetry Ingester          │
│ Leaderboard (React)         │
└─────────────────────────────┘
    ↓
PostgreSQL + TimescaleDB
```

## Features

**Load Generation**
- 5K+ concurrent bots using Go goroutines
- Realistic order patterns (limit orders, cancellations, fills)
- Multiple protocol support (FIX, REST, WebSocket)

**Benchmarking**
- Latency: p50, p90, p99 milliseconds
- Throughput: Orders per second
- Correctness: FIFO validation, fill accuracy, no double-fills

**Isolation & Security**
- Each submission runs in isolated Docker container
- CPU/memory limits enforced
- Network isolation by default
- No data leakage between submissions

**Real-time Monitoring**
- Live leaderboard (WebSocket updates)
- Metrics dashboard (latency curves, throughput)
- Per-submission metrics tracking

## Tech Stack

| Component | Tech | Why |
|-----------|------|-----|
| Load Generator | Go + Goroutines | 5K+ concurrent connections, minimal memory |
| Submission Handler | Go + HTTP | Fast containerization, single binary |
| Metrics | PostgreSQL + TimescaleDB | Time-series optimized, easy scaling |
| Frontend | React + WebSocket | Real-time UI, modern stack |
| Orchestration | Docker Compose + Kubernetes | Local dev easy, production-ready |

## Project Structure

```
Summer-Trading-Hackathon/
├── bot-fleet/              # Load generator (5K concurrent bots)
│   ├── main.go
│   ├── bot/
│   │   ├── ordergen.go     # Realistic order generation
│   │   ├── sender.go       # Send to submission container
│   │   └── states.go       # Bot state machine
│   └── Dockerfile
│
├── submission-handler/     # Accept & containerize submissions
│   ├── main.go
│   ├── handlers/
│   │   ├── upload.go
│   │   └── status.go
│   ├── sandbox/
│   │   └── containerizer.go
│   └── Dockerfile
│
├── telemetry-ingester/     # Collect & validate metrics
│   ├── main.go
│   ├── metrics/
│   │   ├── collector.go
│   │   ├── validator.go    # Correctness checks
│   │   └── aggregator.go   # p50/p90/p99
│   └── Dockerfile
│
├── leaderboard/            # Real-time UI
│   ├── src/
│   │   ├── App.jsx
│   │   ├── components/
│   │   │   ├── ScoreBoard.jsx
│   │   │   └── MetricsGraph.jsx
│   │   └── hooks/
│   │       └── useWebSocket.js
│   └── Dockerfile
│
├── docker-compose.yml      # Local dev (all 4 services)
├── docker-compose.prod.yml # Production config
│
├── ARCHITECTURE.md         # Detailed design (for judges)
├── SETUP.md               # Dev setup guide
└── README.md              # This file
```

## Development Roadmap

**Week 1: Core**
- Submission handler accepts uploads & containerizes
- Bot fleet spawns 100 concurrent bots
- Telemetry captures latency data
- Basic PostgreSQL storage

**Week 2: Scale & Validation**
- Scale to 5K concurrent bots
- Correctness validation (FIFO, fills)
- Calculate p50/p90/p99 latencies
- Leaderboard UI (basic)

**Week 3: Polish**
- Optimize latency (connection pooling)
- Kubernetes manifests
- Error handling
- Production docker-compose

**Week 4: Testing & Docs**
- Stress test platform itself
- Detailed ARCHITECTURE.md
- Demo script
- Final cleanup

## Setup (Local Development)

**Prerequisites**: Docker & Docker Compose, Go 1.21+, Node.js 18+

```bash
git clone https://github.com/pulkit300405/Summer-Trading-Hackathon
cd Summer-Trading-Hackathon

docker-compose up

# Verify services
curl http://localhost:8080/health      # Submission
curl http://localhost:8081/health      # Bot fleet
curl http://localhost:8082/health      # Telemetry
open http://localhost:3000             # Leaderboard

# Submit test code
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"language":"go", "code":"..."}'
```

## Key Metrics (Judges Evaluate)

1. **Architecture** (40%) — Clean microservices, separation of concerns, resilience
2. **Correctness** (30%) — Valid order matching, accurate latency, no data loss
3. **Scale** (20%) — Handles 5K+ bots, real-time performance
4. **Documentation & IaC** (10%) — Clear design, Kubernetes/Terraform configs

## Security

- Submissions run in isolated Docker containers
- CPU/memory limits per submission
- Network isolation by default
- No inter-submission data access

## Documentation

- **[ARCHITECTURE.md](./ARCHITECTURE.md)** — Detailed system design
- **[SETUP.md](./SETUP.md)** — Development setup guide
- **[docs/API_SPEC.md](./docs/API_SPEC.md)** — API endpoints

## Author

**Pulkit Singh** — Full-stack architecture, bot fleet (concurrency), telemetry validation, Kubernetes setup.

GitHub: [@pulkit300405](https://github.com/pulkit300405)

## Timeline

Hackathon: May 9 - June 10, 2026  
Submission Deadline: June 9, 2026  
Status: In Development
