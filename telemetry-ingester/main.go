package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type MetricEvent struct {
	SubmissionID string  `json:"submission_id"`
	TeamName     string  `json:"team_name"`
	LatencyMs    float64 `json:"latency_ms"`
	Success      bool    `json:"success"`
	OrderType    string  `json:"order_type"`
	Timestamp    int64   `json:"timestamp"`
}

type TeamStats struct {
	TeamName     string  `json:"team_name"`
	SubmissionID string  `json:"submission_id"`
	TotalOrders  int     `json:"total_orders"`
	SuccessCount int     `json:"success_count"`
	P50          float64 `json:"p50_ms"`
	P90          float64 `json:"p90_ms"`
	P99          float64 `json:"p99_ms"`
	TPS          float64 `json:"tps"`
	UpdatedAt    string  `json:"updated_at"`
}

type Ingester struct {
	mu       sync.RWMutex
	metrics  map[string][]float64 // submissionID -> latencies
	meta     map[string]*TeamStats
	startTime map[string]time.Time
}

func NewIngester() *Ingester {
	return &Ingester{
		metrics:   make(map[string][]float64),
		meta:      make(map[string]*TeamStats),
		startTime: make(map[string]time.Time),
	}
}

func (ing *Ingester) ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var event MetricEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if event.SubmissionID == "" || event.LatencyMs <= 0 {
		http.Error(w, "Missing submission_id or latency_ms", http.StatusBadRequest)
		return
	}

	ing.mu.Lock()
	defer ing.mu.Unlock()

	// init if first time
	if _, ok := ing.metrics[event.SubmissionID]; !ok {
		ing.metrics[event.SubmissionID] = []float64{}
		ing.startTime[event.SubmissionID] = time.Now()
		ing.meta[event.SubmissionID] = &TeamStats{
			TeamName:     event.TeamName,
			SubmissionID: event.SubmissionID,
		}
	}

	ing.metrics[event.SubmissionID] = append(ing.metrics[event.SubmissionID], event.LatencyMs)

	stats := ing.meta[event.SubmissionID]
	stats.TotalOrders++
	if event.Success {
		stats.SuccessCount++
	}

	// recalculate percentiles
	lats := make([]float64, len(ing.metrics[event.SubmissionID]))
	copy(lats, ing.metrics[event.SubmissionID])
	sort.Float64s(lats)

	stats.P50 = percentile(lats, 50)
	stats.P90 = percentile(lats, 90)
	stats.P99 = percentile(lats, 99)

	elapsed := time.Since(ing.startTime[event.SubmissionID]).Seconds()
	if elapsed > 0 {
		stats.TPS = float64(stats.TotalOrders) / elapsed
	}
	stats.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

func (ing *Ingester) leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	ing.mu.RLock()
	defer ing.mu.RUnlock()

	results := make([]*TeamStats, 0, len(ing.meta))
	for _, s := range ing.meta {
		results = append(results, s)
	}

	// sort by p99 ascending (lower latency = better)
	sort.Slice(results, func(i, j int) bool {
		return results[i].P99 < results[j].P99
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"leaderboard": results,
		"total_teams": len(results),
		"updated_at":  time.Now().UTC().Format(time.RFC3339),
	})
}

func (ing *Ingester) statsHandler(w http.ResponseWriter, r *http.Request) {
	subID := r.URL.Query().Get("submission_id")
	ing.mu.RLock()
	defer ing.mu.RUnlock()

	if subID == "" {
		http.Error(w, "Missing submission_id", http.StatusBadRequest)
		return
	}

	stats, ok := ing.meta[subID]
	if !ok {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (ing *Ingester) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "telemetry-ingester",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p / 100) * float64(len(sorted)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sorted[lo]
	}
	return sorted[lo]*(float64(hi)-idx) + sorted[hi]*(idx-float64(lo))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	ing := NewIngester()

	http.HandleFunc("/ingest", ing.ingestHandler)
	http.HandleFunc("/leaderboard", ing.leaderboardHandler)
	http.HandleFunc("/stats", ing.statsHandler)
	http.HandleFunc("/health", ing.healthHandler)

	log.Printf("🚀 Telemetry Ingester starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
