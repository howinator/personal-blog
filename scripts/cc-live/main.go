package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	gosync "sync"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

var (
	stateDir = filepath.Join(os.Getenv("HOME"), ".cc-live")
	dbPath   = filepath.Join(stateDir, "state.db")
	pidFile  = filepath.Join(stateDir, "daemon.pid")
	logFile  = filepath.Join(stateDir, "daemon.log")
)

// hookPayload represents the JSON payload from Claude Code hooks.
type hookPayload struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	Source         string `json:"source"`
	Model          string `json:"model"`
}

type sessionMetrics struct {
	SessionID    string `json:"session_id"`
	TotalTokens  int64  `json:"total_tokens"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	ToolCalls    int    `json:"tool_calls"`
	UserPrompts  int    `json:"user_prompts"`
	ActiveTime   int    `json:"active_time_seconds"`
	LastPrompt   string `json:"last_prompt"`
	Project      string `json:"project"`
	Model        string `json:"model"`
	Sensitive    bool   `json:"sensitive"`
	Summary      string `json:"summary"`
}

type fileTracker struct {
	offset               int64
	metrics              sessionMetrics
	lastTime             time.Time
	userTexts            []string // all user prompt texts (for summary generation)
	summaryGenerated     bool     // don't re-call LLM after first generation
	hasAssistantResponse bool     // seen at least one assistant entry
	// Entry-level metadata extracted from transcript (used by sync)
	transcriptSessionID string
	transcriptCwd       string
	transcriptVersion   string
	firstTimestamp      string // session date (ISO string from first entry)
}

const maxIdleGap = 300 // 5 minutes — cap gaps between entries

// transcriptOverrides supply values from the hook payload that may not yet be in
// the transcript (e.g. model, sensitive flag).
type transcriptOverrides struct {
	SessionID string
	Cwd       string
	Model     string
	Sensitive bool
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		log.Fatalf("creating state dir: %v", err)
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: cc-live-daemon <register|unregister|serve|sync>\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "register":
		cmdRegister()
	case "unregister":
		cmdUnregister()
	case "serve":
		cmdServe()
	case "sync":
		cmdSync()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func openDB() *sql.DB {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("opening db: %v", err)
	}
	// Set pragmas explicitly — connection string params don't work reliably
	// with modernc.org/sqlite.
	db.Exec(`PRAGMA journal_mode=WAL`)
	db.Exec(`PRAGMA busy_timeout=10000`)

	// Retry CREATE TABLE to handle lock contention when the daemon's serve
	// process already holds the DB open.
	for attempt := 0; attempt < 5; attempt++ {
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			transcript_path TEXT NOT NULL,
			cwd TEXT NOT NULL,
			model TEXT NOT NULL DEFAULT '',
			registered_at TEXT NOT NULL
		)`)
		if err == nil {
			// Migration: add sensitive column (silently fails if already exists)
			db.Exec(`ALTER TABLE sessions ADD COLUMN sensitive INTEGER NOT NULL DEFAULT 0`)

			// Session stats table — source of truth for cc_sessions.json
			db.Exec(`CREATE TABLE IF NOT EXISTS session_stats (
				session_id TEXT PRIMARY KEY,
				transcript_path TEXT NOT NULL DEFAULT '',
				cwd TEXT NOT NULL DEFAULT '',
				project TEXT NOT NULL DEFAULT '',
				model TEXT NOT NULL DEFAULT '',
				date TEXT NOT NULL DEFAULT '',
				summary TEXT NOT NULL DEFAULT '',
				num_user_prompts INTEGER NOT NULL DEFAULT 0,
				num_tool_calls INTEGER NOT NULL DEFAULT 0,
				total_input_tokens INTEGER NOT NULL DEFAULT 0,
				total_output_tokens INTEGER NOT NULL DEFAULT 0,
				total_tokens INTEGER NOT NULL DEFAULT 0,
				active_time_seconds INTEGER NOT NULL DEFAULT 0,
				cc_version TEXT NOT NULL DEFAULT '',
				sensitive INTEGER NOT NULL DEFAULT 0,
				file_size INTEGER NOT NULL DEFAULT 0,
				updated_at TEXT NOT NULL DEFAULT ''
			)`)
			// Migration: add file_size column
			db.Exec(`ALTER TABLE session_stats ADD COLUMN file_size INTEGER NOT NULL DEFAULT 0`)
			return db
		}
		time.Sleep(200 * time.Millisecond)
	}
	log.Fatalf("creating table after retries: %v", err)
	return db
}

func readPayload() hookPayload {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("reading stdin: %v", err)
	}
	var p hookPayload
	if err := json.Unmarshal(data, &p); err != nil {
		log.Fatalf("parsing payload: %v", err)
	}
	return p
}

func cmdRegister() {
	p := readPayload()
	if p.SessionID == "" {
		log.Fatal("session_id is required")
	}

	isSensitive := os.Getenv("CC_LIVE_SENSITIVE") == "1"
	sensitive := 0
	if isSensitive {
		sensitive = 1
	}

	db := openDB()
	defer db.Close()

	_, err := db.Exec(
		`INSERT OR REPLACE INTO sessions (session_id, transcript_path, cwd, model, registered_at, sensitive)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.SessionID, p.TranscriptPath, p.Cwd, p.Model, time.Now().UTC().Format(time.RFC3339), sensitive,
	)
	if err != nil {
		log.Fatalf("inserting session: %v", err)
	}

	// Also process transcript into session_stats
	if p.TranscriptPath != "" {
		processTranscript(db, p.TranscriptPath, &transcriptOverrides{
			SessionID: p.SessionID,
			Cwd:       p.Cwd,
			Model:     p.Model,
			Sensitive: isSensitive,
		})
	}

	// Ensure daemon is running
	if !isDaemonRunning() {
		startDaemon()
	}
}

func cmdUnregister() {
	p := readPayload()
	if p.SessionID == "" {
		log.Fatal("session_id is required")
	}

	db := openDB()
	defer db.Close()

	_, err := db.Exec(`DELETE FROM sessions WHERE session_id = ?`, p.SessionID)
	if err != nil {
		log.Fatalf("deleting session: %v", err)
	}

	// Check remaining sessions
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count); err != nil {
		log.Fatalf("counting sessions: %v", err)
	}

	if count == 0 {
		// No sessions left — send immediate stop and kill daemon
		sendStop()
		killDaemon()
	}
}

// processTranscript parses a transcript file and upserts metrics into session_stats.
// overrides supply values from the hook payload that may not yet be in the transcript.
// Returns the parsed tracker for further use (e.g. summary generation), or nil if skipped.
// parsedSession holds transcript data resolved and ready for DB upsert.
type parsedSession struct {
	tracker        *fileTracker
	sessionID      string
	transcriptPath string
	cwd            string
	project        string
	model          string
	sensitiveInt   int
}

// parseTranscriptData parses a transcript and resolves metadata without touching the DB.
func parseTranscriptData(transcriptPath string, overrides *transcriptOverrides) *parsedSession {
	tracker := &fileTracker{}
	parseTranscriptIncremental(transcriptPath, tracker)

	// Resolve metadata: prefer overrides > tracker's extracted values
	sessionID := tracker.transcriptSessionID
	cwd := tracker.transcriptCwd
	model := ""
	sensitive := false
	if overrides != nil {
		if overrides.SessionID != "" {
			sessionID = overrides.SessionID
		}
		if overrides.Cwd != "" {
			cwd = overrides.Cwd
		}
		model = overrides.Model
		sensitive = overrides.Sensitive
	}

	// Fallback: derive session_id from filename (UUID.jsonl)
	if sessionID == "" {
		base := filepath.Base(transcriptPath)
		sessionID = strings.TrimSuffix(base, filepath.Ext(base))
	}

	if sessionID == "" {
		return nil
	}

	// Skip empty transcripts (no tokens = no real conversation)
	if tracker.metrics.TotalTokens == 0 {
		return nil
	}

	project := filepath.Base(cwd)
	if project == "" || project == "." {
		project = "unknown"
	}

	sensitiveInt := 0
	if sensitive {
		sensitiveInt = 1
	}

	return &parsedSession{
		tracker:        tracker,
		sessionID:      sessionID,
		transcriptPath: transcriptPath,
		cwd:            cwd,
		project:        project,
		model:          model,
		sensitiveInt:   sensitiveInt,
	}
}

// upsertSessionStats writes a parsed session to the DB, preserving existing non-empty summary.
func upsertSessionStats(db *sql.DB, s *parsedSession) error {
	_, err := db.Exec(`INSERT INTO session_stats
		(session_id, transcript_path, cwd, project, model, date, summary,
		 num_user_prompts, num_tool_calls, total_input_tokens, total_output_tokens,
		 total_tokens, active_time_seconds, cc_version, sensitive, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, '',
		 ?, ?, ?, ?,
		 ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
		 transcript_path = excluded.transcript_path,
		 cwd = CASE WHEN excluded.cwd != '' THEN excluded.cwd ELSE session_stats.cwd END,
		 project = CASE WHEN excluded.project != '' AND excluded.project != 'unknown' THEN excluded.project ELSE session_stats.project END,
		 model = CASE WHEN excluded.model != '' THEN excluded.model ELSE session_stats.model END,
		 date = CASE WHEN excluded.date != '' THEN excluded.date ELSE session_stats.date END,
		 summary = CASE WHEN session_stats.summary != '' THEN session_stats.summary ELSE excluded.summary END,
		 num_user_prompts = excluded.num_user_prompts,
		 num_tool_calls = excluded.num_tool_calls,
		 total_input_tokens = excluded.total_input_tokens,
		 total_output_tokens = excluded.total_output_tokens,
		 total_tokens = excluded.total_tokens,
		 active_time_seconds = excluded.active_time_seconds,
		 cc_version = CASE WHEN excluded.cc_version != '' THEN excluded.cc_version ELSE session_stats.cc_version END,
		 sensitive = excluded.sensitive,
		 updated_at = excluded.updated_at`,
		s.sessionID, s.transcriptPath, s.cwd, s.project, s.model,
		s.tracker.firstTimestamp,
		s.tracker.metrics.UserPrompts, s.tracker.metrics.ToolCalls,
		s.tracker.metrics.InputTokens, s.tracker.metrics.OutputTokens,
		s.tracker.metrics.TotalTokens, s.tracker.metrics.ActiveTime,
		s.tracker.transcriptVersion, s.sensitiveInt,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func processTranscript(db *sql.DB, transcriptPath string, overrides *transcriptOverrides) (*fileTracker, error) {
	s := parseTranscriptData(transcriptPath, overrides)
	if s == nil {
		return nil, nil
	}
	if err := upsertSessionStats(db, s); err != nil {
		return s.tracker, fmt.Errorf("upserting session_stats: %w", err)
	}
	return s.tracker, nil
}

func isDaemonRunning() bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if process exists
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func startDaemon() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("finding executable: %v", err)
	}

	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatalf("opening log file: %v", err)
	}

	cmd := exec.Command(exe, "serve")
	cmd.Stdout = lf
	cmd.Stderr = lf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Pass through env vars
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		lf.Close()
		log.Fatalf("starting daemon: %v", err)
	}
	lf.Close()

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		log.Fatalf("writing pid file: %v", err)
	}

	// Detach — don't wait for the child
	cmd.Process.Release()
}

func killDaemon() {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(syscall.SIGTERM)
	os.Remove(pidFile)
}

func cmdSync() {
	db := openDB()
	defer db.Close()

	transcripts := discoverTranscripts()
	log.Printf("sync: discovered %d transcripts", len(transcripts))

	// Build map of existing file sizes for skip-if-unchanged optimization
	existingSizes := make(map[string]int64) // transcript_path → file_size
	rows, err := db.Query(`SELECT transcript_path, file_size FROM session_stats WHERE file_size > 0`)
	if err == nil {
		for rows.Next() {
			var path string
			var size int64
			rows.Scan(&path, &size)
			existingSizes[path] = size
		}
		rows.Close()
	}

	// Filter to only files that need parsing
	type parseJob struct {
		path string
		size int64
	}
	var toParse []parseJob
	skipped := 0
	for _, path := range transcripts {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if prevSize, ok := existingSizes[path]; ok && info.Size() == prevSize {
			skipped++
			continue
		}
		toParse = append(toParse, parseJob{path, info.Size()})
	}

	// Phase 1: Parse files concurrently (no DB writes)
	type parseResult struct {
		session *parsedSession
		size    int64
	}
	var mu gosync.Mutex
	var parsed []parseResult
	var wg gosync.WaitGroup
	sem := make(chan struct{}, 8) // max 8 concurrent file parses

	for _, job := range toParse {
		wg.Add(1)
		go func(j parseJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			s := parseTranscriptData(j.path, nil)
			if s == nil {
				return
			}

			mu.Lock()
			parsed = append(parsed, parseResult{s, j.size})
			mu.Unlock()
		}(job)
	}
	wg.Wait()

	// Phase 2: Write to DB serially
	type trackerInfo struct {
		sessionID string
		userTexts []string
	}
	var needsSummary []trackerInfo
	for _, r := range parsed {
		if err := upsertSessionStats(db, r.session); err != nil {
			log.Printf("sync: upserting %s: %v", r.session.sessionID, err)
			continue
		}
		db.Exec(`UPDATE session_stats SET file_size = ? WHERE session_id = ?`,
			r.size, r.session.sessionID)
		needsSummary = append(needsSummary, trackerInfo{
			sessionID: r.session.sessionID,
			userTexts: r.session.tracker.userTexts,
		})
	}
	log.Printf("sync: parsed %d, skipped %d unchanged", len(toParse), skipped)

	// Filter to only sessions actually needing summaries
	type summaryJob struct {
		sessionID string
		userTexts []string
	}
	var jobs []summaryJob
	for _, info := range needsSummary {
		var existing string
		var sensitive int
		err := db.QueryRow(`SELECT summary, sensitive FROM session_stats WHERE session_id = ?`, info.sessionID).Scan(&existing, &sensitive)
		if err != nil || existing != "" || sensitive == 1 {
			continue
		}
		jobs = append(jobs, summaryJob(info))
	}

	if len(jobs) > 0 {
		log.Printf("sync: generating %d summaries (parallel)", len(jobs))

		// Parallel summary generation with bounded concurrency
		type summaryResult struct {
			sessionID string
			summary   string
		}
		results := make(chan summaryResult, len(jobs))
		sem := make(chan struct{}, 5) // max 5 concurrent API calls

		for _, job := range jobs {
			sem <- struct{}{}
			go func(j summaryJob) {
				defer func() { <-sem }()
				s := generateSummary(j.userTexts, 30*time.Second)
				results <- summaryResult{j.sessionID, s}
			}(job)
		}

		// Collect results
		generated := 0
		for range len(jobs) {
			r := <-results
			db.Exec(`UPDATE session_stats SET summary = ? WHERE session_id = ?`, r.summary, r.sessionID)
			generated++
		}
		log.Printf("sync: generated %d summaries", generated)
	}

	// Export to JSON
	exportSessionsJSON(db)
}

// syncMinDate is the earliest transcript modification time to include in sync.
var syncMinDate = time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)

// discoverTranscripts walks ~/.claude/projects/ for transcript JSONL files
// modified on or after syncMinDate.
func discoverTranscripts() []string {
	claudeDir := filepath.Join(os.Getenv("HOME"), ".claude", "projects")
	var paths []string

	filepath.Walk(claudeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		// Skip subagents directories
		if info.IsDir() && info.Name() == "subagents" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".jsonl") {
			if info.ModTime().Before(syncMinDate) {
				return nil // skip old transcripts
			}
			paths = append(paths, path)
		}
		return nil
	})

	return paths
}

// sessionExport matches the exact JSON schema expected by Hugo's cc-sessions shortcode.
type sessionExport struct {
	SessionID        string `json:"session_id"`
	Date             string `json:"date"`
	DateDisplay      string `json:"date_display"`
	Summary          string `json:"summary"`
	Project          string `json:"project"`
	Cwd              string `json:"cwd"`
	NumUserPrompts   int    `json:"num_user_prompts"`
	NumToolCalls     int    `json:"num_tool_calls"`
	TotalInputTokens int64  `json:"total_input_tokens"`
	TotalOutputTokens int64 `json:"total_output_tokens"`
	TotalTokens      int64  `json:"total_tokens"`
	TotalTokensDisplay      string `json:"total_tokens_display"`
	TotalTokensDisplayShort string `json:"total_tokens_display_short"`
	ActiveTimeSeconds       int    `json:"active_time_seconds"`
	ActiveTimeDisplay string `json:"active_time_display"`
	CcVersion        string `json:"cc_version"`
}

type totalsExport struct {
	SessionCount          int    `json:"session_count"`
	TotalTokens           int64  `json:"total_tokens"`
	TotalTokensDisplay      string `json:"total_tokens_display"`
	TotalTokensDisplayShort string `json:"total_tokens_display_short"`
	TotalToolCalls          int    `json:"total_tool_calls"`
	TotalActiveTimeSeconds int   `json:"total_active_time_seconds"`
	TotalActiveTimeDisplay string `json:"total_active_time_display"`
}

type dataExport struct {
	Sessions []sessionExport `json:"sessions"`
	Totals   totalsExport    `json:"totals"`
}

func exportSessionsJSON(db *sql.DB) {
	rows, err := db.Query(`SELECT session_id, date, summary, project, cwd,
		num_user_prompts, num_tool_calls, total_input_tokens, total_output_tokens,
		total_tokens, active_time_seconds, cc_version
		FROM session_stats
		WHERE total_tokens > 0 AND date >= ?
		ORDER BY date DESC`, "2026-02-07")
	if err != nil {
		log.Fatalf("querying session_stats for export: %v", err)
	}
	defer rows.Close()

	var sessions []sessionExport
	var totalTokens int64
	var totalToolCalls int
	var totalActiveTime int

	for rows.Next() {
		var s sessionExport
		if err := rows.Scan(&s.SessionID, &s.Date, &s.Summary, &s.Project, &s.Cwd,
			&s.NumUserPrompts, &s.NumToolCalls, &s.TotalInputTokens, &s.TotalOutputTokens,
			&s.TotalTokens, &s.ActiveTimeSeconds, &s.CcVersion); err != nil {
			log.Printf("scanning export row: %v", err)
			continue
		}
		s.DateDisplay = formatDate(s.Date)
		s.TotalTokensDisplay = formatTokens(s.TotalTokens)
		s.TotalTokensDisplayShort = formatTokensShort(s.TotalTokens)
		s.ActiveTimeDisplay = formatTime(s.ActiveTimeSeconds)
		sessions = append(sessions, s)

		totalTokens += s.TotalTokens
		totalToolCalls += s.NumToolCalls
		totalActiveTime += s.ActiveTimeSeconds
	}

	data := dataExport{
		Sessions: sessions,
		Totals: totalsExport{
			SessionCount:          len(sessions),
			TotalTokens:           totalTokens,
			TotalTokensDisplay:      formatTokens(totalTokens),
			TotalTokensDisplayShort: formatTokensShort(totalTokens),
			TotalToolCalls:          totalToolCalls,
			TotalActiveTimeSeconds: totalActiveTime,
			TotalActiveTimeDisplay: formatTime(totalActiveTime),
		},
	}

	blogRoot := os.Getenv("CC_STATS_BLOG_ROOT")
	if blogRoot == "" {
		blogRoot = filepath.Join(os.Getenv("HOME"), "projects", "personal-blog")
	}
	dataFile := filepath.Join(blogRoot, "data", "cc_sessions.json")

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(dataFile), 0o755)

	// Atomic write: temp file + rename
	tmpFile, err := os.CreateTemp(filepath.Dir(dataFile), "cc_sessions_*.json")
	if err != nil {
		log.Fatalf("creating temp file: %v", err)
	}
	tmpPath := tmpFile.Name()

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		log.Fatalf("encoding JSON: %v", err)
	}
	tmpFile.Close()

	if err := os.Rename(tmpPath, dataFile); err != nil {
		os.Remove(tmpPath)
		log.Fatalf("renaming temp file: %v", err)
	}

	log.Printf("sync: exported %d sessions to %s", len(sessions), dataFile)
}

func cmdServe() {
	endpoint := os.Getenv("CC_LIVE_ENDPOINT")
	apiKey := os.Getenv("CC_LIVE_API_KEY")
	if endpoint == "" || apiKey == "" {
		// Silently exit if not configured
		os.Exit(0)
	}

	// Write our PID
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		log.Fatalf("writing pid: %v", err)
	}
	defer os.Remove(pidFile)

	log.Printf("daemon started, pid=%d", os.Getpid())

	db := openDB()
	defer db.Close()

	trackers := make(map[string]*fileTracker)

	wasActive := false
	emptyTicks := 0
	const maxEmptyMinutes = 30
	const tickInterval = 30 * time.Second
	const activityTimeout = 15 * time.Minute

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	// Do an immediate check on startup
	active, metrics := checkSessions(db, activityTimeout, trackers)
	if active {
		sendHeartbeat(endpoint, apiKey, metrics)
		wasActive = true
	}

	for range ticker.C {
		// Count registered sessions
		var registered int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&registered); err != nil {
			log.Printf("counting sessions: %v", err)
			continue
		}

		if registered == 0 {
			emptyTicks++
			if wasActive {
				sendStop()
				wasActive = false
			}
			// Clean up trackers for removed sessions
			for k := range trackers {
				delete(trackers, k)
			}
			if emptyTicks >= maxEmptyMinutes*60/int(tickInterval.Seconds()) {
				log.Println("no sessions for 30 minutes, exiting")
				return
			}
			continue
		}
		emptyTicks = 0

		active, metrics := checkSessions(db, activityTimeout, trackers)

		if active {
			sendHeartbeat(endpoint, apiKey, metrics)
			wasActive = true
		} else if wasActive {
			sendStop()
			wasActive = false
		}
	}
}

// checkSessions checks all registered sessions, parses transcripts incrementally,
// and returns per-session metrics.
func checkSessions(db *sql.DB, timeout time.Duration, trackers map[string]*fileTracker) (active bool, metrics []sessionMetrics) {
	rows, err := db.Query(`SELECT session_id, transcript_path, cwd, model, registered_at, sensitive FROM sessions`)
	if err != nil {
		log.Printf("querying sessions: %v", err)
		return false, nil
	}
	defer rows.Close()

	now := time.Now()
	activeIDs := make(map[string]bool)

	for rows.Next() {
		var sessionID, transcriptPath, cwd, model, registeredAt string
		var sensitive int
		if err := rows.Scan(&sessionID, &transcriptPath, &cwd, &model, &registeredAt, &sensitive); err != nil {
			log.Printf("scanning row: %v", err)
			continue
		}

		activeIDs[sessionID] = true
		project := filepath.Base(cwd)

		isSensitive := sensitive == 1

		// Get or create tracker
		tracker, ok := trackers[sessionID]
		if !ok {
			tracker = &fileTracker{
				metrics: sessionMetrics{
					SessionID: sessionID,
					Project:   project,
					Model:     model,
					Sensitive: isSensitive,
				},
			}
			// Load existing summary from session_stats
			var existingSummary string
			if err := db.QueryRow(`SELECT summary FROM session_stats WHERE session_id = ?`, sessionID).Scan(&existingSummary); err == nil && existingSummary != "" {
				tracker.metrics.Summary = existingSummary
				tracker.summaryGenerated = true
			}
			trackers[sessionID] = tracker
		}
		tracker.metrics.Sensitive = isSensitive

		if transcriptPath == "" {
			// No transcript path — can't check activity, assume active with zero metrics
			tracker.metrics.Project = project
			tracker.metrics.Model = model
			metrics = append(metrics, tracker.metrics)
			continue
		}

		info, err := os.Stat(transcriptPath)
		if err != nil {
			// Transcript file doesn't exist yet — fall back to registration time
			regTime, parseErr := time.Parse(time.RFC3339, registeredAt)
			if parseErr == nil && now.Sub(regTime) < timeout {
				tracker.metrics.Project = project
				tracker.metrics.Model = model
				metrics = append(metrics, tracker.metrics)
			}
			continue
		}

		if now.Sub(info.ModTime()) >= timeout {
			continue
		}

		// Parse new transcript lines
		parseTranscriptIncremental(transcriptPath, tracker)
		tracker.metrics.Project = project
		tracker.metrics.Model = model

		// Skip non-conversation transcripts (e.g. file-history-snapshot only)
		if tracker.metrics.UserPrompts == 0 && tracker.metrics.TotalTokens == 0 {
			continue
		}

		// Generate summary after first full turn (user prompt + assistant response)
		if tracker.metrics.UserPrompts >= 1 && tracker.hasAssistantResponse && !tracker.summaryGenerated {
			if !isSensitive {
				summary := generateSummary(tracker.userTexts, 10*time.Second)
				tracker.metrics.Summary = summary
			}
			tracker.summaryGenerated = true // don't retry (also skips for sensitive)
		}

		// Persist metrics to session_stats on each tick
		sensitiveInt := 0
		if isSensitive {
			sensitiveInt = 1
		}
		db.Exec(`INSERT INTO session_stats
			(session_id, transcript_path, cwd, project, model, date, summary,
			 num_user_prompts, num_tool_calls, total_input_tokens, total_output_tokens,
			 total_tokens, active_time_seconds, cc_version, sensitive, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?,
			 ?, ?, ?, ?,
			 ?, ?, ?, ?, ?)
			ON CONFLICT(session_id) DO UPDATE SET
			 num_user_prompts = excluded.num_user_prompts,
			 num_tool_calls = excluded.num_tool_calls,
			 total_input_tokens = excluded.total_input_tokens,
			 total_output_tokens = excluded.total_output_tokens,
			 total_tokens = excluded.total_tokens,
			 active_time_seconds = excluded.active_time_seconds,
			 summary = CASE WHEN session_stats.summary != '' AND excluded.summary = ''
			           THEN session_stats.summary ELSE excluded.summary END,
			 updated_at = excluded.updated_at`,
			sessionID, transcriptPath, cwd, project, model,
			tracker.firstTimestamp, tracker.metrics.Summary,
			tracker.metrics.UserPrompts, tracker.metrics.ToolCalls,
			tracker.metrics.InputTokens, tracker.metrics.OutputTokens,
			tracker.metrics.TotalTokens, tracker.metrics.ActiveTime,
			tracker.transcriptVersion, sensitiveInt,
			time.Now().UTC().Format(time.RFC3339),
		)

		m := tracker.metrics
		if m.Sensitive && m.LastPrompt != "" {
			m.LastPrompt = redactPrompt(m.LastPrompt)
		}
		metrics = append(metrics, m)
	}

	// Clean up trackers for sessions no longer in the DB
	for k := range trackers {
		if !activeIDs[k] {
			delete(trackers, k)
		}
	}

	return len(metrics) > 0, metrics
}

// parseTranscriptIncremental reads new JSONL lines from the given transcript file
// starting at the tracker's offset, and accumulates metrics.
func parseTranscriptIncremental(path string, tracker *fileTracker) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("opening transcript %s: %v", path, err)
		return
	}
	defer f.Close()

	// Seek to last known offset
	if tracker.offset > 0 {
		if _, err := f.Seek(tracker.offset, io.SeekStart); err != nil {
			log.Printf("seeking transcript %s: %v", path, err)
			return
		}
	}

	scanner := bufio.NewScanner(f)
	// Increase buffer size for large JSONL lines (some entries with tool results can be >1MB)
	scanner.Buffer(make([]byte, 0, 256*1024), 32*1024*1024)

	var bytesRead int64
	for scanner.Scan() {
		line := scanner.Text()
		lineLen := int64(len(line)) + 1 // +1 for newline
		bytesRead += lineLen

		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		processEntry(entry, tracker)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("scanning transcript %s: %v", path, err)
	}

	tracker.offset += bytesRead
}

// processEntry processes a single transcript JSONL entry and updates tracker metrics.
func processEntry(entry map[string]interface{}, tracker *fileTracker) {
	entryType, _ := entry["type"].(string)
	msg, _ := entry["message"].(map[string]interface{})

	// Extract entry-level metadata (first occurrence only)
	if tracker.transcriptSessionID == "" {
		if sid, ok := entry["sessionId"].(string); ok && sid != "" {
			tracker.transcriptSessionID = sid
		}
	}
	if tracker.transcriptCwd == "" {
		if cwd, ok := entry["cwd"].(string); ok && cwd != "" {
			tracker.transcriptCwd = cwd
		}
	}
	if tracker.transcriptVersion == "" {
		if ver, ok := entry["version"].(string); ok && ver != "" {
			tracker.transcriptVersion = ver
		}
	}
	if tracker.firstTimestamp == "" {
		if ts, ok := entry["timestamp"].(string); ok && ts != "" {
			tracker.firstTimestamp = ts
		}
	}

	if msg == nil {
		return
	}

	// Parse timestamp for active time calculation
	if ts, ok := entry["timestamp"].(string); ok && ts != "" {
		t, err := time.Parse(time.RFC3339Nano, strings.Replace(ts, "Z", "+00:00", 1))
		if err != nil {
			// Try with Z suffix directly
			t, err = time.Parse("2006-01-02T15:04:05.999999999Z", ts)
			if err != nil {
				t, _ = time.Parse(time.RFC3339, strings.Replace(ts, "Z", "+00:00", 1))
			}
		}
		if !t.IsZero() {
			if !tracker.lastTime.IsZero() {
				gap := int(t.Sub(tracker.lastTime).Seconds())
				if gap < 0 {
					gap = 0
				}
				if gap > maxIdleGap {
					gap = maxIdleGap
				}
				tracker.metrics.ActiveTime += gap
			}
			tracker.lastTime = t
		}
	}

	content, _ := msg["content"].([]interface{})

	switch entryType {
	case "user":
		// Count user prompts and extract last prompt text + collect all texts
		rawContent := msg["content"]
		if str, ok := rawContent.(string); ok && strings.TrimSpace(str) != "" {
			tracker.metrics.UserPrompts++
			text := strings.TrimSpace(str)
			// Collect for summary generation (cap individual texts)
			ut := text
			if len(ut) > 4000 {
				ut = ut[:4000]
			}
			tracker.userTexts = append(tracker.userTexts, ut)
			// LastPrompt for live display
			if len(text) > 500 {
				text = text[:500]
			}
			tracker.metrics.LastPrompt = text
		} else if content != nil {
			hasText := false
			var lastText string
			for _, c := range content {
				block, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				if block["type"] == "text" {
					text, _ := block["text"].(string)
					if strings.TrimSpace(text) != "" {
						hasText = true
						lastText = strings.TrimSpace(text)
						// Collect for summary generation
						ut := strings.TrimSpace(text)
						if len(ut) > 4000 {
							ut = ut[:4000]
						}
						tracker.userTexts = append(tracker.userTexts, ut)
					}
				}
			}
			if hasText {
				tracker.metrics.UserPrompts++
				if len(lastText) > 500 {
					lastText = lastText[:500]
				}
				tracker.metrics.LastPrompt = lastText
			}
		}

	case "assistant":
		tracker.hasAssistantResponse = true
		// Count tool_use blocks
		for _, c := range content {
			block, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if block["type"] == "tool_use" {
				tracker.metrics.ToolCalls++
			}
		}

		// Sum tokens from usage
		usage, _ := msg["usage"].(map[string]interface{})
		if usage != nil {
			input := toInt64(usage["input_tokens"]) + toInt64(usage["cache_creation_input_tokens"]) + toInt64(usage["cache_read_input_tokens"])
			output := toInt64(usage["output_tokens"])
			tracker.metrics.InputTokens += input
			tracker.metrics.OutputTokens += output
			tracker.metrics.TotalTokens += input + output
		}
	}
}

// redactPrompt replaces a prompt with CFG-generated espionage prose of the
// same byte length. The RNG is seeded from the prompt content so the same
// prompt produces identical output across ticks, avoiding re-triggering the
// typewriter animation.
func redactPrompt(prompt string) string {
	subjects := []string{
		"The operative", "A double agent", "The handler", "An informant",
		"The analyst", "A courier", "The mole", "A cryptographer",
		"The spymaster", "A sleeper agent", "The defector", "A sentinel",
		"The asset", "A shadow broker", "The station chief", "A ghost",
		"The cipher clerk", "A turncoat", "The extraction team", "A whistleblower",
		"The case officer", "A deep cover agent", "The surveillance team", "A burned spy",
	}
	verbs := []string{
		"intercepted", "classified", "concealed", "decoded",
		"redacted", "smuggled", "shredded", "encrypted",
		"exfiltrated", "neutralized", "compromised", "surveilled",
		"extracted", "destroyed", "photographed", "sanitized",
		"compartmentalized", "infiltrated", "debriefed", "forged",
		"buried", "erased", "transmitted", "obfuscated",
	}
	objects := []string{
		"the dossier", "a trade secret", "the dead drop", "the cipher key",
		"the microfilm", "the blueprint", "the intel", "the codebook",
		"the safe house", "a burn notice", "the cover story", "the frequency list",
		"the asset roster", "a one-time pad", "the surveillance logs", "the escape route",
		"the black site", "a forged passport", "the wire transcript", "the extraction plan",
		"the sleeper list", "a classified memo", "the double agent's file", "the signal protocol",
	}
	adverbs := []string{
		"covertly", "silently", "under deep cover", "at midnight",
		"behind enemy lines", "through a back channel", "without a trace", "in the shadows",
		"before dawn", "from a rooftop", "inside the embassy", "during the exchange",
		"across the border", "under diplomatic cover", "via dead letter box", "on a secure line",
		"after the rendezvous", "beneath the floorboards", "using invisible ink", "from the control room",
		"under a false flag", "through the tunnels", "past the checkpoint", "off the grid",
	}

	// Deterministic seed from prompt content
	var seed int64
	for _, b := range []byte(prompt) {
		seed = seed*31 + int64(b)
	}
	r := rand.New(rand.NewSource(seed))

	pick := func(list []string) string { return list[r.Intn(len(list))] }
	sentence := func() string {
		return pick(subjects) + " " + pick(verbs) + " " + pick(objects) + " " + pick(adverbs) + "."
	}

	var buf strings.Builder
	for buf.Len() < len(prompt) {
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(sentence())
	}
	return buf.String()[:len(prompt)]
}

// formatTokensShort formats a token count for card headers: ≥1M → "1.2M", ≥1k → "123.4k", else number.
func formatTokensShort(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return strconv.FormatInt(n, 10)
}

// formatTokens formats a token count: ≥10M → "12.3M", else full number with commas.
func formatTokens(n int64) string {
	if n >= 10_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	// Format with commas
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		return s
	}
	// Insert commas from right
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// formatTime formats seconds into human-readable: <60→"Ns", <3600→"Nm Ns", else "Nh Nm".
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

// formatDate converts an ISO timestamp to "Feb 13, 2026" display format.
func formatDate(isoDate string) string {
	if isoDate == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, strings.Replace(isoDate, "Z", "+00:00", 1))
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.999999999Z", isoDate)
		if err != nil {
			t, err = time.Parse(time.RFC3339, strings.Replace(isoDate, "Z", "+00:00", 1))
			if err != nil {
				if len(isoDate) >= 10 {
					return isoDate[:10]
				}
				return isoDate
			}
		}
	}
	return t.Format("Jan 02, 2006")
}

// cleanUserTexts filters out system-injected text from user prompts.
func cleanUserTexts(texts []string) []string {
	skipPrefixes := []string{"[Request interrupted", "<local-command-caveat>", "<system-reminder>"}
	var cleaned []string
	for _, t := range texts {
		skip := false
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(t, prefix) {
				skip = true
				break
			}
		}
		if !skip {
			cleaned = append(cleaned, t)
		}
	}
	return cleaned
}

// fallbackSummary returns the first cleaned user text, truncated to 120 chars.
func fallbackSummary(userTexts []string) string {
	cleaned := cleanUserTexts(userTexts)
	if len(cleaned) == 0 {
		return "Short session"
	}
	fb := cleaned[0]
	if len(fb) > 120 {
		fb = fb[:117] + "..."
	}
	return fb
}

// generateSummary calls the Anthropic API to produce a 1-sentence session summary.
// timeout controls the HTTP request timeout (shorter for live daemon, longer for sync).
func generateSummary(userTexts []string, timeout time.Duration) string {
	if len(userTexts) == 0 {
		return "Empty session"
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fallbackSummary(userTexts)
	}

	cleaned := cleanUserTexts(userTexts)
	if len(cleaned) == 0 {
		return "Short session"
	}

	// Join up to 20 cleaned texts
	limit := 20
	if len(cleaned) < limit {
		limit = len(cleaned)
	}
	context := strings.Join(cleaned[:limit], "\n---\n")
	if len(context) > 4000 {
		context = context[:4000] + "..."
	}

	prompt := "Below are the user prompts from a Claude Code coding session. " +
		"Write a single sentence (max 120 characters) summarizing what the user worked on. " +
		"Be specific and concise. Do not start with 'The user'. " +
		"Just output the summary sentence, nothing else.\n\n" + context

	body, _ := json.Marshal(map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 100,
		"messages":   []map[string]interface{}{{"role": "user", "content": prompt}},
	})

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return fallbackSummary(userTexts)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("summary API call failed: %v", err)
		return fallbackSummary(userTexts)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Content) == 0 {
		return fallbackSummary(userTexts)
	}

	text := strings.TrimSpace(result.Content[0].Text)
	if text == "" {
		return fallbackSummary(userTexts)
	}
	if len(text) > 150 {
		text = text[:147] + "..."
	}
	return text
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	}
	return 0
}

func getEndpointAndKey() (string, string) {
	return os.Getenv("CC_LIVE_ENDPOINT"), os.Getenv("CC_LIVE_API_KEY")
}

func sendHeartbeat(endpoint, apiKey string, sessions []sessionMetrics) {
	payload := struct {
		Sessions []sessionMetrics `json:"sessions"`
	}{Sessions: sessions}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, endpoint+"/api/live/heartbeat", bytes.NewReader(body))
	if err != nil {
		log.Printf("creating heartbeat request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("sending heartbeat: %v", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("heartbeat returned %d", resp.StatusCode)
	}
}

func sendStop() {
	endpoint, apiKey := getEndpointAndKey()
	if endpoint == "" || apiKey == "" {
		return
	}

	req, err := http.NewRequest(http.MethodPost, endpoint+"/api/live/stop", nil)
	if err != nil {
		log.Printf("creating stop request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("sending stop: %v", err)
		return
	}
	resp.Body.Close()
}
