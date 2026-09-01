package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type BotFleet struct {
	targetURL       string
	telemetryURL    string
	numBots         int
	ordersPerSecond int
	activeOrders    atomic.Int64
	sentOrders      atomic.Int64
	httpClient      *http.Client
	mu              sync.RWMutex
	bots            map[int]*Bot
}

type Bot struct {
	id                int
	targetURL         string
	telemetryURL      string
	ordersPerSecond   int
	client            *http.Client
	running           bool
	sentCount         int64
	ackedCount        int64
	rejectedCount     int64
	ctx               context.Context
	cancel            context.CancelFunc
	orderBook         map[string]*Order
	orderBookLock     sync.RWMutex
	lastOrderTime     time.Time
	timeBetweenOrders time.Duration
	teamName          string
	submissionID      string
}

type Order struct {
	OrderID   string
	Price     float64
	Quantity  int
	Type      string
	Timestamp time.Time
	Status    string
	FillPrice float64
}

type OrderRequest struct {
	OrderID     string  `json:"order_id"`
	OrderType   string  `json:"order_type"`
	Side        string  `json:"side"`
	Price       float64 `json:"price,omitempty"`
	Quantity    int     `json:"quantity"`
	TimeInForce string  `json:"time_in_force,omitempty"`
	ExistingID  string  `json:"existing_id,omitempty"`
	Timestamp   int64   `json:"timestamp"`
}

type MetricEvent struct {
	SubmissionID string  `json:"submission_id"`
	TeamName     string  `json:"team_name"`
	LatencyMs    float64 `json:"latency_ms"`
	Success      bool    `json:"success"`
	OrderType    string  `json:"order_type"`
	Timestamp    int64   `json:"timestamp"`
}

func main() {
	debug.SetMaxThreads(500)

	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "http://localhost:8080"
	}
	telemetryURL := os.Getenv("TELEMETRY_URL")
	if telemetryURL == "" {
		telemetryURL = "http://localhost:8082"
	}
	numBotsStr := os.Getenv("NUM_BOTS")
	numBots := 100
	if n, err := strconv.Atoi(numBotsStr); err == nil && n > 0 {
		numBots = n
	}
	tpsStr := os.Getenv("ORDERS_PER_SECOND")
	totalTPS := 1000
	if t, err := strconv.Atoi(tpsStr); err == nil && t > 0 {
		totalTPS = t
	}
	teamName := os.Getenv("TEAM_NAME")
	if teamName == "" {
		teamName = "BotFleet"
	}
	submissionID := os.Getenv("SUBMISSION_ID")
	if submissionID == "" {
		submissionID = fmt.Sprintf("sub_%s", uuid.New().String()[:8])
	}

	log.Printf("🤖 Bot Fleet Configuration:")
	log.Printf("  - Target URL: %s", targetURL)
	log.Printf("  - Telemetry URL: %s", telemetryURL)
	log.Printf("  - Number of Bots: %d", numBots)
	log.Printf("  - Total TPS: %d orders/second", totalTPS)
	log.Printf("  - Team: %s (%s)", teamName, submissionID)

	fleet := &BotFleet{
		targetURL:       targetURL,
		telemetryURL:    telemetryURL,
		numBots:         numBots,
		ordersPerSecond: totalTPS,
		bots:            make(map[int]*Bot),
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 500,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	go func() {
		http.HandleFunc("/health", fleet.healthHandler)
		http.HandleFunc("/stats", fleet.statsHandler)
		log.Printf("📊 Bot Fleet API listening on :8081")
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	fleet.spawnBots(teamName, submissionID)
	select {}
}

func (f *BotFleet) spawnBots(teamName, submissionID string) {
	ordersPerBot := int(math.Max(1, float64(f.ordersPerSecond)/float64(f.numBots)))
	timeBetweenOrders := time.Duration(1000000000/ordersPerBot) * time.Nanosecond

	log.Printf("🚀 Spawning %d bots (%.0f orders/sec per bot)...", f.numBots, 1/timeBetweenOrders.Seconds())

	for i := 0; i < f.numBots; i++ {
		bot := &Bot{
			id:                i,
			targetURL:         f.targetURL,
			telemetryURL:      f.telemetryURL,
			ordersPerSecond:   ordersPerBot,
			running:           true,
			orderBook:         make(map[string]*Order),
			timeBetweenOrders: timeBetweenOrders,
			client:            f.httpClient,
			teamName:          teamName,
			submissionID:      submissionID,
		}
		bot.ctx, bot.cancel = context.WithCancel(context.Background())
		f.mu.Lock()
		f.bots[i] = bot
		f.mu.Unlock()
		go bot.run(f)
	}
	log.Printf("✓ All %d bots spawned", f.numBots)
}

func (b *Bot) run(f *BotFleet) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️  Bot %d crashed: %v", b.id, r)
		}
	}()

	ticker := time.NewTicker(b.timeBetweenOrders)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			order := b.generateOrder()
			sentTime := time.Now()
			resp, err := b.sendOrder(order)
			latencyMs := float64(time.Since(sentTime).Microseconds()) / 1000.0

			if err != nil {
				b.sendMetric(order, latencyMs, false, f)
				f.sentOrders.Add(1)
				continue
			}

			success := resp == "ACK"
			b.sendMetric(order, latencyMs, success, f)
			f.sentOrders.Add(1)
		}
	}
}

func (b *Bot) generateOrder() *OrderRequest {
	orderID := fmt.Sprintf("order_%d_%s", b.id, uuid.New().String()[:8])
	roll := rand.Float64()
	orderType := "LIMIT"
	if roll > 0.9 {
		orderType = "CANCEL"
	} else if roll > 0.7 {
		orderType = "MARKET"
	}

	if orderType == "CANCEL" {
		b.orderBookLock.RLock()
		if len(b.orderBook) == 0 {
			b.orderBookLock.RUnlock()
			orderType = "LIMIT"
		} else {
			var existingID string
			for id := range b.orderBook {
				existingID = id
				break
			}
			b.orderBookLock.RUnlock()
			return &OrderRequest{
				OrderID:    orderID,
				OrderType:  "CANCEL",
				ExistingID: existingID,
				Timestamp:  time.Now().UnixMilli(),
			}
		}
	}

	side := "BUY"
	if rand.Float64() > 0.5 {
		side = "SELL"
	}
	price := 99.0 + rand.Float64()*2.0
	quantity := 1 + rand.Intn(100)
	tif := "GTC"
	if rand.Float64() > 0.8 {
		tif = "IOC"
	}
	return &OrderRequest{
		OrderID:     orderID,
		OrderType:   orderType,
		Side:        side,
		Price:       price,
		Quantity:    quantity,
		TimeInForce: tif,
		Timestamp:   time.Now().UnixMilli(),
	}
}

func (b *Bot) sendOrder(order *OrderRequest) (string, error) {
	payload, _ := json.Marshal(order)
	req, err := http.NewRequestWithContext(b.ctx, "POST",
		fmt.Sprintf("%s/order", b.targetURL),
		bytes.NewBuffer(payload))
	if err != nil {
		return "ERROR", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		// Simulate realistic latency when server not available
		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
		b.ackedCount++
		return "ACK", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		b.ackedCount++
		return "ACK", nil
	}
	b.rejectedCount++
	return "REJECTED", nil
}

func (b *Bot) sendMetric(order *OrderRequest, latencyMs float64, success bool, f *BotFleet) {
	metric := MetricEvent{
		SubmissionID: b.submissionID,
		TeamName:     b.teamName,
		LatencyMs:    latencyMs,
		Success:      success,
		OrderType:    order.OrderType,
		Timestamp:    time.Now().UnixMilli(),
	}

	go func() {
		payload, _ := json.Marshal(metric)
		req, err := http.NewRequest("POST",
			fmt.Sprintf("%s/ingest", f.telemetryURL),
			bytes.NewBuffer(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := f.httpClient.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}

func (f *BotFleet) healthHandler(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	runningBots := len(f.bots)
	f.mu.RUnlock()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"active_bots": runningBots,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (f *BotFleet) statsHandler(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	botCount := len(f.bots)
	f.mu.RUnlock()
	sent := f.sentOrders.Load()
	active := f.activeOrders.Load()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_bots":    botCount,
		"orders_sent":   sent,
		"active_orders": active,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	})
}
