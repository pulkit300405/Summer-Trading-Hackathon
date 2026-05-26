# System Architecture: Distributed Benchmarking Platform

**Document Version**: 1.0  
**Last Updated**: May 14, 2026  
**Team**: IICPC Summer Hackathon 2026

---

## Executive Summary

We are building a **Distributed Benchmarking and Hosting Platform** that:
1. Accepts contestant-submitted trading code (C++, Rust, Go)
2. Containerizes and isolates each submission in a secure sandbox
3. Generates a massive distributed load (5,000+ concurrent "trading bots")
4. Measures performance: latency (p50/p90/p99), throughput (TPS), correctness (FIFO fills)
5. Streams results to a real-time leaderboard

**Core Philosophy**: Contestants' code must survive extreme stress while maintaining correctness. Our platform is the *referee*—fair, accurate, resilient.

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    CONTESTANT SUBMISSION                         │
│              (orderbook/matching engine binary)                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                ┌────────────▼────────────┐
                │  SUBMISSION HANDLER     │
                │  (Receive & Validate)   │
                │  (Containerize)         │
                │  (Deploy)               │
                └────────────┬────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
   ┌────▼──────┐      ┌──────▼──────┐      ┌────▼──────┐
   │ BOT FLEET │      │ TELEMETRY   │      │LEADERBOARD│
   │ (Load Gen)├─────►│ INGESTER    ├─────►│  (React)  │
   └────┬──────┘      └──────┬──────┘      └──────────┘
        │                    │
        │ (orders sent)      │ (metrics collected)
        │                    │
        └────────────┬───────┘
                     │
            ┌────────▼──────────┐
            │   PostgreSQL +    │
            │   TimescaleDB     │
            │   (metrics store) │
            └───────────────────┘
```

---

## Component Deep Dive

### 1. **Submission & Sandboxing Engine**

**Responsibility**: Accept contestant code, validate, containerize, deploy in isolation.

**Architecture**:
```
HTTP Request (upload code)
        │
        ▼
┌─────────────────────────┐
│ Validation Layer        │ → Check file size, language support
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│ Dockerfile Generator    │ → Create isolated Dockerfile from code
│ (dynamic creation)      │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│ Docker Build & Push     │ → Build container image
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│ Container Registry      │ → Store image locally
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│ Container Orchestrator  │ → Launch container with resource limits
│ (docker run)            │   - CPU pinning (1 core)
│                         │   - Memory limit (512MB)
│                         │   - Network isolation (unique port)
└─────────────────────────┘
```

**Key Decisions**:
- **Dynamic Dockerfile generation**: Each submission gets auto-generated Dockerfile. Prevents escape vectors.
- **Resource limits**: CPU pinning + memory caps per submission prevent resource hogging.
- **Network isolation**: Each submission on unique port (8100+), prevents cross-contamination.
- **Restart policy**: Failed submissions auto-restart (fault tolerance).

**API**:
```
POST /submit
  - Input: code (string), language (string), team_name (string)
  - Output: submission_id, container_id, endpoint_url
  - Example: {"language":"go","code":"...","team_name":"team_alpha"}
```

**Technology**:
- Language: Go
- Docker SDK for Go (for containerization)
- Simple HTTP server (no heavy frameworks)

---

### 2. **Distributed Load Generator (Bot Fleet)**

**Responsibility**: Spawn thousands of concurrent "trading bots" that bombard submissions with realistic orders.

**Architecture**:
```
┌──────────────────────────────────────┐
│ Bot Fleet Manager                    │
│ (Coordinates bot lifecycle)          │
└───────┬──────────────────────────────┘
        │
    ┌───┴────────────────────────────────────────────────┐
    │                                                    │
    ▼ (Spawn 5000 goroutines)                           ▼
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│ Bot 1       │      │ Bot 2       │ ...  │ Bot 5000    │
│ (Goroutine) │      │ (Goroutine) │      │ (Goroutine) │
│ - Order gen │      │ - Order gen │      │ - Order gen │
│ - Send      │      │ - Send      │      │ - Send      │
│ - Track ack │      │ - Track ack │      │ - Track ack │
└──────┬──────┘      └──────┬──────┘      └──────┬──────┘
       │                    │                    │
       └────────────────────┼────────────────────┘
                            │
                    (All orders sent to
                     submission container)
```

**Bot Behavior**:
Each bot simulates a realistic market participant:
1. **Order Generation** (realistic mix):
   - 70% Limit orders (with time-in-force)
   - 20% Market orders
   - 10% Cancellations
2. **Timing**: Orders spaced randomly (Poisson distribution)
3. **Order Book Awareness**: Bots may adjust price based on market depth (if observable)

**Key Decisions**:
- **Goroutines over threads**: Go's lightweight goroutines allow 5K+ concurrent connections per machine
- **Connection pooling**: Reuse HTTP/WebSocket connections (faster, realistic)
- **Order rate control**: Configurable TPS (transactions per second) to avoid overwhelming network
- **Latency measurement**: Timestamp order at send, measure time-to-ack

**Metrics Collected** (sent to telemetry ingester):
```json
{
  "bot_id": 1234,
  "order_id": "order_5678",
  "timestamp_sent": "2026-05-14T23:30:45.123Z",
  "timestamp_ack": "2026-05-14T23:30:45.234Z",
  "latency_ms": 111,
  "order_type": "LIMIT",
  "price": 100.50,
  "quantity": 10,
  "status": "ACK" | "REJECTED" | "TIMEOUT"
}
```

**Technology**:
- Language: Go
- HTTP/2 client (for low-latency requests)
- WebSocket library (for real-time order streams)
- Built-in `time` package for latency measurement

---

### 3. **Telemetry & Validation Ingester**

**Responsibility**: Collect metrics from bot-submission interactions, validate correctness, aggregate to leaderboard.

**Architecture**:
```
┌────────────────────────────────────────┐
│ Bot Fleet emits order metrics          │
│ (latency, order_id, status, price)     │
└────────────────┬───────────────────────┘
                 │
         ┌───────▼────────┐
         │ Metrics Queue  │ (in-memory buffer, or Redis)
         └───────┬────────┘
                 │
         ┌───────▼──────────────────────┐
         │ Telemetry Collector          │
         │ - Parse metrics              │
         │ - Low-latency ingestion      │
         │ - Batch writes to DB         │
         └───────┬──────────────────────┘
                 │
         ┌───────▼──────────────────────┐
         │ Correctness Validator        │
         │ - FIFO order check           │
         │ - Fill accuracy validation   │
         │ - No double-fills            │
         │ - Price consistency          │
         └───────┬──────────────────────┘
                 │
         ┌───────▼──────────────────────┐
         │ Metrics Aggregator           │
         │ - Calculate p50/p90/p99      │
         │ - Throughput (TPS)           │
         │ - Correctness rate (%)       │
         │ - Composite score            │
         └───────┬──────────────────────┘
                 │
         ┌───────▼──────────────────────┐
         │ PostgreSQL + TimescaleDB     │
         │ - Time-series storage        │
         │ - Quick aggregation queries  │
         └──────────────────────────────┘
```

**Correctness Validation Rules**:

1. **FIFO Order Priority** (price-time):
   - Orders at same price filled in time-received order
   - Higher-priority orders (better price) filled first
   - Validated by: checking fill sequence against order timestamp

2. **Fill Accuracy**:
   - Each fill matches a valid outstanding order
   - Fill quantity ≤ order quantity
   - No fills after order cancellation
   - Validated by: comparing fill against order book state

3. **Latency Measurement**:
   - p50 (median): 50th percentile latency
   - p90 (tail): 90th percentile latency
   - p99 (extreme): 99th percentile latency
   - Calculated from all order ack latencies in measurement window

**Metrics Schema** (TimescaleDB):
```sql
CREATE TABLE metrics (
  time TIMESTAMPTZ NOT NULL,
  submission_id TEXT NOT NULL,
  bot_id INT,
  order_id TEXT,
  latency_ms INT,
  order_type TEXT,  -- LIMIT, MARKET, CANCEL
  status TEXT,      -- ACK, REJECTED, FILL, TIMEOUT
  price DECIMAL,
  quantity INT,
  fill_price DECIMAL,
  correctness_valid BOOLEAN
) PARTITION BY TIME (time INTERVAL '1 hour');

-- Aggregation table (updated every 5 seconds)
CREATE TABLE leaderboard_metrics (
  submission_id TEXT PRIMARY KEY,
  p50_latency_ms DECIMAL,
  p90_latency_ms DECIMAL,
  p99_latency_ms DECIMAL,
  throughput_tps DECIMAL,
  correctness_rate DECIMAL,
  composite_score DECIMAL,
  updated_at TIMESTAMPTZ
);
```

**Technology**:
- Language: Go
- Database: PostgreSQL with TimescaleDB extension (time-series optimized)
- Validation: Custom Go logic for correctness checks
- Message queue: Optional (start without, add Redis if needed)

---

### 4. **Real-Time Leaderboard & Analytics**

**Responsibility**: Display live metrics, rank contestants, update in real-time.

**Architecture**:
```
┌─────────────────────────────────┐
│ React Frontend                  │
│ - Leaderboard table             │
│ - Metrics graphs (p99, TPS)     │
│ - Submission status             │
└────────────┬────────────────────┘
             │
      ┌──────▼──────────┐
      │ WebSocket       │
      │ Connection      │
      └──────┬──────────┘
             │
      ┌──────▼──────────────┐
      │ Backend WS Server   │
      │ (Go + gorilla/ws)   │
      └──────┬──────────────┘
             │
      ┌──────▼──────────────────────┐
      │ Leaderboard Service         │
      │ - Query latest metrics      │
      │ - Push updates every 1s     │
      │ - Composite score calc      │
      └──────┬──────────────────────┘
             │
      ┌──────▼──────────────┐
      │ PostgreSQL          │
      │ (leaderboard_metrics│
      │  table)             │
      └─────────────────────┘
```

**Leaderboard UI**:
```
┌─────────────────────────────────────────────────────────────┐
│ IICPC Summer Hackathon 2026 - Live Leaderboard             │
├─────────────────────────────────────────────────────────────┤
│ Rank │ Team Name      │ p99(ms) │ TPS   │ Correct │ Score  │
├──────┼────────────────┼─────────┼───────┼─────────┼────────┤
│  1   │ Alpha Team     │  45.2   │ 12.5K │  99.8%  │ 9850   │
│  2   │ Beta Traders   │  67.8   │ 10.2K │  99.5%  │ 9620   │
│  3   │ Gamma Systems  │  89.1   │  8.5K │  98.9%  │ 9200   │
└──────┴────────────────┴─────────┴───────┴─────────┴────────┘

Detailed Metrics (selected team):
- p50 latency: 12.3ms
- p90 latency: 34.5ms
- p99 latency: 45.2ms
- Throughput: 12,500 TPS
- Correctness: 99.8%
- Orders processed: 1.2M
```

**Composite Score Calculation**:
```
Score = (Correctness_Rate × 100) × (Throughput / MaxTPS) / sqrt(p99_Latency)

Example:
- Correctness: 99.8% (0.998)
- Throughput: 10K TPS / 15K max = 0.667
- p99 Latency: 50ms
- Score = (0.998 × 100) × 0.667 / sqrt(50) = 99.8 × 0.667 / 7.07 ≈ 9.4K
```

**Technology**:
- Frontend: React 18, Vite (build tool)
- WebSocket: gorilla/websocket (Go backend)
- Charts: Recharts (React charting library)
- Real-time updates: 1 Hz (every 1 second)

---

## Data Flow (End-to-End)

```
1. SUBMISSION
   ┌─────────────────────────────────┐
   │ Contestant uploads code          │
   │ POST /submit (C++/Rust/Go code)  │
   └──────────────┬──────────────────┘
                  │
                  ▼
   ┌─────────────────────────────────┐
   │ Submission Handler validates    │
   │ Creates Dockerfile              │
   │ Builds container image          │
   │ Deploys to isolation layer      │
   └──────────────┬──────────────────┘
                  │
2. LOAD TESTING
   ├─────────────────────────────────┐
   │ Bot Fleet spawns 5K goroutines   │
   │ Each bot: generates orders,     │
   │ sends to submission container   │
   │ measures time-to-ack            │
   └────────────┬────────────────────┘
                │
3. METRICS COLLECTION
   ├─────────────────────────────────┐
   │ Bots emit metrics (latency, etc)│
   │ Telemetry Ingester collects     │
   │ Validates correctness (FIFO)    │
   │ Stores in TimescaleDB           │
   └────────────┬────────────────────┘
                │
4. LEADERBOARD UPDATE
   ├─────────────────────────────────┐
   │ Every 1 second:                 │
   │ - Calculate p50/p90/p99         │
   │ - Calculate TPS                 │
   │ - Calculate correctness rate    │
   │ - Compute composite score       │
   │ - Push to frontend via WS       │
   └────────────┬────────────────────┘
                │
   ┌────────────▼──────────────────────┐
   │ React Leaderboard updates         │
   │ (contestants watch live scores)   │
   └───────────────────────────────────┘
```

---

## Key Design Decisions & Rationale

### 1. **Why Go?**
- **Concurrency**: Goroutines make 5K+ concurrent bots trivial. Java threads are overkill.
- **Performance**: Single-threaded concurrency with minimal memory overhead (~1MB per 1000 goroutines).
- **Deployability**: Compiles to single binary, runs in ~10MB Docker image.
- **Learning curve**: Simple syntax, excellent for systems programming.

**Tradeoff**: Smaller ecosystem vs. Java, but more than enough for this use case.

### 2. **Why PostgreSQL + TimescaleDB?**
- **Time-series queries**: TimescaleDB makes p50/p90/p99 calculations trivial (built-in).
- **Scalability**: Can handle millions of metrics/second with compression.
- **Simplicity**: No additional infrastructure (Kafka, Redis, etc.).
- **Aggregation**: SQL window functions calculate percentiles efficiently.

**Tradeoff**: Single-node bottleneck at extreme scale, but sufficient for 5K bots.

### 3. **Why Docker Compose (not Kubernetes)?**
- **Local development**: One `docker-compose up` spins up entire stack.
- **Simplicity**: No Kubernetes complexity (no ETCD, no scheduling overhead).
- **Portability**: Works on laptop, cloud, anywhere Docker runs.
- **Sufficient scale**: 5K bots per machine is achievable without orchestration.

**Path to scale**: If needed, Docker Swarm or Kubernetes manifests provided as IaC.

### 4. **Why Dynamic Dockerfile Generation?**
- **Security**: We control the container, not contestant code.
- **Isolation**: Prevents escape vectors (e.g., malicious Dockerfile).
- **Consistency**: All submissions run under same resource constraints.
- **Validation**: We can enforce language version, dependencies.

**Tradeoff**: Can't support every build system, but covers C++, Rust, Go.

### 5. **Why Goroutines over Threads?**
- **Memory**: 1000 goroutines = ~1MB; 1000 threads = ~10MB+.
- **Context switching**: Goroutines have minimal overhead.
- **Multiplexing**: Go runtime efficiently schedules goroutines on available CPUs.
- **Concurrency model**: Simpler than thread pools, channels for communication.

---

## Resilience & Fault Tolerance

### Failure Scenarios & Handling

| Scenario | Handling |
|----------|----------|
| Submission crashes | Auto-restart, mark as "failed", stop bots |
| Bot network timeout | Retry with exponential backoff, mark as TIMEOUT |
| Telemetry DB full | Rotate logs, keep recent data (1 week) |
| Frontend disconnects | WebSocket reconnect, catch up on metrics |
| Bot goroutine panic | Recover, log, continue (circuit breaker pattern) |
| Metrics calculation error | Fallback to previous value, log alert |

### High Availability

- **Stateless services**: Bots, telemetry can horizontally scale (add machines)
- **Persistent metrics**: PostgreSQL replicated (for production)
- **Health checks**: Each service exports `/health` endpoint
- **Graceful shutdown**: Services drain connections before terminating

---

## Testing Strategy

### Unit Tests
- Order generation logic (correct order types, prices)
- Correctness validation (FIFO priority, fill accuracy)
- Latency aggregation (p50/p90/p99 calculation)

### Integration Tests
- End-to-end: upload code → deploy → run bots → measure metrics
- Correctness: inject known orders, verify fills are correct
- Stress: 5K concurrent bots for 1 minute, measure stability

### Load Tests
- Max throughput: How many TPS can submission handle?
- Max bots: How many concurrent goroutines can single machine support?
- Metrics pipeline: Can telemetry keep up with order rate?

---

## Scaling Plan

### Phase 1 (Current)
- Single submission at a time
- 5K bots per machine
- Docker Compose (local)

### Phase 2 (Stretch)
- Multiple submissions in parallel
- 10K bots across 2 machines (Docker Swarm)
- Redis for metrics queue (reduce DB writes)

### Phase 3 (Production)
- Kubernetes orchestration
- PostgreSQL replication (HA)
- Bot fleet across multiple nodes
- Distributed metrics aggregation

---

## Security Considerations

### Submission Isolation
- **Network**: Each submission on unique port, no inter-submission traffic
- **Filesystem**: Read-only root, temporary scratch space (tmpfs)
- **Process**: No privileged access, limited syscalls
- **Resource limits**: CPU pinning (1 core), memory (512MB), file descriptors (1024)

### Platform Security
- **Input validation**: Code upload size limit, language whitelist
- **Secrets**: Database credentials in environment (not in code)
- **Logging**: All orders logged (audit trail), PII redacted
- **Monitoring**: Alerts for anomalous behavior (sudden crashes, high memory)

---

## Monitoring & Observability

### Metrics Exposed
```
Submission Handler:
  - upload_latency_ms (histogram)
  - container_build_duration_ms (histogram)
  - submission_status (gauge: running, failed, completed)

Bot Fleet:
  - orders_sent_total (counter)
  - order_latency_ms (histogram)
  - bot_goroutines_active (gauge)

Telemetry Ingester:
  - metrics_ingested_total (counter)
  - validation_errors_total (counter)
  - database_write_latency_ms (histogram)

Leaderboard:
  - websocket_connections_active (gauge)
  - leaderboard_update_latency_ms (histogram)
```

### Logging
- Structure: JSON logs (easy to parse)
- Level: DEBUG in dev, INFO in prod
- Correlation ID: Trace requests across services

---

## Deliverables Checklist

- [x] Architecture design (this document)
- [ ] Working prototype (all 4 components)
- [ ] Docker Compose (runs entire platform)
- [ ] Kubernetes manifests (IaC)
- [ ] 5K concurrent bots with accurate metrics
- [ ] Real-time leaderboard
- [ ] Correctness validation (FIFO, fills)
- [ ] README with setup instructions

---

## References & Resources

- Go concurrency: https://go.dev/blog/pipelines
- TimescaleDB docs: https://docs.timescale.com/
- Docker best practices: https://docs.docker.com/develop/dev-best-practices/
- React + WebSocket: https://socket.io/docs/v4/react/

---

**Next Steps**:
1. Scaffold project structure (directories, go.mod, docker-compose.yml)
2. Build Submission Handler (weeks 1-2)
3. Build Bot Fleet (weeks 1-2)
4. Build Telemetry Ingester (week 2)
5. Build Leaderboard (week 2-3)
6. Stress test & polish (week 3-4)
