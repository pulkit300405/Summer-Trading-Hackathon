package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateSubmission checks submission for security and correctness
func ValidateSubmission(req *SubmissionRequest) error {
	// Check language
	validLanguages := map[string]bool{"go": true, "rust": true, "cpp": true, "c++": true}
	if !validLanguages[strings.ToLower(req.Language)] {
		return fmt.Errorf("invalid language: %s (must be Go, Rust, or C++)", req.Language)
	}

	// Check team name
	if req.TeamName == "" {
		return fmt.Errorf("team_name is required")
	}
	if len(req.TeamName) > 100 {
		return fmt.Errorf("team_name too long (max 100 chars)")
	}

	// Check code
	if req.Code == "" {
		return fmt.Errorf("code is required")
	}
	if len(req.Code) > 1000000 { // 1MB limit
		return fmt.Errorf("code too large (max 1MB)")
	}

	// Check for dangerous patterns
	if err := detectMaliciousCode(req.Code); err != nil {
		return err
	}

	return nil
}

// detectMaliciousCode checks for potentially dangerous code patterns
func detectMaliciousCode(code string) error {
	dangerousPatterns := []string{
		`(?i)os\.Remove`,       // File deletion
		`(?i)exec\.Command`,    // Shell execution
		`(?i)system\(`,         // System calls
		`(?i)fork\(\)`,         // Process forking
		`(?i)rm\s+-rf`,         // Dangerous shell commands
		`(?i)shutdown`,         // System shutdown
		`(?i)/bin/bash`,        // Shell invocation
	}

	for _, pattern := range dangerousPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(code) {
			return fmt.Errorf("code contains dangerous pattern: %s", pattern)
		}
	}

	return nil
}

// ValidateMetricEvent checks incoming metric events
func ValidateMetricEvent(event *MetricEvent) error {
	if event.SubmissionID == "" {
		return fmt.Errorf("submission_id is required")
	}
	if event.LatencyMs < 0 {
		return fmt.Errorf("latency_ms must be non-negative")
	}
	if event.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	return nil
}
