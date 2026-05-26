# Technical Decisions Document

## Overview

This document outlines the key technical decisions made for the IICPC Summer Hackathon 2026 Distributed Benchmarking Platform and the rationale behind each.

---

## 1. Language Selection: Go

### Decision
**Use Go for all backend services** (Submission Handler, Bot Fleet, Telemetry Ingester)

### Rationale

| Aspect | Go | Alternative (Java) | Alternative (Python) |
|--------|----|--------------------|----------------------|
| Concurrency | Goroutines (1000s with minimal memory) | Thread pools (overkill, 10MB+ per 1000 threads) | Asyncio (complex, GIL limits true parallelism) |
| Compilation | Single binary, 10-20MB Docker image | JVM overhead, 500MB+ image | Interpreted, deployment issues |
| Performance | Fast (compiled), low latency | Good but heavier | Slow for I/O-bound operations |
| Learning curve | Simple syntax, easy for systems programming | Complex frameworks, verbose | Simple but lacks typed rigor |
| Ecosystem | Excellent for networks/systems | Mature but bloated | Good but less standardized |

### Why Goroutines Win
- **Lightweight**: 1,000 goroutines ≈ 1MB; 1,000 threads ≈ 10MB+
- **Simple**: `go func() {}` vs Java thread pool configuration
- **Efficient**: Goroutines are multiplexed on a few OS threads (M:N scheduling)
- **Perfect for load generation**: Spawning 5K concurrent bots is trivial in Go

### Example: 5K Bots in Go
```go
for i := 0; i < 5000; i++ {
  go bot.run()  // Spawn 5000 concurrent goroutines
}
// That's it. Go runtime handles scheduling, memory, everything.
```

---

## 2. Database: PostgreSQL + TimescaleDB

### Decision
**Use PostgreSQL 15 with TimescaleDB extension for metrics storage**

### Rationale

| Feature | PostgreSQL + TimescaleDB | InfluxDB | Prometheus | Cassandra |
|---------|--------------------------|----------|-----------|-----------|
| Time-series optimization | ✅ TimescaleDB extension | ✅ Built-in | ✅ Built-in | ✅ Optimized |
| Query language | ✅ SQL (familiar) | ❌ InfluxQL | ❌ Prometheus QL | ❌ CQL |
| Percentile queries | ✅ Built-in functions | ⚠️ Complex | ❌ Not well-suited | ⚠️ Complex |
| Simplicity | ✅ Single database | ❌ Add-ons needed | ❌ Additional infrastructure | ❌ Complex cluster |
| Compression | ✅ TimescaleDB compression | ✅ Good | ✅ Good | ⚠️ Requires tuning |
| For hackathon? | ✅ BEST | ❌ Overkill | ❌ Pull-only | ❌ Too complex |

### Key Advantages
1. **Time-series optimized**: TimescaleDB compresses and partitions by time automatically
2. **SQL for aggregations**: Calculate p50/p90/p99 with standard `percentile_cont()` function
3. **No additional infrastructure**: Single database, no message queues needed initially
4. **Replication-ready**: PostgreSQL has mature replication for production
5. **Easy to debug**: Can query directly with `psql`, no proprietary tools

### Example: P99 Latency Calculation
```sql
-- Calculate p99 latency for a submission (simple!)
SELECT percentile_cont(0.99) WITHIN GROUP (ORDER BY latency_ms)
FROM metrics
WHERE submission_id = 'sub_123' AND time > NOW() - INTERVAL '1 minute';
-- Result: 45.2 (ms)
```

---

## 3. Containerization: Docker Compose (Local) + Kubernetes (IaC)

### Decision
**Use Docker Compose for local development, Kubernetes manifests for production-ready IaC**

### Rationale

| Setup | Docker Compose | Kubernetes | Docker Swarm |
|-------|----------------|-----------|--------------|
| Local dev | ✅ Trivial | ❌ Overkill | ⚠️ OK |
| `docker-compose up` | ✅ One command | ❌ Complex setup | ⚠️ Multiple commands |
| Production scaling | ⚠️ Needs migration | ✅ Built for scale | ⚠️ Limited |
| Learning curve | ✅ Easy | ❌ Steep | ⚠️ Medium |
| Hackathon speed | ✅ BEST | ❌ Slow to setup | ⚠️ OK |

### Why Docker Compose for Development
- **One command**: `docker-compose up` starts entire platform locally
- **No K8s complexity**: No ETCD, no scheduler, no networking drama
- **Portable**: Works on laptop, Cloud VMs, anywhere Docker runs
- **Sufficient scale**: 5K bots per machine is achievable without orchestration

### Path to Scale
- **Week 1-3**: Docker Compose (local testing)
- **Week 4 (if needed)**: Add Kubernetes manifests (included in `infrastructure/k8s/`)
- **Production**: Deploy with K8s for multi-node scaling

---

## 4. Load Generator Architecture: Goroutines > Thread Pools

### Decision
**Use Go goroutines for bot concurrency, NOT thread pools or event loops**

### Rationale

```
┌──────────────────────────────────────────────────────┐
│ 5,000 Bots                                           │
├──────────────────────────────────────────────────────┤
│ Go Goroutines (lightweight)                          │
│ - 1 goroutine per bot                                │
│ - M:N scheduling on OS threads                       │
│ - ~1MB total memory for 5K goroutines               │
│ - Simple: go bot.run()                               │
│                                                       │
│ VS                                                   │
│                                                       │
│ Java Thread Pool (heavyweight)                       │
│ - ThreadPoolExecutor with ~1000 worker threads      │
│ - 1:1 mapping with OS threads                        │
│ - ~10MB+ memory for 5K threads                      │
│ - Complex: executor.submit(bot)                      │
│                                                       │
│ GOROUTINES WIN ✓                                     │
└──────────────────────────────────────────────────────┘
```

### Why This Matters
- **Memory**: Goroutines fit on a laptop; Java threads need beefy machines
- **Context switching**: Go scheduler is O(1); OS thread scheduler is O(n)
- **Simplicity**: Goroutines eliminate thread pool configuration nightmares
- **Scaling**: Easy to scale from 100 bots to 5K without code changes

---

## 5. Real-time Leaderboard: React + WebSocket

### Decision
**Use React frontend with WebSocket connection to Go backend for real-time updates**

### Rationale

| Feature | React + WebSocket | Vue + WebSocket | Angular | Static HTML |
|---------|-------------------|----------------|---------|-------------|
| Real-time updates | ✅ Native | ✅ Native | ✅ Native | ❌ Polling |
| Development speed | ✅ Fast | ✅ Fast | ⚠️ Slower | ✅ Fastest |
| UI complexity | ✅ Great | ✅ Good | ✅ Excellent | ❌ Limited |
| Team familiarity | ✅ Most common | ⚠️ Less common | ❌ Less common | ✅ Familiar |
| For hackathon? | ✅ BEST | ✅ OK | ❌ Overkill | ✅ OK |

### Why React
- **Ecosystem**: Recharts (charting), large component library
- **Tooling**: Vite is fast and requires minimal config
- **Team knowledge**: More engineers know React than Vue/Angular
- **Real-time**: WebSocket integration is straightforward
- **Deployment**: Builds to static HTML, easy to serve

### Why WebSocket (not HTTP polling)
- **Efficiency**: Push updates instead of polling every second
- **Latency**: Real-time (sub-100ms) vs polling (1-5s delay)
- **Bandwidth**: Single persistent connection vs repeated HTTP calls

---

## 6. Metrics Pipeline: In-Process Channels (Initially)

### Decision
**Start with in-process Go channels for metrics queueing, migrate to Redis if needed**

### Rationale

| Component | In-Process Channel | Redis Queue | Kafka |
|-----------|-------------------|-------------|-------|
| Setup complexity | ✅ Zero (built-in) | ⚠️ One Docker container | ❌ Complex setup |
| Throughput | ✅ Sufficient (10K TPS) | ✅ Excellent | ✅ Excellent |
| Persistence | ❌ In-memory only | ✅ Persistent | ✅ Persistent |
| For week 1-3? | ✅ PERFECT | ⚠️ Overkill | ❌ Overkill |
| For production? | ⚠️ May hit limits | ✅ Good | ✅ Best-in-class |

### Evolution Path
1. **Week 1**: In-process channel (simple, no deps)
2. **Week 3** (if needed): Add Redis for durability
3. **Production**: Kafka for distributed streaming (not needed for hackathon)

### Why Not Kafka from Day 1?
- **Complexity**: Requires ZooKeeper, multiple brokers, cluster setup
- **Overkill**: Hackathon has 1 platform doing 1 thing
- **Time**: 2-3 days to setup properly; we have 4 weeks total
- **Simplicity wins**: In-process channels handle 10K+ TPS easily

---

## 7. Submission Containerization: Dynamic Dockerfile Generation

### Decision
**Auto-generate Dockerfile for each submission instead of accepting pre-built images**

### Rationale

| Approach | Dynamic Generation | Pre-built Docker Images | Host process |
|----------|-------------------|----------------------|---------------|
| Security | ✅ We control sandbox | ❌ Trust contestant | ❌ No isolation |
| Consistency | ✅ All equal constraints | ⚠️ Varies per image | ❌ No constraints |
| Resource limits | ✅ CPU/memory enforced | ⚠️ Contestant config | ❌ No limits |
| Escape vectors | ✅ Minimal | ❌ Malicious Dockerfile | ❌ Direct access |
| Simplicity | ⚠️ Code generation | ✅ Just accept URL | ✅ Trivial |

### Why Dynamic Generation
- **Security**: We own the containerization, prevents malicious code escapes
- **Fairness**: All submissions run under identical resource constraints
- **Validation**: We can enforce language versions, dependency lists
- **Isolation**: We control network, filesystem, process access

### Example: Supported Languages
```
go:
  - Source file: main.go
  - Build: go build -o app
  - Run: ./app
  
rust:
  - Source file: main.rs
  - Build: rustc -O main.rs
  - Run: ./main

cpp:
  - Source file: main.cpp
  - Build: g++ -O3 -o app main.cpp
  - Run: ./app
```

---

## 8. Correctness Validation: Post-hoc Analysis

### Decision
**Validate order correctness after submission completes, not in real-time**

### Rationale

| Approach | Real-time Validation | Post-hoc Analysis |
|----------|-------------------|-------------------|
| Implementation | Complex (state machine) | Simple (replay logs) |
| Accuracy | ✅ Perfect | ✅ Perfect |
| Performance impact | ❌ Slows submission | ✅ No impact |
| Debugging | ⚠️ Hard to trace | ✅ Full audit log |
| For hackathon? | ❌ Overengineering | ✅ BETTER |

### How It Works
1. **Bots send orders** → Submission responds
2. **Metrics ingester records everything** (latency, status, price)
3. **After submission done**:
   - Pull complete audit log
   - Replay orders in sequence
   - Validate FIFO, fills, no double-fills
   - Record correctness score

### Advantages
- **Simple**: No complex state machine in bot code
- **Accurate**: Can re-run validation with different rules
- **Debuggable**: Full audit trail for post-mortems
- **Fast**: Doesn't block load testing

---

## 9. Scaling Strategy: Single Machine → Multi-Node (Progressive)

### Decision
**Design for single machine initially, add multi-node capability in week 3-4**

### Rationale

| Phase | Scale | Implementation | Timeline |
|-------|-------|----------------|----------|
| Phase 1 | 1 machine, 5K bots | Docker Compose, in-memory metrics | Weeks 1-2 |
| Phase 2 | 2 machines, 10K bots | Docker Compose + Redis, Kubernetes ready | Week 3 |
| Phase 3 | N machines, 100K bots | Kubernetes, distributed metrics | Week 4 (stretch) |

### Why Progressive Scaling
- **Week 1-2**: Focus on correctness, not scale
- **Week 3**: If bottleneck found, add horizontal scaling
- **Week 4**: If time permits, full K8s deployment

### Single-Machine Capabilities
```
Bot Fleet (Go):
  - 5,000 concurrent goroutines
  - ~100MB memory
  - ~500MB CPU usage (multi-core)
  - Can run on 2-core laptop

Submission Container:
  - 1 core dedicated
  - 512MB memory limit
  - Network isolated

Telemetry Ingester:
  - Batch inserts (1000 metrics/second)
  - PostgreSQL handles 10K+ TPS

Leaderboard:
  - React frontend (static)
  - Go WebSocket server
  - Updates every 1 second
```

---

## 10. Error Handling: Graceful Degradation

### Decision
**Fail fast, log extensively, but never crash the platform**

### Patterns Used

```go
// Pattern 1: Recoverable errors (log and continue)
if err := submitOrder(order); err != nil {
  log.Printf("⚠️  Order failed: %v", err)
  recordMetric(order, "REJECTED")
  continue  // Don't stop bot, keep going
}

// Pattern 2: Panic recovery (catch crashes)
defer func() {
  if r := recover(); r != nil {
    log.Printf("⚠️  Bot crashed: %v, restarting", r)
    go bot.run()  // Restart bot
  }
}()

// Pattern 3: Timeouts (prevent hanging)
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
resp, err := client.Do(req.WithContext(ctx))
```

### Why This Matters
- **Resilience**: One bot crash doesn't affect others
- **Observability**: Every error is logged for post-mortem analysis
- **Graceful degradation**: If one service is slow, others continue

---

## Summary: Why These Choices?

| Decision | Optimizes For | Wins Hackathon By |
|----------|---------------|------------------|
| Go | Simplicity + Concurrency | Fast iteration, 5K bots trivial |
| PostgreSQL + TimescaleDB | Simplicity + Time-series | No infrastructure overhead, SQL percentiles |
| Docker Compose | Development speed | `docker-compose up` one-liner |
| Goroutines | Memory + Performance | 5K bots on laptop |
| React + WebSocket | Real-time updates | Live leaderboard out-of-the-box |
| Dynamic Dockerfiles | Security | Fair, isolated submissions |
| Single-machine scaling | Focus | Perfect for 4-week timeline |

---

## Trade-offs Made

| What We Didn't Choose | Why | Cost |
|----------------------|------|------|
| Java + Spring | Language preference | None (Go is faster anyway) |
| InfluxDB | Added complexity | None (PostgreSQL sufficient) |
| Kubernetes day 1 | Speed | None (compose works great) |
| Kafka | Overkill | None (channels fast enough) |
| Real-time validation | Complexity | Slight (post-hoc validation fine) |
| Multiple machines | Scope | None (single machine is sufficient) |

---

## Technology Matrix (Final Stack)

```
┌─────────────────────────────────────────────────────────┐
│ IICPC Summer Hackathon 2026 - Tech Stack               │
├─────────────────────────────────────────────────────────┤
│ SERVICE              │ LANGUAGE │ FRAMEWORK  │ PURPOSE   │
├──────────────────────┼──────────┼────────────┼───────────┤
│ Submission Handler   │ Go       │ net/http   │ Uploads   │
│ Bot Fleet            │ Go       │ goroutines │ Load Gen  │
│ Telemetry Ingester   │ Go       │ net/http   │ Metrics   │
│ WebSocket Server     │ Go       │ gorilla/ws │ Real-time │
│ Leaderboard          │ React    │ Vite       │ Frontend  │
│ Metrics Storage      │ SQL      │ PostgreSQL │ Data      │
│ Orchestration        │ YAML     │ Compose    │ Deploy    │
│ IaC (Production)     │ YAML     │ Kubernetes │ Scale     │
└─────────────────────────────────────────────────────────┘
```

---

## Questions for the Team

1. **Do we need authentication?** (Likely no for hackathon, add in production)
2. **Should we support network communication?** (Probably restrict for now)
3. **Do we rate-limit submissions?** (Yes, 1 per team at a time)
4. **What's max submission size?** (100KB code limit)

---

## Future Improvements (After Hackathon)

1. **Distributed Tracing**: Add Jaeger for request tracing
2. **Metrics Export**: Prometheus-compatible `/metrics` endpoint
3. **Auth & RBAC**: Team-based access control
4. **Persistence**: Durable message queue (Redis/Kafka)
5. **Advanced Analytics**: ML-based anomaly detection
6. **Multi-language support**: Python, JavaScript submissions
7. **WebAssembly**: Run submissions in WASM sandbox (browser-like isolation)
