// One-time migration script: reads historical sessions from the local cc-live
// SQLite database and POSTs them as heartbeats to the claug API.
//
// Usage: go run . [--dry-run]
//
// Delete this script after successful backfill.
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type claugAuthConfig struct {
	APIKey   string `json:"api_key"`
	Endpoint string `json:"endpoint"`
}

type sessionMetrics struct {
	SessionID            string         `json:"session_id"`
	TotalTokens          int64          `json:"total_tokens"`
	InputTokens          int64          `json:"input_tokens"`
	CacheReadInputTokens int64          `json:"cache_read_input_tokens"`
	OutputTokens         int64          `json:"output_tokens"`
	ToolCalls            int            `json:"tool_calls"`
	ToolCounts           map[string]int `json:"tool_counts,omitempty"`
	UserPrompts          int            `json:"user_prompts"`
	ActiveTime           int            `json:"active_time_seconds"`
	LastPrompt           string         `json:"last_prompt,omitempty"`
	Project              string         `json:"project"`
	Model                string         `json:"model"`
	Summary              string         `json:"summary,omitempty"`
	PrivacyLevel         string         `json:"privacy_level"`
}

type heartbeatPayload struct {
	Sessions []sessionMetrics `json:"sessions"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	dryRun := false
	for _, arg := range os.Args[1:] {
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	cfg := loadConfig()
	sessions := readSQLiteSessions()

	log.Printf("found %d sessions in SQLite", len(sessions))

	if dryRun {
		log.Printf("dry run — not sending to API")
		for _, s := range sessions {
			log.Printf("  %s: project=%s tokens=%d tools=%d", s.SessionID, s.Project, s.TotalTokens, s.ToolCalls)
		}
		return
	}

	// Send in batches of 10
	batchSize := 10
	sent := 0
	failed := 0
	for i := 0; i < len(sessions); i += batchSize {
		end := i + batchSize
		if end > len(sessions) {
			end = len(sessions)
		}
		batch := sessions[i:end]

		payload := heartbeatPayload{Sessions: batch}
		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("ERROR marshaling batch %d-%d: %v", i, end, err)
			failed += len(batch)
			continue
		}

		req, err := http.NewRequest("POST", cfg.Endpoint+"/api/sessions/heartbeat", bytes.NewReader(body))
		if err != nil {
			log.Printf("ERROR creating request: %v", err)
			failed += len(batch)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("ERROR sending batch %d-%d: %v", i, end, err)
			failed += len(batch)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			log.Printf("ERROR batch %d-%d: API returned status %d", i, end, resp.StatusCode)
			failed += len(batch)
			continue
		}

		sent += len(batch)
		log.Printf("sent batch %d-%d (%d/%d)", i, end, sent, len(sessions))
	}

	log.Printf("backfill complete: %d sent, %d failed out of %d total", sent, failed, len(sessions))
}

func loadConfig() claugAuthConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("getting home dir: %v", err)
	}

	authFile := filepath.Join(home, ".config", "claug", "auth.json")
	data, err := os.ReadFile(authFile)
	if err != nil {
		log.Fatalf("reading %s: %v\nRun 'claug login' to authenticate.", authFile, err)
	}

	var cfg claugAuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("parsing %s: %v", authFile, err)
	}

	if cfg.APIKey == "" {
		log.Fatalf("no api_key found in %s", authFile)
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://claug.ai"
	}

	return cfg
}

func readSQLiteSessions() []sessionMetrics {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("getting home dir: %v", err)
	}

	dbPath := filepath.Join(home, ".cc-live", "state.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("opening SQLite: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT session_id, project, model, summary,
		num_user_prompts, num_tool_calls, total_input_tokens, total_cache_read_input_tokens,
		total_output_tokens, total_tokens, active_time_seconds, cc_version, sensitive,
		tool_counts_json
		FROM session_stats
		WHERE total_tokens > 0
		ORDER BY date DESC`)
	if err != nil {
		log.Fatalf("querying session_stats: %v", err)
	}
	defer rows.Close()

	var sessions []sessionMetrics
	for rows.Next() {
		var (
			sessionID      string
			project        string
			model          string
			summary        string
			userPrompts    int
			toolCalls      int
			inputTokens    int64
			cacheReadTokens int64
			outputTokens   int64
			totalTokens    int64
			activeTime     int
			ccVersion      string
			sensitive      int
			toolCountsJSON string
		)

		if err := rows.Scan(&sessionID, &project, &model, &summary,
			&userPrompts, &toolCalls, &inputTokens, &cacheReadTokens,
			&outputTokens, &totalTokens, &activeTime, &ccVersion, &sensitive,
			&toolCountsJSON); err != nil {
			log.Printf("scanning row: %v", err)
			continue
		}

		privacyLevel := "full_context"
		if sensitive == 1 {
			privacyLevel = "metrics_only"
		}

		// Parse tool counts
		var toolCounts map[string]int
		if toolCountsJSON != "" && toolCountsJSON != "{}" {
			if err := json.Unmarshal([]byte(toolCountsJSON), &toolCounts); err != nil {
				toolCounts = nil
			}
		}

		// Clean project name from cwd if needed
		if project == "" {
			// project might not be stored; that's ok
		}

		_ = ccVersion // not used in heartbeat payload
		_ = strings.TrimSpace(summary)

		sessions = append(sessions, sessionMetrics{
			SessionID:            sessionID,
			TotalTokens:          totalTokens,
			InputTokens:          inputTokens,
			CacheReadInputTokens: cacheReadTokens,
			OutputTokens:         outputTokens,
			ToolCalls:            toolCalls,
			ToolCounts:           toolCounts,
			UserPrompts:          userPrompts,
			ActiveTime:           activeTime,
			Project:              project,
			Model:                model,
			Summary:              summary,
			PrivacyLevel:         privacyLevel,
		})
	}

	return sessions
}

func cleanToolName(name string) string {
	parts := strings.Split(name, "__")
	if len(parts) >= 3 && parts[0] == "mcp" {
		providerParts := strings.Split(parts[1], "_")
		service := providerParts[len(providerParts)-1]
		return service + ": " + strings.Join(parts[2:], "__")
	}
	return name
}

func formatTokens(n int64) string {
	if n >= 10_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	s := fmt.Sprintf("%d", n)
	if n < 0 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
