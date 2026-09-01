package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

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
	SubmissionID string `json:"submission_id"`
	ContainerID  string `json:"container_id"`
	EndpointURL  string `json:"endpoint_url"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
}

func main() {
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		dbURL = "postgres://iicpc_user:iicpc_password_dev@localhost/iicpc_metrics?sslmode=disable"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Printf("⚠ DB not available (running without DB): %v", err)
	} else {
		log.Println("✓ Connected to PostgreSQL")
	}

	server := &Server{db: db, port: port}

	http.HandleFunc("/health", server.healthHandler)
	http.HandleFunc("/submit", server.submitHandler)
	http.HandleFunc("/status", server.statusHandler)
	http.HandleFunc("/", server.indexHandler)

	log.Printf("🚀 Submission Handler starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Service:   "submission-handler",
	})
}

func (s *Server) submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Language == "" || req.Code == "" || req.TeamName == "" {
		http.Error(w, "Missing required fields: language, code, team_name", http.StatusBadRequest)
		return
	}

	supported := map[string]bool{"go": true, "rust": true, "cpp": true, "c++": true, "python": true}
	lang := strings.ToLower(req.Language)
	if !supported[lang] {
		http.Error(w, "Unsupported language", http.StatusBadRequest)
		return
	}

	if len(req.Code) > 100*1024 {
		http.Error(w, "Code exceeds 100KB limit", http.StatusBadRequest)
		return
	}

	submissionID := fmt.Sprintf("sub_%d_%s", time.Now().UnixMilli(), generateShortID())

	// write code to temp dir
	tmpDir := fmt.Sprintf("/tmp/%s", submissionID)
	os.MkdirAll(tmpDir, 0755)

	codeFile := codeFilename(lang)
	os.WriteFile(fmt.Sprintf("%s/%s", tmpDir, codeFile), []byte(req.Code), 0644)
	os.WriteFile(fmt.Sprintf("%s/Dockerfile", tmpDir), []byte(generateDockerfile(lang, codeFile)), 0644)

	// build docker image
	imageName := fmt.Sprintf("submission-%s", submissionID)
	buildCmd := exec.Command("docker", "build", "-t", imageName, tmpDir)
	buildOut, buildErr := buildCmd.CombinedOutput()
	if buildErr != nil {
		log.Printf("❌ Docker build failed: %v\n%s", buildErr, buildOut)
		http.Error(w, "Container build failed", http.StatusInternalServerError)
		return
	}
	log.Printf("✓ Docker image built: %s", imageName)

	// pick a port and run container
	hostPort := pickPort()
	runCmd := exec.Command("docker", "run", "-d",
		"--name", submissionID,
		"-p", fmt.Sprintf("%d:8090", hostPort),
		"--memory=256m", "--cpus=1",
		"--network=none", // no internet access
		imageName,
	)
	runOut, runErr := runCmd.CombinedOutput()
	containerID := strings.TrimSpace(string(runOut))
	if runErr != nil {
		log.Printf("❌ Docker run failed: %v\n%s", runErr, runOut)
		http.Error(w, "Container start failed", http.StatusInternalServerError)
		return
	}
	log.Printf("✓ Container running: %s on port %d", containerID[:12], hostPort)

	endpointURL := fmt.Sprintf("http://localhost:%d", hostPort)

	// store in DB (best effort)
	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.db.ExecContext(ctx,
			`INSERT INTO submissions (submission_id, team_name, language, status, container_id, endpoint_url)
			 VALUES ($1,$2,$3,'running',$4,$5)
			 ON CONFLICT DO NOTHING`,
			submissionID, req.TeamName, lang, containerID[:12], endpointURL,
		)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SubmissionResponse{
		SubmissionID: submissionID,
		ContainerID:  containerID[:12],
		EndpointURL:  endpointURL,
		Status:       "running",
		Message:      "Submission containerized and running",
	})
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	submissionID := r.URL.Query().Get("submission_id")
	if submissionID == "" {
		http.Error(w, "Missing submission_id parameter", http.StatusBadRequest)
		return
	}

	// check docker directly
	out, err := exec.Command("docker", "inspect", "--format={{.State.Status}}", submissionID).Output()
	status := "unknown"
	if err == nil {
		status = strings.TrimSpace(string(out))
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submission_id": submissionID,
		"status":        status,
	})
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "IICPC Submission Handler",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"POST /submit": "Upload and containerize submission",
			"GET /status":  "Check submission status (?submission_id=)",
			"GET /health":  "Health check",
		},
	})
}

func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func codeFilename(lang string) string {
	switch lang {
	case "go":
		return "main.go"
	case "rust":
		return "main.rs"
	case "cpp", "c++":
		return "main.cpp"
	case "python":
		return "main.py"
	}
	return "main.go"
}

func generateDockerfile(lang, codeFile string) string {
	switch lang {
	case "go":
		return `FROM golang:1.21-alpine
WORKDIR /app
COPY main.go .
RUN go mod init submission && go build -o server main.go
EXPOSE 8090
CMD ["./server"]`
	case "rust":
		return `FROM rust:1.75-alpine
WORKDIR /app
COPY main.rs .
RUN rustc main.rs -o server
EXPOSE 8090
CMD ["./server"]`
	case "cpp", "c++":
		return `FROM gcc:13
WORKDIR /app
COPY main.cpp .
RUN g++ -O2 -o server main.cpp
EXPOSE 8090
CMD ["./server"]`
	case "python":
		return `FROM python:3.11-alpine
WORKDIR /app
COPY main.py .
EXPOSE 8090
CMD ["python", "main.py"]`
	}
	return ""
}

// simple port picker starting from 9000
var portCounter = 9000

func pickPort() int {
	portCounter++
	return portCounter
}
