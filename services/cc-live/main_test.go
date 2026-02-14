package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

const testAPIKey = "test-secret"

func newTestState() *state {
	return &state{
		clients: make(map[*client]struct{}),
	}
}

// --- heartbeat handler ---

func TestHeartbeatHandler_MethodNotAllowed(t *testing.T) {
	st := newTestState()
	h := heartbeatHandler(testAPIKey, st)

	req := httptest.NewRequest(http.MethodGet, "/api/live/heartbeat", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHeartbeatHandler_Unauthorized(t *testing.T) {
	st := newTestState()
	h := heartbeatHandler(testAPIKey, st)

	// Missing auth
	req := httptest.NewRequest(http.MethodPost, "/api/live/heartbeat", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing auth, got %d", w.Code)
	}

	// Wrong auth
	req = httptest.NewRequest(http.MethodPost, "/api/live/heartbeat", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong auth, got %d", w.Code)
	}
}

func TestHeartbeatHandler_ValidHeartbeat(t *testing.T) {
	st := newTestState()
	h := heartbeatHandler(testAPIKey, st)

	sessions := []sessionData{
		{SessionID: "s1", TotalTokens: 100, Project: "myproject"},
	}
	payload, _ := json.Marshal(heartbeatPayload{Sessions: sessions})

	req := httptest.NewRequest(http.MethodPost, "/api/live/heartbeat", strings.NewReader(string(payload)))
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	st.mu.RLock()
	defer st.mu.RUnlock()
	if !st.active {
		t.Fatal("expected state to be active")
	}
	if len(st.sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(st.sessions))
	}
	if st.sessions[0].SessionID != "s1" {
		t.Fatalf("expected session_id=s1, got %s", st.sessions[0].SessionID)
	}
}

// --- stop handler ---

func TestStopHandler_MethodNotAllowed(t *testing.T) {
	st := newTestState()
	h := stopHandler(testAPIKey, st)

	req := httptest.NewRequest(http.MethodGet, "/api/live/stop", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestStopHandler_Unauthorized(t *testing.T) {
	st := newTestState()
	h := stopHandler(testAPIKey, st)

	req := httptest.NewRequest(http.MethodPost, "/api/live/stop", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestStopHandler_StopsActive(t *testing.T) {
	st := newTestState()
	st.active = true
	st.sessions = []sessionData{{SessionID: "s1"}}

	h := stopHandler(testAPIKey, st)
	req := httptest.NewRequest(http.MethodPost, "/api/live/stop", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	st.mu.RLock()
	defer st.mu.RUnlock()
	if st.active {
		t.Fatal("expected state to be inactive")
	}
	if st.sessions != nil {
		t.Fatal("expected sessions to be nil")
	}
}

func TestStopHandler_AlreadyInactive(t *testing.T) {
	st := newTestState()
	// active is already false by default

	h := stopHandler(testAPIKey, st)
	req := httptest.NewRequest(http.MethodPost, "/api/live/stop", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if st.active {
		t.Fatal("expected state to remain inactive")
	}
}

// --- broadcast ---

func TestBroadcast_SendsToAll(t *testing.T) {
	st := newTestState()
	st.active = true
	st.sessions = []sessionData{{SessionID: "s1"}}

	// Create 3 clients with buffered send channels
	clients := make([]*client, 3)
	for i := range clients {
		clients[i] = &client{send: make(chan string, 16)}
		st.clients[clients[i]] = struct{}{}
	}

	broadcastState(st)

	for i, c := range clients {
		select {
		case msg := <-c.send:
			var m message
			if err := json.Unmarshal([]byte(msg), &m); err != nil {
				t.Fatalf("client %d: invalid JSON: %v", i, err)
			}
			if !m.Active {
				t.Fatalf("client %d: expected active=true", i)
			}
		case <-time.After(time.Second):
			t.Fatalf("client %d: timed out waiting for message", i)
		}
	}
}

func TestBroadcast_DropsSlowClient(t *testing.T) {
	st := newTestState()
	st.active = true

	// Create a client with a full buffer (size 1, pre-filled)
	slow := &client{send: make(chan string, 1)}
	slow.send <- "blocking"
	st.clients[slow] = struct{}{}

	// Create a fast client
	fast := &client{send: make(chan string, 16)}
	st.clients[fast] = struct{}{}

	broadcastState(st)

	// Fast client should receive the message
	select {
	case <-fast.send:
	case <-time.After(time.Second):
		t.Fatal("fast client should have received message")
	}

	// Slow client should have been removed from state
	st.mu.RLock()
	_, slowExists := st.clients[slow]
	st.mu.RUnlock()
	if slowExists {
		t.Fatal("slow client should have been removed")
	}
}

// --- WebSocket handler (end-to-end via httptest) ---

func startTestServer(st *state) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/live/heartbeat", heartbeatHandler(testAPIKey, st))
	mux.HandleFunc("/api/live/stop", stopHandler(testAPIKey, st))
	mux.Handle("/ws/live", websocket.Handler(func(ws *websocket.Conn) {
		wsHandler(ws, st)
	}))
	return httptest.NewServer(mux)
}

func dialWS(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/live"
	ws, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dialing websocket: %v", err)
	}
	return ws
}

func readWSMessage(t *testing.T, ws *websocket.Conn) message {
	t.Helper()
	var raw string
	done := make(chan error, 1)
	go func() {
		done <- websocket.Message.Receive(ws, &raw)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("reading ws message: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading ws message")
	}
	var m message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshalling ws message: %v", err)
	}
	return m
}

func TestWsHandler_InitialState(t *testing.T) {
	st := newTestState()
	st.active = true
	st.sessions = []sessionData{{SessionID: "s1", TotalTokens: 42}}

	server := startTestServer(st)
	defer server.Close()

	ws := dialWS(t, server)
	defer ws.Close()

	m := readWSMessage(t, ws)
	if !m.Active {
		t.Fatal("expected active=true in initial state")
	}
	if len(m.Sessions) != 1 || m.Sessions[0].SessionID != "s1" {
		t.Fatalf("unexpected initial sessions: %+v", m.Sessions)
	}
}

func TestWsHandler_ReceivesBroadcast(t *testing.T) {
	st := newTestState()
	server := startTestServer(st)
	defer server.Close()

	ws := dialWS(t, server)
	defer ws.Close()

	// Read initial state (inactive)
	m := readWSMessage(t, ws)
	if m.Active {
		t.Fatal("expected inactive initial state")
	}

	// Send heartbeat via HTTP
	payload, _ := json.Marshal(heartbeatPayload{
		Sessions: []sessionData{{SessionID: "s2", TotalTokens: 99}},
	})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/live/heartbeat", strings.NewReader(string(payload)))
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("heartbeat request: %v", err)
	}
	resp.Body.Close()

	// WS should receive broadcast
	m = readWSMessage(t, ws)
	if !m.Active {
		t.Fatal("expected active=true after heartbeat")
	}
	if len(m.Sessions) != 1 || m.Sessions[0].SessionID != "s2" {
		t.Fatalf("unexpected broadcast sessions: %+v", m.Sessions)
	}
}

// --- removeClient ---

func TestRemoveClient_Idempotent(t *testing.T) {
	st := newTestState()
	c := &client{send: make(chan string, 1)}
	st.clients[c] = struct{}{}

	// Should not panic when called twice
	removeClient(c, st)
	removeClient(c, st)

	st.mu.RLock()
	_, exists := st.clients[c]
	st.mu.RUnlock()
	if exists {
		t.Fatal("client should have been removed")
	}

	// Channel should be closed (writing should panic, but reading should return zero value)
	select {
	case _, ok := <-c.send:
		if ok {
			t.Fatal("expected channel to be closed")
		}
	default:
		t.Fatal("expected channel to be closed and readable")
	}
}

// --- concurrent broadcast + connect ---

func TestConcurrentBroadcastAndConnect(t *testing.T) {
	st := newTestState()
	server := startTestServer(st)
	defer server.Close()

	// Run multiple broadcasts and connections concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ws := dialWS(t, server)
			defer ws.Close()
			_ = readWSMessage(t, ws)
		}()
	}

	// Trigger broadcasts while clients connect
	for i := 0; i < 3; i++ {
		st.mu.Lock()
		st.active = true
		st.mu.Unlock()
		broadcastState(st)
	}

	wg.Wait()
}
