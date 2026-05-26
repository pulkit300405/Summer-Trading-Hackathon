# Development Setup Guide

## Prerequisites

### Required
- **Docker** (20.10+): https://docs.docker.com/install/
- **Docker Compose** (2.0+): Included with Docker Desktop
- **Git**: For version control

### Optional (for local development without Docker)
- **Go** 1.21+: https://golang.org/dl/
- **Node.js** 18+: https://nodejs.org/
- **PostgreSQL** client tools: `psql`, `pgweb`

---

## Quick Start (Recommended)

### 1. Clone Repository
```bash
git clone https://github.com/your-username/iicpc-summer-hackathon-2026.git
cd iicpc-summer-hackathon-2026
```

### 2. Start All Services
```bash
docker-compose up -d
```

This will:
- Start PostgreSQL (port 5432)
- Start Submission Handler (port 8080)
- Start Bot Fleet (port 8081)
- Start Telemetry Ingester (port 8082)
- Start WebSocket Server (port 8083)
- Start React Leaderboard (port 3000)

### 3. Verify All Services Are Running
```bash
# Check all containers are healthy
docker-compose ps

# Expected output:
# NAME                           STATUS
# iicpc-postgres                 healthy
# iicpc-submission-handler       healthy
# iicpc-bot-fleet                healthy
# iicpc-telemetry-ingester       healthy
# iicpc-ws-server                healthy
# iicpc-leaderboard              Up
```

### 4. Test the Platform

#### A. Health Checks
```bash
# Submission Handler
curl http://localhost:8080/health

# Bot Fleet
curl http://localhost:8081/health

# Telemetry Ingester
curl http://localhost:8082/health

# Expected response:
# {"status": "healthy", "timestamp": "2026-05-14T23:30:45Z"}
```

#### B. Upload a Test Submission
```bash
# Create a simple Go submission (orderbook)
cat > /tmp/test_orderbook.go << 'EOF'
package main

import (
  "fmt"
  "net/http"
)

func main() {
  http.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, `{"status":"ACK"}`)
  })
  http.ListenAndServe(":8000", nil)
}
EOF

# Upload the submission
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{
    "language": "go",
    "code": "'"$(cat /tmp/test_orderbook.go)"'",
    "team_name": "test_team"
  }'

# Expected response:
# {
#   "submission_id": "sub_123abc",
#   "container_id": "abc123xyz789",
#   "endpoint_url": "http://submission-handler:8100"
# }
```

#### C. View Leaderboard
Open in browser:
```
http://localhost:3000
```

You should see the leaderboard UI updating in real-time.

#### D. Query Metrics (Database)
```bash
# Open PostgreSQL shell
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics

# List tables
\dt

# Query metrics
SELECT submission_id, COUNT(*) as order_count, AVG(latency_ms) as avg_latency
FROM metrics
GROUP BY submission_id;

# Exit
\q
```

---

## Development Setup (Local Go Development)

If you want to develop locally without Docker rebuilding:

### 1. Install Go 1.21+
```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Verify
go version
```

### 2. Install PostgreSQL (Local)
```bash
# macOS
brew install postgresql

# Linux
sudo apt-get install postgresql postgresql-contrib

# Start PostgreSQL
brew services start postgresql  # macOS
sudo service postgresql start   # Linux
```

### 3. Initialize Database
```bash
# Create database
createdb iicpc_metrics

# Initialize schema
psql iicpc_metrics < scripts/init-db.sql
```

### 4. Run Each Service Locally

#### Submission Handler
```bash
cd submission-handler
export POSTGRES_URL="postgres://localhost/iicpc_metrics"
export DOCKER_HOST="unix:///var/run/docker.sock"
go run main.go
```

#### Bot Fleet
```bash
cd bot-fleet
export TARGET_URL="http://localhost:8080"
export TELEMETRY_URL="http://localhost:8082"
export NUM_BOTS="100"
go run main.go
```

#### Telemetry Ingester
```bash
cd telemetry-ingester
export POSTGRES_URL="postgres://localhost/iicpc_metrics"
go run main.go
```

#### Leaderboard (React)
```bash
cd leaderboard
npm install
npm start
```

---

## Useful Docker Commands

### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f submission-handler

# Last 100 lines
docker-compose logs --tail 100 bot-fleet
```

### Stop Services
```bash
# Stop all (containers stay)
docker-compose stop

# Stop specific service
docker-compose stop bot-fleet

# Stop and remove all
docker-compose down

# Stop and remove volumes (CAREFUL: deletes data)
docker-compose down -v
```

### Rebuild Services
```bash
# Rebuild all
docker-compose build

# Rebuild specific service (after code changes)
docker-compose build submission-handler

# Rebuild and restart
docker-compose up -d --build submission-handler
```

### Exec Commands in Containers
```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics

# View submission handler logs
docker-compose exec submission-handler cat /var/log/submission-handler.log

# Check bot goroutines
docker-compose exec bot-fleet ps aux
```

---

## Database Management

### Access PostgreSQL
```bash
# Using docker-compose
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics

# Or using local psql (if PostgreSQL installed locally)
psql -h localhost -U iicpc_user -d iicpc_metrics
```

### Useful Queries
```sql
-- Check table sizes
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check metrics count
SELECT COUNT(*) FROM metrics;

-- View recent metrics
SELECT * FROM metrics ORDER BY time DESC LIMIT 10;

-- Calculate latency percentiles
SELECT 
  submission_id,
  percentile_cont(0.50) WITHIN GROUP (ORDER BY latency_ms) as p50,
  percentile_cont(0.90) WITHIN GROUP (ORDER BY latency_ms) as p90,
  percentile_cont(0.99) WITHIN GROUP (ORDER BY latency_ms) as p99
FROM metrics
GROUP BY submission_id;

-- View leaderboard
SELECT * FROM leaderboard_metrics ORDER BY composite_score DESC;
```

### Reset Database (DEVELOPMENT ONLY)
```bash
# Drop all tables (WARNING: deletes all data)
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Reinitialize schema
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics < scripts/init-db.sql
```

---

## Troubleshooting

### Services Won't Start
```bash
# Check Docker daemon
docker ps

# Check compose version
docker-compose version

# Check for port conflicts
lsof -i :8080  # Check if port 8080 is in use
lsof -i :5432  # Check PostgreSQL port

# Solution: Kill process using the port
kill -9 <PID>
```

### Database Connection Errors
```bash
# Verify PostgreSQL is running
docker-compose ps postgres

# Check database initialization
docker-compose logs postgres

# Manually initialize if needed
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics < scripts/init-db.sql
```

### Bot Fleet Not Sending Orders
```bash
# Check bot logs
docker-compose logs bot-fleet

# Verify submission handler is running
curl http://localhost:8080/health

# Check target URL env variable
docker-compose exec bot-fleet env | grep TARGET_URL
```

### Leaderboard Not Updating
```bash
# Check WebSocket connection
docker-compose logs ws-server

# Check React logs
docker-compose logs leaderboard

# Browser console (F12) for JavaScript errors
```

### High PostgreSQL Disk Usage
```bash
# Check table sizes
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) FROM pg_tables WHERE schemaname='public' ORDER BY pg_total_relation_size DESC;"

# Vacuum (reclaim space)
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics -c "VACUUM FULL ANALYZE;"
```

---

## Performance Tuning

### Increase Bot Count (for testing)
Edit `docker-compose.yml`:
```yaml
bot-fleet:
  environment:
    NUM_BOTS: "5000"  # Increase from 100
    ORDERS_PER_SECOND: "10000"  # Increase TPS
```

Then restart:
```bash
docker-compose up -d --build bot-fleet
```

### Database Optimization
```bash
# Create index on submission_id for faster queries
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics -c "CREATE INDEX idx_metrics_submission ON metrics(submission_id, time DESC);"

# Enable compression for TimescaleDB
docker-compose exec postgres psql -U iicpc_user -d iicpc_metrics -c "ALTER TABLE metrics SET (timescaledb.compress, timescaledb.compress_orderby='time DESC');"
```

### WebSocket Performance
Edit `telemetry-ingester/ws-server.go` (when created):
```go
// Increase update frequency (default 1 Hz, can go to 10 Hz)
const updateInterval = 100 * time.Millisecond  // 10 updates per second
```

---

## CI/CD Setup (Optional)

### GitHub Actions Workflow
Create `.github/workflows/test.yml`:
```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test ./...
```

---

## Next Steps

1. ✅ Clone repo
2. ✅ Run `docker-compose up -d`
3. ✅ Verify all services via health checks
4. ✅ Upload test submission
5. ✅ Check leaderboard at `http://localhost:3000`
6. ✅ Start developing!

---

## Questions?

Check logs:
```bash
docker-compose logs -f
```

Check service health:
```bash
docker-compose ps
```

See [TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md) for more.
