package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/websocket"
)

type sessionData struct {
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

// client wraps a WebSocket connection with a buffered send channel.
// A dedicated writer goroutine drains the channel, ensuring all writes
// to the connection are serialized (websocket.Conn is not concurrency-safe).
type client struct {
	conn      *websocket.Conn
	send      chan string
	closeOnce sync.Once
}

type state struct {
	mu            sync.RWMutex
	active        bool
	sessions      []sessionData
	lastHeartbeat time.Time
	clients       map[*client]struct{}
}

// message is the WebSocket payload sent to browser clients.
type message struct {
	Active   bool          `json:"active"`
	Sessions []sessionData `json:"sessions"`
}

// heartbeatPayload is the JSON body accepted on POST /api/live/heartbeat.
type heartbeatPayload struct {
	Sessions []sessionData `json:"sessions"`
}

var s = &state{
	clients: make(map[*client]struct{}),
}

// heartbeatHandler returns an http.HandlerFunc that accepts heartbeat POSTs.
func heartbeatHandler(apiKey string, st *state) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var payload heartbeatPayload
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			if len(body) > 0 {
				_ = json.Unmarshal(body, &payload)
			}
		}

		st.mu.Lock()
		st.lastHeartbeat = time.Now()
		st.active = true
		st.sessions = payload.Sessions
		st.mu.Unlock()

		broadcastState(st)
		w.WriteHeader(http.StatusOK)
	}
}

// stopHandler returns an http.HandlerFunc that stops the active session.
func stopHandler(apiKey string, st *state) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		st.mu.Lock()
		wasActive := st.active
		st.active = false
		st.sessions = nil
		st.mu.Unlock()
		if wasActive {
			broadcastState(st)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func main() {
	apiKey := os.Getenv("CC_LIVE_API_KEY")
	if apiKey == "" {
		log.Fatal("CC_LIVE_API_KEY must be set")
	}

	http.HandleFunc("/api/live/heartbeat", heartbeatHandler(apiKey, s))
	http.HandleFunc("/api/live/stop", stopHandler(apiKey, s))
	http.Handle("/ws/live", websocket.Handler(func(ws *websocket.Conn) {
		wsHandler(ws, s)
	}))

	// Background goroutine: mark inactive if no heartbeat in 60s
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.mu.Lock()
			if s.active && time.Since(s.lastHeartbeat) > 60*time.Second {
				s.active = false
				s.sessions = nil
				s.mu.Unlock()
				broadcastState(s)
			} else {
				s.mu.Unlock()
			}
		}
	}()

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		<-sigCh
		log.Println("shutting down")
		os.Exit(0)
	}()

	log.Println("cc-live listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func wsHandler(ws *websocket.Conn, st *state) {
	c := &client{
		conn: ws,
		send: make(chan string, 16),
	}

	// Dedicated writer goroutine: serializes all writes to this connection.
	go func() {
		for msg := range c.send {
			if err := websocket.Message.Send(c.conn, msg); err != nil {
				c.conn.Close()
				return
			}
		}
	}()

	// Register client and enqueue initial state atomically — the writer
	// goroutine will send it before any broadcast messages.
	st.mu.Lock()
	st.clients[c] = struct{}{}
	msg, _ := json.Marshal(message{Active: st.active, Sessions: st.sessions})
	st.mu.Unlock()

	c.send <- string(msg)

	// Keep connection open, read to detect close
	for {
		var buf string
		if err := websocket.Message.Receive(ws, &buf); err != nil {
			break
		}
	}

	removeClient(c, st)
}

// removeClient unregisters a client and closes its send channel exactly once.
func removeClient(c *client, st *state) {
	st.mu.Lock()
	delete(st.clients, c)
	st.mu.Unlock()
	c.closeOnce.Do(func() { close(c.send) })
}

func broadcastState(st *state) {
	st.mu.RLock()
	active := st.active
	sessions := st.sessions
	clients := make([]*client, 0, len(st.clients))
	for c := range st.clients {
		clients = append(clients, c)
	}
	st.mu.RUnlock()

	msg, _ := json.Marshal(message{Active: active, Sessions: sessions})
	payload := string(msg)
	for _, c := range clients {
		// Non-blocking send: drop the message if the client's buffer is full
		// rather than blocking the entire broadcast loop.
		select {
		case c.send <- payload:
		default:
			// Client is too slow — disconnect it.
			removeClient(c, st)
		}
	}
}
