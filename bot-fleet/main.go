package main

import (
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
	orderBook         map[string]*Order // Track outstanding orders
	orderBookLock     sync.RWMutex
	lastOrderTime     time.Time
	timeBetweenOrders time.Duration
}

type Order struct {
	OrderID   string
	Price     float64
	Quantity  int
	Type      string
	Timestamp time.Time
	Status    string // PENDING, ACKED, REJECTED, FILLED, CANCELLED
	FillPrice float64
}

type OrderRequest struct {
	OrderID     string  `json:"order_id"`
	OrderType   string  `json:"order_type"` // LIMIT, MARKET, CANCEL
	Side        string  `json:"side"`       // BUY, SELL
	Price       float64 `json:"price,omitempty"`
	Quantity    int     `json:"quantity"`
	TimeInForce string  `json:"time_in_force,omitempty"` // GTC, IOC, FOK
	ExistingID  string  `json:"existing_id,omitempty"`   // For CANCEL
	Timestamp   int64   `json:"timestamp"`
}

type MetricEvent struct {
	BotID            int     `json:"bot_id"`
	OrderID          string  `json:"order_id"`
	TimestampSent    int64   `json:"timestamp_sent"`
	TimestampAck     int64   `json:"timestamp_ack"`
	LatencyMs        int     `json:"latency_ms"`
	OrderType        string  `json:"order_type"`
	Status           string  `json:"status"`
	Price            float64 `json:"price"`
	Quantity         int     `json:"quantity"`
	CorrectnessValid bool    `json:"correctness_valid"`
}

func main() {
	debug.SetMaxThreads(500)
	// Get environment variables
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

	log.Printf("🤖 Bot Fleet Configuration:")
	log.Printf("  - Target URL: %s", targetURL)
	log.Printf("  - Telemetry URL: %s", telemetryURL)
	log.Printf("  - Number of Bots: %d", numBots)
	log.Printf("  - Total TPS: %d orders/second", totalTPS)

	fleet := &BotFleet{
		targetURL:       targetURL,
		telemetryURL:    telemetryURL,
		numBots:         numBots,
		ordersPerSecond: totalTPS,
		bots:            make(map[int]*Bot),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	// Start HTTP server for health checks
	go func() {
		http.HandleFunc("/health", fleet.healthHandler)
		http.HandleFunc("/stats", fleet.statsHandler)
		log.Printf("📊 Bot Fleet API listening on :8081")
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	// Spawn bots
	fleet.spawnBots()

	// Keep running
	select {}
}

// spawnBots: Create and start bot goroutines
func (f *BotFleet) spawnBots() {
	// Calculate orders per second per bot
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
		}

		bot.ctx, bot.cancel = context.WithCancel(context.Background())

		f.mu.Lock()
		f.bots[i] = bot
		f.mu.Unlock()

		// Start bot goroutine
		go bot.run(f)
	}

	log.Printf("✓ All %d bots spawned", f.numBots)
}

// run: Main bot loop - generate and send orders
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
			// Generate random order
			order := b.generateOrder()

			// Send order to submission
			sentTime := time.Now()
			resp, err := b.sendOrder(order)

			if err != nil {
				b.recordMetric(order, sentTime, time.Now(), "ERROR", f)
				f.sentOrders.Add(1) // count attempts too
				continue
			}

			// Record metric
			latencyMs := time.Since(sentTime).Milliseconds()
			_ = latencyMs
			b.recordMetric(order, sentTime, time.Now(), resp, f)

			f.sentOrders.Add(1)
		}
	}
}

// generateOrder: Create a realistic trading order
func (b *Bot) generateOrder() *OrderRequest {
	orderID := fmt.Sprintf("order_%d_%s", b.id, uuid.New().String()[:8])

	// 70% LIMIT, 20% MARKET, 10% CANCEL
	roll := rand.Float64()
	orderType := "LIMIT"
	if roll > 0.9 {
		orderType = "CANCEL"
	} else if roll > 0.7 {
		orderType = "MARKET"
	}

	// Check if we have outstanding orders for CANCEL
	if orderType == "CANCEL" {
		b.orderBookLock.RLock()
		if len(b.orderBook) == 0 {
			b.orderBookLock.RUnlock()
			orderType = "LIMIT" // Fall back to LIMIT if no orders
		} else {
			// Pick random outstanding order to cancel
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

	// Random side
	side := "BUY"
	if rand.Float64() > 0.5 {
		side = "SELL"
	}

	// Random price (between 99.0 and 101.0)
	price := 99.0 + rand.Float64()*2.0

	// Random quantity (1-100)
	quantity := 1 + rand.Intn(100)

	// Random time-in-force
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

// sendOrder: Send order to submission container
func (b *Bot) sendOrder(order *OrderRequest) (string, error) {
	// Simulate realistic latency (1-10ms)
	time.Sleep(time.Duration(1+rand.Intn(10)) * time.Millisecond)

	// 95% success rate simulation
	if rand.Float64() < 0.95 {
		b.ackedCount++
		return "ACK", nil
	}
	b.rejectedCount++
	return "REJECTED", nil
}

// recordMetric: Send metrics to telemetry ingester
func (b *Bot) recordMetric(order *OrderRequest, sentTime time.Time, ackTime time.Time, status string, f *BotFleet) {
	latencyMs := int(ackTime.Sub(sentTime).Milliseconds())

	metric := MetricEvent{
		BotID:         b.id,
		OrderID:       order.OrderID,
		TimestampSent: sentTime.UnixMilli(),
		TimestampAck:  ackTime.UnixMilli(),
		LatencyMs:     latencyMs,
		OrderType:     order.OrderType,
		Status:        status,
		Price:         order.Price,
		Quantity:      order.Quantity,
	}

	// Send asynchronously to avoid blocking bot
	// TODO: re-enable when telemetry is running
	_ = metric
}

// healthHandler: /health endpoint
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

// statsHandler: /stats endpoint - Return bot fleet statistics
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
