-- Initialize IICPC Metrics Database
-- TimescaleDB extension for time-series metrics

-- Create TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Metrics table: Raw order metrics from bot-submission interactions
CREATE TABLE IF NOT EXISTS metrics (
  time TIMESTAMPTZ NOT NULL,
  submission_id TEXT NOT NULL,
  bot_id INT NOT NULL,
  order_id TEXT NOT NULL,
  order_type TEXT NOT NULL,  -- LIMIT, MARKET, CANCEL
  status TEXT NOT NULL,      -- ACK, REJECTED, FILL, TIMEOUT, ERROR
  price DECIMAL(10, 2),
  quantity INT,
  fill_price DECIMAL(10, 2),
  fill_quantity INT,
  latency_ms INT,
  correctness_valid BOOLEAN DEFAULT true,
  error_message TEXT
);

-- Create hypertable for time-series compression
SELECT create_hypertable('metrics', 'time', if_not_exists => TRUE);

-- Enable compression for cost reduction
ALTER TABLE metrics SET (
  timescaledb.compress,
  timescaledb.compress_orderby = 'time DESC'
);

-- Compression policy: compress data older than 1 hour
SELECT add_compression_policy('metrics', INTERVAL '1 hour', if_not_exists => TRUE);

-- Create indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_metrics_submission_time 
  ON metrics (submission_id, time DESC);

CREATE INDEX IF NOT EXISTS idx_metrics_bot_id 
  ON metrics (bot_id, time DESC);

CREATE INDEX IF NOT EXISTS idx_metrics_order_type 
  ON metrics (order_type);

CREATE INDEX IF NOT EXISTS idx_metrics_status 
  ON metrics (status);

-- Aggregated leaderboard metrics (updated every 5 seconds)
CREATE TABLE IF NOT EXISTS leaderboard_metrics (
  submission_id TEXT PRIMARY KEY,
  team_name TEXT,
  orders_processed INT DEFAULT 0,
  p50_latency_ms DECIMAL(10, 2),
  p90_latency_ms DECIMAL(10, 2),
  p99_latency_ms DECIMAL(10, 2),
  throughput_tps DECIMAL(10, 2),
  correctness_rate DECIMAL(5, 2),  -- Percentage (0-100)
  composite_score DECIMAL(10, 2),
  status TEXT DEFAULT 'running',  -- running, completed, failed
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  CONSTRAINT chk_correctness_rate CHECK (correctness_rate >= 0 AND correctness_rate <= 100)
);

-- Submissions table: Track uploaded submissions
CREATE TABLE IF NOT EXISTS submissions (
  submission_id TEXT PRIMARY KEY,
  team_name TEXT NOT NULL,
  language TEXT NOT NULL,  -- go, rust, cpp, python
  container_id TEXT,
  endpoint_url TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  status TEXT DEFAULT 'pending',  -- pending, building, running, completed, failed
  error_message TEXT
);

-- Create index for submission queries
CREATE INDEX IF NOT EXISTS idx_submissions_status 
  ON submissions (status);

CREATE INDEX IF NOT EXISTS idx_submissions_created_at 
  ON submissions (created_at DESC);

-- Audit log: Track all orders for correctness validation
CREATE TABLE IF NOT EXISTS audit_log (
  id BIGSERIAL PRIMARY KEY,
  time TIMESTAMPTZ DEFAULT NOW(),
  submission_id TEXT NOT NULL,
  event_type TEXT NOT NULL,  -- ORDER_SENT, ORDER_ACK, ORDER_FILL, ORDER_REJECT, ORDER_CANCEL
  order_id TEXT,
  price DECIMAL(10, 2),
  quantity INT,
  details JSONB
);

-- Create index for audit log
CREATE INDEX IF NOT EXISTS idx_audit_log_submission_time 
  ON audit_log (submission_id, time DESC);

-- Health check table
CREATE TABLE IF NOT EXISTS service_health (
  service_name TEXT PRIMARY KEY,
  status TEXT NOT NULL,  -- healthy, unhealthy, degraded
  last_check TIMESTAMPTZ DEFAULT NOW(),
  last_healthy_at TIMESTAMPTZ,
  error_message TEXT
);

-- Initialize service health records
INSERT INTO service_health (service_name, status, last_healthy_at)
VALUES 
  ('submission-handler', 'healthy', NOW()),
  ('bot-fleet', 'healthy', NOW()),
  ('telemetry-ingester', 'healthy', NOW()),
  ('ws-server', 'healthy', NOW())
ON CONFLICT (service_name) DO UPDATE SET 
  status = EXCLUDED.status,
  last_healthy_at = EXCLUDED.last_healthy_at;

-- View: Leaderboard ranking
CREATE OR REPLACE VIEW leaderboard_ranking AS
SELECT 
  ROW_NUMBER() OVER (ORDER BY composite_score DESC) as rank,
  submission_id,
  team_name,
  orders_processed,
  p50_latency_ms,
  p90_latency_ms,
  p99_latency_ms,
  throughput_tps,
  correctness_rate,
  composite_score,
  updated_at
FROM leaderboard_metrics
WHERE status = 'running' OR status = 'completed'
ORDER BY composite_score DESC;

-- View: Submission details with metrics
CREATE OR REPLACE VIEW submission_details AS
SELECT 
  s.submission_id,
  s.team_name,
  s.language,
  s.status,
  s.endpoint_url,
  s.created_at,
  s.started_at,
  s.completed_at,
  l.orders_processed,
  l.p99_latency_ms,
  l.throughput_tps,
  l.correctness_rate,
  l.composite_score
FROM submissions s
LEFT JOIN leaderboard_metrics l ON s.submission_id = l.submission_id;

-- Permissions (optional: for multi-user access)
-- GRANT CONNECT ON DATABASE iicpc_metrics TO iicpc_user;
-- GRANT USAGE ON SCHEMA public TO iicpc_user;
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO iicpc_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO iicpc_user;

-- Print initialization status
SELECT 'Database initialized successfully' as status;
SELECT 'TimescaleDB version:' as info, extversion FROM pg_extension WHERE extname = 'timescaledb';
