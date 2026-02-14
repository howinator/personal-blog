package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// --- formatTokensShort ---

func TestFormatTokensShort(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{45200, "45.2k"},
		{1500000, "1.5M"},
	}
	for _, tt := range tests {
		got := formatTokensShort(tt.input)
		if got != tt.expected {
			t.Errorf("formatTokensShort(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// --- formatTokens ---

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{42, "42"},
		{1234, "1,234"},
		{1234567, "1,234,567"},
		{12345678, "12.3M"},
	}
	for _, tt := range tests {
		got := formatTokens(tt.input)
		if got != tt.expected {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// --- formatTime ---

func TestFormatTime(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{45, "45s"},
		{120, "2m"},
		{125, "2m 5s"},
		{3600, "1h"},
		{3725, "1h 2m"},
	}
	for _, tt := range tests {
		got := formatTime(tt.input)
		if got != tt.expected {
			t.Errorf("formatTime(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// --- formatDate ---

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2026-02-10T14:00:00-06:00", "Feb 10, 2026"},
		{"2026-02-10T20:00:00Z", "Feb 10, 2026"},
		{"", ""},
		{"not-a-date", "not-a-date"},
	}
	for _, tt := range tests {
		got := formatDate(tt.input)
		if got != tt.expected {
			t.Errorf("formatDate(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// --- cleanUserTexts ---

func TestCleanUserTexts(t *testing.T) {
	input := []string{
		"Add a login button",
		"[Request interrupted by user",
		"<local-command-caveat>something",
		"<system-reminder>reminder text",
		"Fix the CSS",
	}
	got := cleanUserTexts(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 cleaned texts, got %d: %v", len(got), got)
	}
	if got[0] != "Add a login button" || got[1] != "Fix the CSS" {
		t.Errorf("unexpected cleaned texts: %v", got)
	}
}

// --- fallbackSummary ---

func TestFallbackSummary(t *testing.T) {
	// Normal case
	texts := []string{"Add a login button to the header"}
	got := fallbackSummary(texts)
	if got != "Add a login button to the header" {
		t.Errorf("fallbackSummary = %q, want %q", got, "Add a login button to the header")
	}

	// All filtered
	filtered := []string{"[Request interrupted by user"}
	got = fallbackSummary(filtered)
	if got != "Short session" {
		t.Errorf("fallbackSummary (all filtered) = %q, want %q", got, "Short session")
	}

	// Truncation (>120 chars)
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	got = fallbackSummary([]string{string(long)})
	if len(got) != 120 {
		t.Errorf("fallbackSummary truncation: len=%d, want 120", len(got))
	}
	if got[117:] != "..." {
		t.Errorf("fallbackSummary should end with '...', got %q", got[117:])
	}
}

// --- toInt64 ---

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{"float64", float64(42.0), 42},
		{"int64", int64(99), 99},
		{"json.Number", json.Number("123"), 123},
		{"nil", nil, 0},
		{"string", "hello", 0},
	}
	for _, tt := range tests {
		got := toInt64(tt.input)
		if got != tt.expected {
			t.Errorf("toInt64(%v) [%s] = %d, want %d", tt.input, tt.name, got, tt.expected)
		}
	}
}

// --- redactPrompt ---

func TestRedactPrompt(t *testing.T) {
	prompt := "Add a login button"
	redacted := redactPrompt(prompt)

	// Same length
	if len(redacted) != len(prompt) {
		t.Errorf("redacted length %d != prompt length %d", len(redacted), len(prompt))
	}

	// Deterministic — same input same output
	redacted2 := redactPrompt(prompt)
	if redacted != redacted2 {
		t.Error("redactPrompt is not deterministic")
	}

	// Different input → different output
	other := redactPrompt("Something completely different here")
	if other == redacted {
		t.Error("different prompts produced identical redacted output")
	}
}

// --- processEntry ---

func TestProcessEntry_UserPrompt(t *testing.T) {
	tracker := &fileTracker{}
	entry := map[string]interface{}{
		"type":      "user",
		"message":   map[string]interface{}{"role": "user", "content": "Hello world"},
		"timestamp": "2026-02-10T14:00:00-06:00",
	}

	processEntry(entry, tracker)

	if tracker.metrics.UserPrompts != 1 {
		t.Errorf("UserPrompts = %d, want 1", tracker.metrics.UserPrompts)
	}
	if tracker.metrics.LastPrompt != "Hello world" {
		t.Errorf("LastPrompt = %q, want %q", tracker.metrics.LastPrompt, "Hello world")
	}
	if len(tracker.userTexts) != 1 || tracker.userTexts[0] != "Hello world" {
		t.Errorf("userTexts = %v, want [Hello world]", tracker.userTexts)
	}
}

func TestProcessEntry_AssistantTokens(t *testing.T) {
	tracker := &fileTracker{}
	entry := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role":    "assistant",
			"content": []interface{}{map[string]interface{}{"type": "text", "text": "response"}},
			"usage": map[string]interface{}{
				"input_tokens":                float64(100),
				"output_tokens":               float64(50),
				"cache_creation_input_tokens":  float64(20),
				"cache_read_input_tokens":      float64(10),
			},
		},
		"timestamp": "2026-02-10T14:00:30-06:00",
	}

	processEntry(entry, tracker)

	if tracker.metrics.InputTokens != 130 { // 100 + 20 + 10
		t.Errorf("InputTokens = %d, want 130", tracker.metrics.InputTokens)
	}
	if tracker.metrics.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", tracker.metrics.OutputTokens)
	}
	if tracker.metrics.TotalTokens != 180 { // 130 + 50
		t.Errorf("TotalTokens = %d, want 180", tracker.metrics.TotalTokens)
	}
	if !tracker.hasAssistantResponse {
		t.Error("expected hasAssistantResponse=true")
	}
}

func TestProcessEntry_ToolCalls(t *testing.T) {
	tracker := &fileTracker{}
	entry := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": "response"},
				map[string]interface{}{"type": "tool_use", "id": "t1", "name": "Read"},
				map[string]interface{}{"type": "tool_use", "id": "t2", "name": "Write"},
			},
			"usage": map[string]interface{}{"input_tokens": float64(10), "output_tokens": float64(5)},
		},
		"timestamp": "2026-02-10T14:00:30-06:00",
	}

	processEntry(entry, tracker)

	if tracker.metrics.ToolCalls != 2 {
		t.Errorf("ToolCalls = %d, want 2", tracker.metrics.ToolCalls)
	}
}

func TestProcessEntry_ActiveTime(t *testing.T) {
	tracker := &fileTracker{}

	// First entry — sets lastTime, no gap
	entry1 := map[string]interface{}{
		"type":      "user",
		"message":   map[string]interface{}{"role": "user", "content": "hello"},
		"timestamp": "2026-02-10T14:00:00-06:00",
	}
	processEntry(entry1, tracker)
	if tracker.metrics.ActiveTime != 0 {
		t.Errorf("ActiveTime after first entry = %d, want 0", tracker.metrics.ActiveTime)
	}

	// Second entry — 30s gap
	entry2 := map[string]interface{}{
		"type":      "assistant",
		"message":   map[string]interface{}{"role": "assistant", "content": []interface{}{}, "usage": map[string]interface{}{"input_tokens": float64(1), "output_tokens": float64(1)}},
		"timestamp": "2026-02-10T14:00:30-06:00",
	}
	processEntry(entry2, tracker)
	if tracker.metrics.ActiveTime != 30 {
		t.Errorf("ActiveTime after 30s gap = %d, want 30", tracker.metrics.ActiveTime)
	}

	// Third entry — 10 minute gap, should be capped at maxIdleGap (300s)
	entry3 := map[string]interface{}{
		"type":      "user",
		"message":   map[string]interface{}{"role": "user", "content": "still here"},
		"timestamp": "2026-02-10T14:10:30-06:00",
	}
	processEntry(entry3, tracker)
	if tracker.metrics.ActiveTime != 30+maxIdleGap {
		t.Errorf("ActiveTime after capped gap = %d, want %d", tracker.metrics.ActiveTime, 30+maxIdleGap)
	}
}

// --- parseTranscriptIncremental ---

func TestParseTranscriptIncremental(t *testing.T) {
	fixture := filepath.Join("testdata", "sample_transcript.jsonl")
	tracker := &fileTracker{}
	parseTranscriptIncremental(fixture, tracker)

	if tracker.transcriptSessionID != "test-session-123" {
		t.Errorf("sessionID = %q, want %q", tracker.transcriptSessionID, "test-session-123")
	}
	if tracker.transcriptCwd != "/home/user/myproject" {
		t.Errorf("cwd = %q, want %q", tracker.transcriptCwd, "/home/user/myproject")
	}
	if tracker.transcriptVersion != "1.0.5" {
		t.Errorf("version = %q, want %q", tracker.transcriptVersion, "1.0.5")
	}
	if tracker.metrics.UserPrompts != 2 {
		t.Errorf("UserPrompts = %d, want 2", tracker.metrics.UserPrompts)
	}
	if tracker.metrics.ToolCalls != 2 {
		t.Errorf("ToolCalls = %d, want 2", tracker.metrics.ToolCalls)
	}
	// Input: (150+20+10) + (200+0+50) + (100+0+0) = 530
	if tracker.metrics.InputTokens != 530 {
		t.Errorf("InputTokens = %d, want 530", tracker.metrics.InputTokens)
	}
	// Output: 80 + 120 + 60 = 260
	if tracker.metrics.OutputTokens != 260 {
		t.Errorf("OutputTokens = %d, want 260", tracker.metrics.OutputTokens)
	}
	if tracker.metrics.TotalTokens != 790 {
		t.Errorf("TotalTokens = %d, want 790", tracker.metrics.TotalTokens)
	}
	if tracker.metrics.LastPrompt != "Now add some CSS styling for the button" {
		t.Errorf("LastPrompt = %q", tracker.metrics.LastPrompt)
	}
	if len(tracker.userTexts) != 2 {
		t.Errorf("userTexts count = %d, want 2", len(tracker.userTexts))
	}
	if tracker.offset == 0 {
		t.Error("offset should be > 0 after parsing")
	}
}

func TestParseTranscriptIncremental_Offset(t *testing.T) {
	// Create a temp file with the first few lines
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	line1 := `{"type":"user","message":{"role":"user","content":"first prompt"},"timestamp":"2026-02-10T14:00:00-06:00"}`
	line2 := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":10,"output_tokens":5}},"timestamp":"2026-02-10T14:00:10-06:00"}`

	os.WriteFile(path, []byte(line1+"\n"+line2+"\n"), 0o644)

	tracker := &fileTracker{}
	parseTranscriptIncremental(path, tracker)

	if tracker.metrics.UserPrompts != 1 {
		t.Fatalf("after first parse: UserPrompts = %d, want 1", tracker.metrics.UserPrompts)
	}
	if tracker.metrics.TotalTokens != 15 {
		t.Fatalf("after first parse: TotalTokens = %d, want 15", tracker.metrics.TotalTokens)
	}
	savedOffset := tracker.offset

	// Append more lines
	line3 := `{"type":"user","message":{"role":"user","content":"second prompt"},"timestamp":"2026-02-10T14:01:00-06:00"}`
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(line3 + "\n")
	f.Close()

	// Re-parse — should only process new lines
	parseTranscriptIncremental(path, tracker)

	if tracker.offset <= savedOffset {
		t.Error("offset should have advanced")
	}
	if tracker.metrics.UserPrompts != 2 {
		t.Errorf("after re-parse: UserPrompts = %d, want 2", tracker.metrics.UserPrompts)
	}
	// Tokens should not be double-counted
	if tracker.metrics.TotalTokens != 15 {
		t.Errorf("after re-parse: TotalTokens = %d, want 15 (unchanged)", tracker.metrics.TotalTokens)
	}
}

// --- DB tests (in-memory SQLite) ---

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS session_stats (
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
	if err != nil {
		t.Fatalf("creating table: %v", err)
	}
	return db
}

func TestUpsertSessionStats_Insert(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	s := &parsedSession{
		tracker: &fileTracker{
			metrics: sessionMetrics{
				UserPrompts:  3,
				ToolCalls:    5,
				InputTokens:  500,
				OutputTokens: 200,
				TotalTokens:  700,
				ActiveTime:   120,
			},
			firstTimestamp:    "2026-02-10T14:00:00-06:00",
			transcriptVersion: "1.0.5",
		},
		sessionID:      "test-1",
		transcriptPath: "/tmp/test.jsonl",
		cwd:            "/home/user/proj",
		project:        "proj",
		model:          "opus",
		sensitiveInt:   0,
	}

	if err := upsertSessionStats(db, s); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	var sid, project, model string
	var tokens int64
	var prompts int
	err := db.QueryRow(`SELECT session_id, project, model, total_tokens, num_user_prompts FROM session_stats WHERE session_id = ?`, "test-1").
		Scan(&sid, &project, &model, &tokens, &prompts)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if sid != "test-1" || project != "proj" || model != "opus" || tokens != 700 || prompts != 3 {
		t.Errorf("unexpected row: sid=%s project=%s model=%s tokens=%d prompts=%d", sid, project, model, tokens, prompts)
	}
}

func TestUpsertSessionStats_PreservesSummary(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Insert with a summary
	_, err := db.Exec(`INSERT INTO session_stats (session_id, summary, total_tokens, updated_at)
		VALUES ('test-1', 'existing summary', 100, ?)`, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Upsert with empty summary — should preserve existing
	s := &parsedSession{
		tracker: &fileTracker{
			metrics: sessionMetrics{TotalTokens: 200},
		},
		sessionID: "test-1",
	}
	if err := upsertSessionStats(db, s); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	var summary string
	var tokens int64
	db.QueryRow(`SELECT summary, total_tokens FROM session_stats WHERE session_id = ?`, "test-1").
		Scan(&summary, &tokens)
	if summary != "existing summary" {
		t.Errorf("summary = %q, want %q", summary, "existing summary")
	}
	if tokens != 200 {
		t.Errorf("tokens = %d, want 200 (updated)", tokens)
	}
}

func TestExportSessionsJSON(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Insert test data
	db.Exec(`INSERT INTO session_stats
		(session_id, date, summary, project, cwd, num_user_prompts, num_tool_calls,
		 total_input_tokens, total_output_tokens, total_tokens, active_time_seconds, cc_version, updated_at)
		VALUES ('s1', '2026-02-10T14:00:00-06:00', 'Test session', 'myproj', '/home/user/myproj',
		 5, 10, 1000, 500, 1500, 300, '1.0.5', ?)`,
		time.Now().UTC().Format(time.RFC3339))

	// Set blog root to temp dir
	dir := t.TempDir()
	t.Setenv("CC_STATS_BLOG_ROOT", dir)
	os.MkdirAll(filepath.Join(dir, "data"), 0o755)

	exportSessionsJSON(db)

	dataFile := filepath.Join(dir, "data", "cc_sessions.json")
	data, err := os.ReadFile(dataFile)
	if err != nil {
		t.Fatalf("reading exported JSON: %v", err)
	}

	var export dataExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("unmarshalling exported JSON: %v", err)
	}

	if len(export.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(export.Sessions))
	}
	if export.Sessions[0].SessionID != "s1" {
		t.Errorf("session_id = %q, want %q", export.Sessions[0].SessionID, "s1")
	}
	if export.Sessions[0].Summary != "Test session" {
		t.Errorf("summary = %q, want %q", export.Sessions[0].Summary, "Test session")
	}
	if export.Totals.SessionCount != 1 {
		t.Errorf("session_count = %d, want 1", export.Totals.SessionCount)
	}
	if export.Totals.TotalTokens != 1500 {
		t.Errorf("total_tokens = %d, want 1500", export.Totals.TotalTokens)
	}
}
