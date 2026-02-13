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

type state struct {
	mu            sync.RWMutex
	active        bool
	sessions      int
	lastHeartbeat time.Time
	clients       map[*websocket.Conn]struct{}
}

// message is the WebSocket payload sent to browser clients.
type message struct {
	Active   bool `json:"active"`
	Sessions int  `json:"sessions"`
}

// heartbeatPayload is the JSON body accepted on POST /api/live/heartbeat.
type heartbeatPayload struct {
	Sessions int `json:"sessions"`
}

var s = &state{
	clients: make(map[*websocket.Conn]struct{}),
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

		// Parse optional JSON body for session count
		var payload heartbeatPayload
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			if len(body) > 0 {
				json.Unmarshal(body, &payload)
			}
		}
		if payload.Sessions < 1 {
			payload.Sessions = 1
		}

		s.mu.Lock()
		s.lastHeartbeat = time.Now()
		wasActive := s.active
		prevSessions := s.sessions
		s.active = true
		s.sessions = payload.Sessions
		s.mu.Unlock()

		if !wasActive || prevSessions != payload.Sessions {
			broadcast()
		}
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
		s.sessions = 0
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
				s.sessions = 0
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
	s.mu.Lock()
	s.clients[ws] = struct{}{}
	active := s.active
	sessions := s.sessions
	s.mu.Unlock()

	// Send current state on connect
	msg, _ := json.Marshal(message{Active: active, Sessions: sessions})
	websocket.Message.Send(ws, string(msg))

	// Keep connection open, read to detect close
	for {
		var buf string
		if err := websocket.Message.Receive(ws, &buf); err != nil {
			break
		}
	}

	s.mu.Lock()
	delete(s.clients, ws)
	s.mu.Unlock()
}

func broadcast() {
	s.mu.RLock()
	active := s.active
	sessions := s.sessions
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.RUnlock()

	msg, _ := json.Marshal(message{Active: active, Sessions: sessions})
	for _, c := range clients {
		if err := websocket.Message.Send(c, string(msg)); err != nil {
			s.mu.Lock()
			delete(s.clients, c)
			s.mu.Unlock()
			c.Close()
		}
	}
}
