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

func main() {
	apiKey := os.Getenv("CC_LIVE_API_KEY")
	if apiKey == "" {
		log.Fatal("CC_LIVE_API_KEY must be set")
	}

	http.HandleFunc("/api/live/heartbeat", func(w http.ResponseWriter, r *http.Request) {
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
				json.Unmarshal(body, &payload)
			}
		}

		s.mu.Lock()
		s.lastHeartbeat = time.Now()
		s.active = true
		s.sessions = payload.Sessions
		s.mu.Unlock()

		broadcast()
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/live/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.mu.Lock()
		wasActive := s.active
		s.active = false
		s.sessions = nil
		s.mu.Unlock()
		if wasActive {
			broadcast()
		}
		w.WriteHeader(http.StatusOK)
	})

	http.Handle("/ws/live", websocket.Handler(wsHandler))

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
				broadcast()
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

func wsHandler(ws *websocket.Conn) {
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
	s.mu.Lock()
	s.clients[c] = struct{}{}
	msg, _ := json.Marshal(message{Active: s.active, Sessions: s.sessions})
	s.mu.Unlock()

	c.send <- string(msg)

	// Keep connection open, read to detect close
	for {
		var buf string
		if err := websocket.Message.Receive(ws, &buf); err != nil {
			break
		}
	}

	removeClient(c)
}

// removeClient unregisters a client and closes its send channel exactly once.
func removeClient(c *client) {
	s.mu.Lock()
	delete(s.clients, c)
	s.mu.Unlock()
	c.closeOnce.Do(func() { close(c.send) })
}

func broadcast() {
	s.mu.RLock()
	active := s.active
	sessions := s.sessions
	clients := make([]*client, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.RUnlock()

	msg, _ := json.Marshal(message{Active: active, Sessions: sessions})
	payload := string(msg)
	for _, c := range clients {
		// Non-blocking send: drop the message if the client's buffer is full
		// rather than blocking the entire broadcast loop.
		select {
		case c.send <- payload:
		default:
			// Client is too slow — disconnect it.
			removeClient(c)
		}
	}
}
