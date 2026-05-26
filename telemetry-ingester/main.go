package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type TelemetryServer struct {
	db     *sql.DB
	port   string
	metrics chan MetricEvent
	mu     sync.RWMutex
}

type MetricEvent struct {
	BotID             int     `json:"bot_id"`
	OrderID           string  `json:"order_id"`
	TimestampSent     int64   `json:"timestamp_sent"`
	TimestampAck      int64   `json:"timestamp_ack"`
	LatencyMs         int     `json:"latency_ms"`
	OrderType         string  `json:"order_type"`
	Status            string  `json:"status"`
	Price             float64 `json:"price"`
	Quantity          int     `json:"quantity"`
	CorrectnessValid  bool    `json:"correctness_valid"`
	SubmissionID      string  `json:"submission_id,omitempty"`
}

type LeaderboardEntry struct {
	Rank             int     `json:"rank"`
	SubmissionID     string  `json:"submission_id"`
	TeamName         string  `json:"team_name"`
	OrdersProcessed  int     `json:"orders_processed"`
	P50LatencyMs     float64 `json:"p50_latency_ms"`
	P90LatencyMs     float64 `json:"p90_latency_ms"`
	P99LatencyMs     float64 `json:"p99_latency_ms"`
	ThroughputTps    float64 `json:"throughput_tps"`
	CorrectnessRate  float64 `json:"correctness_rate"`
	CompositeScore   float64 `json:"composite_score"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
}

func main() {
	// Get environment variables
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		dbURL = "postgres://iicpc_user:iicpc_password_dev@localhost/iicpc_metrics?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✓ Connected to PostgreSQL")

	server := &TelemetryServer{
		db:      db,
		port:    port,
		metrics: make(chan MetricEvent, 10000),
	}

	// Start metrics ingestion worker
	go server.metricsWorker()

	// Start leaderboard aggregation (every 5 seconds)
	go server.aggregationWorker()

	// Register HTTP handlers
	http.HandleFunc("/health", server.healthHandler)
	http.HandleFunc("/metrics", server.metricsHandler)
	http.HandleFunc("/leaderboard", server.leaderboardHandler)
	http.HandleFunc("/", server.indexHandler)

	// Start server
	addr := ":" + port
	log.Printf("📊 Telemetry Ingester starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// metricsHandler: POST /metrics - Accept metrics from bots
func (s *TelemetryServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var metric MetricEvent
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Queue for batch insertion
	select {
	case s.metrics <- metric:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	default:
		http.Error(w, "Metrics queue full", http.StatusServiceUnavailable)
	}
}

// metricsWorker: Batch insert metrics into database
func (s *TelemetryServer) metricsWorker() {
	batch := make([]MetricEvent, 0, 1000)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case metric := <-s.metrics:
			batch = append(batch, metric)

			// Insert when batch is full
			if len(batch) >= 1000 {
				s.insertMetrics(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// Insert remaining metrics
			if len(batch) > 0 {
				s.insertMetrics(batch)
				batch = batch[:0]
			}
		}
	}
}

// insertMetrics: Batch insert metrics into database
func (s *TelemetryServer) insertMetrics(metrics []MetricEvent) {
	if len(metrics) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ Failed to begin transaction: %v", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics (time, submission_id, bot_id, order_id, order_type, status, price, quantity, latency_ms, correctness_valid)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)
	if err != nil {
		log.Printf("❌ Failed to prepare statement: %v", err)
		return
	}
	defer stmt.Close()

	for _, m := range metrics {
		_, err := stmt.ExecContext(ctx,
			time.UnixMilli(m.TimestampAck),
			"TODO",  // submission_id not provided by bot, TODO: add context
			m.BotID,
			m.OrderID,
			m.OrderType,
			m.Status,
			m.Price,
			m.Quantity,
			m.LatencyMs,
			m.CorrectnessValid,
		)
		if err != nil {
			log.Printf("⚠️  Failed to insert metric: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("❌ Failed to commit transaction: %v", err)
		return
	}

	log.Printf("✓ Inserted %d metrics", len(metrics))
}

// aggregationWorker: Update leaderboard metrics every 5 seconds
func (s *TelemetryServer) aggregationWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.updateLeaderboard()
	}
}

// updateLeaderboard: Calculate aggregate metrics and update leaderboard_metrics table
func (s *TelemetryServer) updateLeaderboard() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get distinct submissions
	rows, err := s.db.QueryContext(ctx, "SELECT DISTINCT submission_id FROM submissions WHERE status = 'running'")
	if err != nil {
		log.Printf("⚠️  Failed to query submissions: %v", err)
		return
	}
	defer rows.Close()

	var submissions []string
	for rows.Next() {
		var subID string
		if err := rows.Scan(&subID); err != nil {
			continue
		}
		submissions = append(submissions, subID)
	}

	// Update metrics for each submission
	for _, subID := range submissions {
		s.calculateAndStoreMetrics(ctx, subID)
	}
}

// calculateAndStoreMetrics: Calculate metrics for a specific submission
func (s *TelemetryServer) calculateAndStoreMetrics(ctx context.Context, submissionID string) {
	// Get all metrics for this submission (last 60 seconds)
	since := time.Now().Add(-60 * time.Second)

	query := `
		SELECT latency_ms, status
		FROM metrics
		WHERE submission_id = $1 AND time > $2
		ORDER BY time DESC
	`

	rows, err := s.db.QueryContext(ctx, query, submissionID, since)
	if err != nil {
		log.Printf("⚠️  Failed to query metrics for %s: %v", submissionID, err)
		return
	}
	defer rows.Close()

	var latencies []int
	var statusCounts = make(map[string]int)
	totalOrders := 0

	for rows.Next() {
		var latencyMs int
		var status string

		if err := rows.Scan(&latencyMs, &status); err != nil {
			continue
		}

		latencies = append(latencies, latencyMs)
		statusCounts[status]++
		totalOrders++
	}

	if totalOrders == 0 {
		return
	}

	// Calculate percentiles
	sort.Ints(latencies)
	p50 := float64(percentile(latencies, 0.50))
	p90 := float64(percentile(latencies, 0.90))
	p99 := float64(percentile(latencies, 0.99))

	// Calculate throughput (orders per second)
	throughputTps := float64(totalOrders) / 60.0

	// Calculate correctness rate
	validCount := statusCounts["ACK"] + statusCounts["FILL"]
	correctnessRate := 100.0 * float64(validCount) / float64(totalOrders)

	// Calculate composite score
	compositeScore := (correctnessRate / 100.0) * throughputTps / (1.0 + float64(p99)/1000.0)

	// Update leaderboard_metrics table
	updateQuery := `
		INSERT INTO leaderboard_metrics (submission_id, orders_processed, p50_latency_ms, p90_latency_ms, p99_latency_ms, throughput_tps, correctness_rate, composite_score, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (submission_id) DO UPDATE SET
			orders_processed = EXCLUDED.orders_processed,
			p50_latency_ms = EXCLUDED.p50_latency_ms,
			p90_latency_ms = EXCLUDED.p90_latency_ms,
			p99_latency_ms = EXCLUDED.p99_latency_ms,
			throughput_tps = EXCLUDED.throughput_tps,
			correctness_rate = EXCLUDED.correctness_rate,
			composite_score = EXCLUDED.composite_score,
			updated_at = NOW()
	`

	if _, err := s.db.ExecContext(ctx, updateQuery,
		submissionID,
		totalOrders,
		p50, p90, p99,
		throughputTps,
		correctnessRate,
		compositeScore,
	); err != nil {
		log.Printf("⚠️  Failed to update leaderboard for %s: %v", submissionID, err)
	}
}

// percentile: Calculate percentile from sorted array
func percentile(values []int, p float64) int {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}

	index := int(float64(len(values)-1) * p)
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

// leaderboardHandler: GET /leaderboard - Return current rankings
func (s *TelemetryServer) leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			ROW_NUMBER() OVER (ORDER BY composite_score DESC),
			submission_id,
			team_name,
			orders_processed,
			p50_latency_ms,
			p90_latency_ms,
			p99_latency_ms,
			throughput_tps,
			correctness_rate,
			composite_score
		FROM leaderboard_metrics
		WHERE status = 'running'
		ORDER BY composite_score DESC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.Rank, &e.SubmissionID, &e.TeamName, &e.OrdersProcessed,
			&e.P50LatencyMs, &e.P90LatencyMs, &e.P99LatencyMs, &e.ThroughputTps, &e.CorrectnessRate, &e.CompositeScore); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// healthHandler: /health endpoint
func (s *TelemetryServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:    "unhealthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Service:   "telemetry-ingester",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Service:   "telemetry-ingester",
	})
}

// indexHandler: GET / - Info endpoint
func (s *TelemetryServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service": "IICPC Telemetry Ingester",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"POST /metrics":    "Record order metrics from bot",
			"GET /leaderboard": "Get current leaderboard rankings",
			"GET /health":      "Health check",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
