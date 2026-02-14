package integration

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func baseURL() string {
	if u := os.Getenv("CC_LIVE_TEST_URL"); u != "" {
		return u
	}
	return "http://localhost:18080"
}

func apiKey() string {
	if k := os.Getenv("CC_LIVE_TEST_API_KEY"); k != "" {
		return k
	}
	return "test-secret"
}

func wsURL() string {
	base := baseURL()
	return "ws" + strings.TrimPrefix(base, "http") + "/ws/live"
}

type message struct {
	Active   bool            `json:"active"`
	Sessions json.RawMessage `json:"sessions"`
}

func dialWS(t *testing.T) *websocket.Conn {
	t.Helper()
	ws, err := websocket.Dial(wsURL(), "", baseURL())
	if err != nil {
		t.Fatalf("dialing websocket: %v", err)
	}
	return ws
}

func readMessage(t *testing.T, ws *websocket.Conn) message {
	t.Helper()
	var raw string
	done := make(chan error, 1)
	go func() {
		done <- websocket.Message.Receive(ws, &raw)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("reading ws: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out reading ws message")
	}
	var m message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshalling: %v", err)
	}
	return m
}

func sendHeartbeat(t *testing.T, sessions string) {
	t.Helper()
	body := `{"sessions":` + sessions + `}`
	req, err := http.NewRequest(http.MethodPost, baseURL()+"/api/live/heartbeat", strings.NewReader(body))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey())
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending heartbeat: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat returned %d", resp.StatusCode)
	}
}

func sendStop(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL()+"/api/live/stop", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending stop: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop returned %d", resp.StatusCode)
	}
}

func TestIntegration_HeartbeatAndWebSocket(t *testing.T) {
	ws := dialWS(t)
	defer ws.Close()

	// Read initial state
	m := readMessage(t, ws)
	// Initial state might be active or inactive depending on previous tests
	_ = m

	// Send heartbeat
	sendHeartbeat(t, `[{"session_id":"int-s1","total_tokens":42,"project":"test"}]`)

	// WS should receive broadcast with active=true
	m = readMessage(t, ws)
	if !m.Active {
		t.Fatal("expected active=true after heartbeat")
	}

	// Clean up
	sendStop(t)
	_ = readMessage(t, ws) // drain the stop broadcast
}

func TestIntegration_StopAndWebSocket(t *testing.T) {
	ws := dialWS(t)
	defer ws.Close()

	// Read initial state
	_ = readMessage(t, ws)

	// Activate then stop
	sendHeartbeat(t, `[{"session_id":"int-s2"}]`)
	_ = readMessage(t, ws)

	sendStop(t)
	m := readMessage(t, ws)
	if m.Active {
		t.Fatal("expected active=false after stop")
	}
}

func TestIntegration_MultipleClients(t *testing.T) {
	// Ensure clean state
	sendStop(t)

	const numClients = 3
	clients := make([]*websocket.Conn, numClients)
	for i := range clients {
		clients[i] = dialWS(t)
		defer clients[i].Close()
		// Read initial state
		_ = readMessage(t, clients[i])
	}

	// Send heartbeat
	sendHeartbeat(t, `[{"session_id":"int-multi"}]`)

	// All clients should receive the broadcast
	var wg sync.WaitGroup
	for i, ws := range clients {
		wg.Add(1)
		go func(idx int, c *websocket.Conn) {
			defer wg.Done()
			m := readMessage(t, c)
			if !m.Active {
				t.Errorf("client %d: expected active=true", idx)
			}
		}(i, ws)
	}
	wg.Wait()

	// Clean up
	sendStop(t)
}

func TestIntegration_AuthRejection(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, baseURL()+"/api/live/heartbeat", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer wrong-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
