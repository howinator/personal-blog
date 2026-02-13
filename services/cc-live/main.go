package main

import (
	"encoding/json"
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
	lastHeartbeat time.Time
	clients       map[*websocket.Conn]struct{}
}

type message struct {
	Active bool `json:"active"`
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
		s.mu.Lock()
		s.lastHeartbeat = time.Now()
		wasActive := s.active
		s.active = true
		s.mu.Unlock()
		if !wasActive {
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
	s.mu.Unlock()

	// Send current state on connect
	msg, _ := json.Marshal(message{Active: active})
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
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.RUnlock()

	msg, _ := json.Marshal(message{Active: active})
	for _, c := range clients {
		if err := websocket.Message.Send(c, string(msg)); err != nil {
			s.mu.Lock()
			delete(s.clients, c)
			s.mu.Unlock()
			c.Close()
		}
	}
}
