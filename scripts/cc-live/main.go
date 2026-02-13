package main

import (
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
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("opening db: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		session_id TEXT PRIMARY KEY,
		transcript_path TEXT NOT NULL,
		cwd TEXT NOT NULL,
		model TEXT NOT NULL DEFAULT '',
		registered_at TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatalf("creating table: %v", err)
	}
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

	wasActive := false
	emptyTicks := 0
	const maxEmptyMinutes = 30
	const tickInterval = 30 * time.Second
	const activityTimeout = 15 * time.Minute

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	// Do an immediate check on startup
	active, count := checkSessions(db, activityTimeout)
	if active {
		sendHeartbeat(endpoint, apiKey, count)
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
			if emptyTicks >= maxEmptyMinutes*60/int(tickInterval.Seconds()) {
				log.Println("no sessions for 30 minutes, exiting")
				return
			}
			continue
		}
		emptyTicks = 0

		active, count := checkSessions(db, activityTimeout)

		if active {
			sendHeartbeat(endpoint, apiKey, count)
			wasActive = true
		} else if wasActive {
			sendStop()
			wasActive = false
		}
	}
}

// checkSessions checks all registered sessions for recent transcript activity.
func checkSessions(db *sql.DB, timeout time.Duration) (active bool, count int) {
	rows, err := db.Query(`SELECT session_id, transcript_path, registered_at FROM sessions`)
	if err != nil {
		log.Printf("querying sessions: %v", err)
		return false, 0
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var sessionID, transcriptPath, registeredAt string
		if err := rows.Scan(&sessionID, &transcriptPath, &registeredAt); err != nil {
			log.Printf("scanning row: %v", err)
			continue
		}

		if transcriptPath == "" {
			// No transcript path — can't check activity, assume active
			count++
			continue
		}

		info, err := os.Stat(transcriptPath)
		if err != nil {
			// Transcript file doesn't exist yet — this is normal right after
			// SessionStart (file is created on first API response). Fall back
			// to registration time to decide if session is still "new".
			regTime, parseErr := time.Parse(time.RFC3339, registeredAt)
			if parseErr == nil && now.Sub(regTime) < timeout {
				count++
			}
			continue
		}

		if now.Sub(info.ModTime()) < timeout {
			count++
		}
	}

	return count > 0, count
}

func getEndpointAndKey() (string, string) {
	return os.Getenv("CC_LIVE_ENDPOINT"), os.Getenv("CC_LIVE_API_KEY")
}

func sendHeartbeat(endpoint, apiKey string, sessions int) {
	body, _ := json.Marshal(map[string]int{"sessions": sessions})
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
