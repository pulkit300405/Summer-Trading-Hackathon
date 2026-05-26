package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type Server struct {
	db   *sql.DB
	port string
}

type SubmissionRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	TeamName string `json:"team_name"`
}

type SubmissionResponse struct {
	SubmissionID  string `json:"submission_id"`
	ContainerID   string `json:"container_id"`
	EndpointURL   string `json:"endpoint_url"`
	Status        string `json:"status"`
	Message       string `json:"message"`
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
		port = "8080"
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

	server := &Server{
		db:   db,
		port: port,
	}

	// Register HTTP handlers
	http.HandleFunc("/health", server.healthHandler)
	http.HandleFunc("/submit", server.submitHandler)
	http.HandleFunc("/status", server.statusHandler)
	http.HandleFunc("/", server.indexHandler)

	// Start server
	addr := ":" + port
	log.Printf("🚀 Submission Handler starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// healthHandler: /health endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check database connectivity
	if err := s.db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:    "unhealthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Service:   "submission-handler",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Service:   "submission-handler",
	})
}

// submitHandler: POST /submit - Accept and containerize submissions
func (s *Server) submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req SubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Language == "" || req.Code == "" || req.TeamName == "" {
		http.Error(w, "Missing required fields: language, code, team_name", http.StatusBadRequest)
		return
	}

	// Validate language
	supportedLanguages := map[string]bool{
		"go":     true,
		"rust":   true,
		"cpp":    true,
		"c++":    true,
		"python": true,
	}
	if !supportedLanguages[strings.ToLower(req.Language)] {
		http.Error(w, "Unsupported language. Supported: go, rust, cpp, python", http.StatusBadRequest)
		return
	}

	// Check code size (limit to 100KB)
	if len(req.Code) > 100*1024 {
		http.Error(w, "Code exceeds 100KB limit", http.StatusBadRequest)
		return
	}

	// Generate submission ID
	submissionID := fmt.Sprintf("sub_%d_%s", time.Now().UnixMilli(), generateShortID())

	// Store submission in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO submissions (submission_id, team_name, language, status)
		VALUES ($1, $2, $3, 'building')
		RETURNING submission_id
	`

	var returnedID string
	if err := s.db.QueryRowContext(ctx, query, submissionID, req.TeamName, strings.ToLower(req.Language)).Scan(&returnedID); err != nil {
		log.Printf("❌ Failed to store submission: %v", err)
		http.Error(w, "Failed to store submission", http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Submission created: %s (team: %s, language: %s)", submissionID, req.TeamName, req.Language)

	// TODO: In production, containerize the code
	// For now, return mock response
	containerID := fmt.Sprintf("container_%s", submissionID[4:])
	endpointURL := fmt.Sprintf("http://submission-handler:8100/%s", submissionID)

	// Update submission status
	updateQuery := `UPDATE submissions SET status = 'running', container_id = $1, endpoint_url = $2 WHERE submission_id = $3`
	if _, err := s.db.ExecContext(ctx, updateQuery, containerID, endpointURL, submissionID); err != nil {
		log.Printf("❌ Failed to update submission status: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SubmissionResponse{
		SubmissionID: submissionID,
		ContainerID:  containerID,
		EndpointURL:  endpointURL,
		Status:       "running",
		Message:      "Submission accepted and running",
	})
}

// statusHandler: GET /status?submission_id=<id> - Check submission status
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	submissionID := r.URL.Query().Get("submission_id")
	if submissionID == "" {
		http.Error(w, "Missing submission_id parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT submission_id, team_name, language, status, endpoint_url, error_message
		FROM submissions
		WHERE submission_id = $1
	`

	var id, teamName, language, status, endpoint, errMsg string
	if err := s.db.QueryRowContext(ctx, query, submissionID).Scan(&id, &teamName, &language, &status, &endpoint, &errMsg); err == sql.ErrNoRows {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submission_id": id,
		"team_name":     teamName,
		"language":      language,
		"status":        status,
		"endpoint_url":  endpoint,
		"error":         errMsg,
	})
}

// indexHandler: GET / - Info endpoint
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service": "IICPC Submission Handler",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"POST /submit":     "Upload and containerize submission",
			"GET /status":      "Check submission status (query param: submission_id)",
			"GET /health":      "Health check",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Helper: Generate short random ID
func generateShortID() string {
	b := make([]byte, 4)
	io.ReadFull(os.Urandom(4), b)
	return fmt.Sprintf("%x", b)
}
