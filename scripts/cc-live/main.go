package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	SessionID   string `json:"session_id"`
	TotalTokens int64  `json:"total_tokens"`
	ToolCalls   int    `json:"tool_calls"`
	UserPrompts int    `json:"user_prompts"`
	ActiveTime  int    `json:"active_time_seconds"`
	LastPrompt  string `json:"last_prompt"`
	Project     string `json:"project"`
	Model       string `json:"model"`
}

type fileTracker struct {
	offset   int64
	metrics  sessionMetrics
	lastTime time.Time
}

const maxIdleGap = 300 // 5 minutes — cap gaps between entries

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		log.Fatalf("creating state dir: %v", err)
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: cc-live-daemon <register|unregister|serve>\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "register":
		cmdRegister()
	case "unregister":
		cmdUnregister()
	case "serve":
		cmdServe()
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

	db := openDB()
	defer db.Close()

	_, err := db.Exec(
		`INSERT OR REPLACE INTO sessions (session_id, transcript_path, cwd, model, registered_at)
		 VALUES (?, ?, ?, ?, ?)`,
		p.SessionID, p.TranscriptPath, p.Cwd, p.Model, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		log.Fatalf("inserting session: %v", err)
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
	rows, err := db.Query(`SELECT session_id, transcript_path, cwd, model, registered_at FROM sessions`)
	if err != nil {
		log.Printf("querying sessions: %v", err)
		return false, nil
	}
	defer rows.Close()

	now := time.Now()
	activeIDs := make(map[string]bool)

	for rows.Next() {
		var sessionID, transcriptPath, cwd, model, registeredAt string
		if err := rows.Scan(&sessionID, &transcriptPath, &cwd, &model, &registeredAt); err != nil {
			log.Printf("scanning row: %v", err)
			continue
		}

		activeIDs[sessionID] = true
		project := filepath.Base(cwd)

		// Get or create tracker
		tracker, ok := trackers[sessionID]
		if !ok {
			tracker = &fileTracker{
				metrics: sessionMetrics{
					SessionID: sessionID,
					Project:   project,
					Model:     model,
				},
			}
			trackers[sessionID] = tracker
		}

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

		metrics = append(metrics, tracker.metrics)
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
	// Increase buffer size for large JSONL lines
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

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
		// Count user prompts and extract last prompt text
		rawContent := msg["content"]
		if str, ok := rawContent.(string); ok && strings.TrimSpace(str) != "" {
			tracker.metrics.UserPrompts++
			prompt := strings.TrimSpace(str)
			if len(prompt) > 500 {
				prompt = prompt[:500]
			}
			tracker.metrics.LastPrompt = prompt
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
			tracker.metrics.TotalTokens += toInt64(usage["input_tokens"])
			tracker.metrics.TotalTokens += toInt64(usage["output_tokens"])
			tracker.metrics.TotalTokens += toInt64(usage["cache_creation_input_tokens"])
			tracker.metrics.TotalTokens += toInt64(usage["cache_read_input_tokens"])
		}
	}
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
