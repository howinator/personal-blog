package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// claugAuthCredentials matches one entry in ~/.config/claug/auth.json (keyed by env name).
type claugAuthCredentials struct {
	Token  string `json:"token,omitempty"`
	APIKey string `json:"api_key"`
}

// claugEnvConfig matches one entry in config.yaml's envs map.
type claugEnvConfig struct {
	Endpoint string `yaml:"endpoint"`
}

// claugConfig matches ~/.config/claug/config.yaml.
type claugConfig struct {
	Envs   map[string]claugEnvConfig `yaml:"envs"`
	Active []string                  `yaml:"active"`
}

// resolvedConfig is the final API key + endpoint for a single environment.
type resolvedConfig struct {
	APIKey   string
	Endpoint string
}

// claugSessionStats matches the JSON returned by GET /api/sessions.
type claugSessionStats struct {
	ID                        string            `json:"id"`
	SessionID                 string            `json:"session_id"`
	UserID                    string            `json:"user_id"`
	Provider                  string            `json:"provider"`
	Project                   string            `json:"project"`
	Model                     string            `json:"model"`
	StartedAt                 *time.Time        `json:"started_at"`
	Summary                   string            `json:"summary"`
	LastPrompt                string            `json:"last_prompt"`
	NumUserPrompts            int               `json:"num_user_prompts"`
	NumToolCalls              int               `json:"num_tool_calls"`
	TotalInputTokens          int64             `json:"total_input_tokens"`
	TotalCacheReadInputTokens int64             `json:"total_cache_read_input_tokens"`
	TotalOutputTokens         int64             `json:"total_output_tokens"`
	TotalTokens               int64             `json:"total_tokens"`
	ActiveTimeSeconds         int               `json:"active_time_seconds"`
	ProviderVersion           string            `json:"provider_version"`
	PrivacyLevel              string            `json:"privacy_level"`
	ToolCounts                map[string]int    `json:"tool_counts"`
	UpdatedAt                 time.Time         `json:"updated_at"`
}

type sessionsResponse struct {
	Sessions []claugSessionStats `json:"sessions"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PerPage  int                 `json:"per_page"`
}

// Export types — match the exact JSON schema expected by Hugo's cc-sessions shortcode.
type sessionExport struct {
	SessionID                   string `json:"session_id"`
	Date                        string `json:"date"`
	DateDisplay                 string `json:"date_display"`
	Summary                     string `json:"summary"`
	Project                     string `json:"project"`
	Cwd                         string `json:"cwd"`
	NumUserPrompts              int    `json:"num_user_prompts"`
	NumToolCalls                int    `json:"num_tool_calls"`
	TotalInputTokens            int64  `json:"total_input_tokens"`
	TotalCacheReadInputTokens   int64  `json:"total_cache_read_input_tokens"`
	TotalCacheReadTokensDisplay string `json:"total_cache_read_tokens_display"`
	TotalOutputTokens           int64  `json:"total_output_tokens"`
	TotalTokens                 int64  `json:"total_tokens"`
	TotalTokensDisplay          string `json:"total_tokens_display"`
	TotalTokensDisplayShort     string `json:"total_tokens_display_short"`
	ActiveTimeSeconds           int    `json:"active_time_seconds"`
	ActiveTimeDisplay           string `json:"active_time_display"`
	CcVersion                   string `json:"cc_version"`
}

type toolEntry struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Display string `json:"display"`
}

type totalsExport struct {
	SessionCount                int         `json:"session_count"`
	TotalTokens                 int64       `json:"total_tokens"`
	TotalTokensDisplay          string      `json:"total_tokens_display"`
	TotalTokensDisplayShort     string      `json:"total_tokens_display_short"`
	TotalInputTokens            int64       `json:"total_input_tokens"`
	TotalInputTokensDisplay     string      `json:"total_input_tokens_display"`
	TotalCacheReadInputTokens   int64       `json:"total_cache_read_input_tokens"`
	TotalCacheReadTokensDisplay string      `json:"total_cache_read_tokens_display"`
	TotalOutputTokens           int64       `json:"total_output_tokens"`
	TotalOutputTokensDisplay    string      `json:"total_output_tokens_display"`
	TotalToolCalls              int         `json:"total_tool_calls"`
	TotalActiveTimeSeconds      int         `json:"total_active_time_seconds"`
	TotalActiveTimeDisplay      string      `json:"total_active_time_display"`
	TopTools                    []toolEntry `json:"top_tools"`
}

type dataExport struct {
	Sessions []sessionExport `json:"sessions"`
	Totals   totalsExport    `json:"totals"`
}

const (
	defaultEndpoint = "https://api.claug.ai"
	perPage         = 100
	// Only export sessions from this date forward (matches cc-live behavior)
	fromDate = "2026-02-07T00:00:00Z"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg := loadResolvedConfig()
	sessions := fetchAllSessions(cfg)

	log.Printf("fetched %d sessions from claug API", len(sessions))

	var exports []sessionExport
	var totalTokens, totalInputTokens, totalCacheReadTokens, totalOutputTokens int64
	var totalToolCalls, totalActiveTime int
	var allToolCounts []map[string]int

	for _, s := range sessions {
		// Skip empty sessions
		if s.TotalTokens == 0 {
			continue
		}

		e := sessionExport{
			SessionID:                   s.SessionID,
			Summary:                     s.Summary,
			Project:                     s.Project,
			Cwd:                         "",
			NumUserPrompts:              s.NumUserPrompts,
			NumToolCalls:                s.NumToolCalls,
			TotalInputTokens:            s.TotalInputTokens,
			TotalCacheReadInputTokens:   s.TotalCacheReadInputTokens,
			TotalCacheReadTokensDisplay: formatTokens(s.TotalCacheReadInputTokens),
			TotalOutputTokens:           s.TotalOutputTokens,
			TotalTokens:                 s.TotalTokens,
			TotalTokensDisplay:          formatTokens(s.TotalTokens),
			TotalTokensDisplayShort:     formatTokensShort(s.TotalTokens),
			ActiveTimeSeconds:           s.ActiveTimeSeconds,
			ActiveTimeDisplay:           formatTime(s.ActiveTimeSeconds),
			CcVersion:                   s.ProviderVersion,
		}

		if s.StartedAt != nil {
			e.Date = s.StartedAt.Format(time.RFC3339)
			e.DateDisplay = formatDate(s.StartedAt.Format(time.RFC3339))
		}

		exports = append(exports, e)

		totalTokens += s.TotalTokens
		totalInputTokens += s.TotalInputTokens
		totalCacheReadTokens += s.TotalCacheReadInputTokens
		totalOutputTokens += s.TotalOutputTokens
		totalToolCalls += s.NumToolCalls
		totalActiveTime += s.ActiveTimeSeconds

		if len(s.ToolCounts) > 0 {
			allToolCounts = append(allToolCounts, s.ToolCounts)
		}
	}

	// Sort sessions by date descending (newest first)
	sort.Slice(exports, func(i, j int) bool {
		return exports[i].Date > exports[j].Date
	})

	data := dataExport{
		Sessions: exports,
		Totals: totalsExport{
			SessionCount:                len(exports),
			TotalTokens:                 totalTokens,
			TotalTokensDisplay:          formatTokens(totalTokens),
			TotalTokensDisplayShort:     formatTokensShort(totalTokens),
			TotalInputTokens:            totalInputTokens,
			TotalInputTokensDisplay:     formatTokens(totalInputTokens),
			TotalCacheReadInputTokens:   totalCacheReadTokens,
			TotalCacheReadTokensDisplay: formatTokens(totalCacheReadTokens),
			TotalOutputTokens:           totalOutputTokens,
			TotalOutputTokensDisplay:    formatTokens(totalOutputTokens),
			TotalToolCalls:              totalToolCalls,
			TotalActiveTimeSeconds:      totalActiveTime,
			TotalActiveTimeDisplay:      formatTime(totalActiveTime),
			TopTools:                    topTools(allToolCounts, 5),
		},
	}

	writeExport(data)
}

func loadResolvedConfig() resolvedConfig {
	configDir := os.Getenv("CLAUG_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("getting home dir: %v", err)
		}
		configDir = filepath.Join(home, ".config", "claug")
	}

	// Determine which environment to use.
	env := os.Getenv("CLAUG_ENV")
	if env == "" {
		env = "prod" // default
	}

	// Load config.yaml to resolve the endpoint and optionally override env from active list.
	configFile := filepath.Join(configDir, "config.yaml")
	endpoint := defaultEndpoint
	if cfgData, err := os.ReadFile(configFile); err == nil {
		var cfg claugConfig
		if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
			log.Fatalf("parsing %s: %v", configFile, err)
		}
		if envCfg, ok := cfg.Envs[env]; ok && envCfg.Endpoint != "" {
			endpoint = envCfg.Endpoint
		}
	}

	// Load auth.json (keyed by env name).
	authFile := filepath.Join(configDir, "auth.json")
	authData, err := os.ReadFile(authFile)
	if err != nil {
		log.Fatalf("reading %s: %v\nRun 'claug login' to authenticate.", authFile, err)
	}

	var allCreds map[string]*claugAuthCredentials
	if err := json.Unmarshal(authData, &allCreds); err != nil {
		log.Fatalf("parsing %s: %v", authFile, err)
	}

	creds, ok := allCreds[env]
	if !ok || creds.APIKey == "" {
		log.Fatalf("no api_key found for env %q in %s. Run 'claug login' to authenticate.", env, authFile)
	}

	return resolvedConfig{
		APIKey:   creds.APIKey,
		Endpoint: endpoint,
	}
}

func fetchAllSessions(cfg resolvedConfig) []claugSessionStats {
	client := &http.Client{Timeout: 30 * time.Second}
	var allSessions []claugSessionStats
	page := 1

	for {
		u, err := url.Parse(cfg.Endpoint + "/api/sessions")
		if err != nil {
			log.Fatalf("parsing endpoint URL: %v", err)
		}
		q := u.Query()
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("from", fromDate)
		u.RawQuery = q.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			log.Fatalf("creating request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("fetching sessions (page %d): %v", page, err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Fatalf("API returned status %d on page %d. Check your API key.", resp.StatusCode, page)
		}

		var result sessionsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			log.Fatalf("decoding response (page %d): %v", page, err)
		}
		resp.Body.Close()

		allSessions = append(allSessions, result.Sessions...)

		log.Printf("page %d: got %d sessions (total so far: %d/%d)", page, len(result.Sessions), len(allSessions), result.Total)

		if len(allSessions) >= result.Total || len(result.Sessions) == 0 {
			break
		}
		page++
	}

	return allSessions
}

func writeExport(data dataExport) {
	blogRoot := os.Getenv("CC_STATS_BLOG_ROOT")
	if blogRoot == "" {
		// Default: relative to this script's location (scripts/build-sessions -> site/)
		exe, err := os.Getwd()
		if err != nil {
			log.Fatalf("getting working directory: %v", err)
		}
		blogRoot = filepath.Join(exe, "..", "..", "site")
	}
	dataFile := filepath.Join(blogRoot, "data", "cc_sessions.json")

	if err := os.MkdirAll(filepath.Dir(dataFile), 0o755); err != nil {
		log.Fatalf("creating data directory: %v", err)
	}

	// Atomic write: temp file + rename
	tmpFile, err := os.CreateTemp(filepath.Dir(dataFile), "cc_sessions_*.json")
	if err != nil {
		log.Fatalf("creating temp file: %v", err)
	}
	tmpPath := tmpFile.Name()

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		log.Fatalf("encoding JSON: %v", err)
	}
	_ = tmpFile.Close()

	if err := os.Rename(tmpPath, dataFile); err != nil {
		_ = os.Remove(tmpPath)
		log.Fatalf("renaming temp file: %v", err)
	}

	log.Printf("exported %d sessions to %s", len(data.Sessions), dataFile)
}

// --- Formatting helpers (matching cc-live's output exactly) ---

func formatTokensShort(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return strconv.FormatInt(n, 10)
}

func formatTokens(n int64) string {
	if n >= 10_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	s := strconv.FormatInt(n, 10)
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

func formatTime(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	secs := seconds % 60
	if minutes < 60 {
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", minutes, secs)
		}
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dh", hours)
}

func formatDate(isoDate string) string {
	if isoDate == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, strings.Replace(isoDate, "Z", "+00:00", 1))
	if err != nil {
		t, err = time.Parse(time.RFC3339, strings.Replace(isoDate, "Z", "+00:00", 1))
		if err != nil {
			if len(isoDate) >= 10 {
				return isoDate[:10]
			}
			return isoDate
		}
	}
	return t.Format("Jan 2, 2006")
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

func topTools(maps []map[string]int, n int) []toolEntry {
	merged := make(map[string]int)
	for _, m := range maps {
		for name, count := range m {
			merged[name] += count
		}
	}

	entries := make([]toolEntry, 0, len(merged))
	for name, count := range merged {
		entries = append(entries, toolEntry{
			Name:    name,
			Count:   count,
			Display: cleanToolName(name),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	if len(entries) > n {
		entries = entries[:n]
	}
	return entries
}
