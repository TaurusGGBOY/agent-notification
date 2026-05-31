package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type recordingNotifier struct {
	calls []recordedNotification
}

type recordedNotification struct {
	style   string
	event   string
	title   string
	message string
}

func (n *recordingNotifier) Notify(title, message string) error {
	n.calls = append(n.calls, recordedNotification{title: title, message: message})
	return nil
}

func (n *recordingNotifier) NotifyWithStyle(style, event, title, message, agent string) error {
	n.calls = append(n.calls, recordedNotification{style: style, event: event, title: title, message: message})
	return nil
}

func newTestServer(cfg *Config) *Server {
	server := NewServer(cfg)
	server.notifier = &recordingNotifier{}
	return server
}

// === Payload Validation Tests ===

func TestNotifyHandler_ValidStartEvent(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	body := `{"agent":"test","event":"start","project":"test-project"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestNotifyHandler_ValidStopEvent(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	body := `{"agent":"test","event":"stop"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestNotifyHandler_CaseInsensitiveEvent(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	testCases := []string{"START", "Stop", "Start", "STOP", "  start  "}
	for _, event := range testCases {
		body := `{"agent":"test","event":"` + event + `"}`
		req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		server.NotifyHandler(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("event %q: expected status %d, got %d", event, http.StatusNoContent, w.Code)
		}
	}
}

func TestNotifyHandler_InvalidEvent(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	body := `{"agent":"test","event":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "event must be 'start' or 'stop'" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestNotifyHandler_EmptyEvent(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	body := `{"agent":"test","event":""}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestNotifyHandler_MissingEvent(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	body := `{"agent":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestNotifyHandler_InvalidJSON(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestNotifyHandler_WrongMethod(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/notify", nil)
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestNotifyHandler_DisabledEvent(t *testing.T) {
	cfg := &Config{
		NotificationStyle: "clean",
		EnabledEvents:     []string{"stop"}, // start disabled
		FutureOverrides:   make(map[string]string),
	}
	server := newTestServer(cfg)

	body := `{"agent":"test","event":"start"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d for disabled event, got %d", http.StatusNoContent, w.Code)
	}
}

func TestNotifyHandler_ReloadsConfigAndNormalizesStyle(t *testing.T) {
	tmpDir := t.TempDir()
	oldAppData := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tmpDir)
	defer os.Setenv("APPDATA", oldAppData)

	configDir := filepath.Join(tmpDir, "AgentNotify")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir config dir failed: %v", err)
	}
	configJSON := `{"notificationStyle":"agent-badge","enabledEvents":["start","stop"],"futureOverrides":{}}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	server := newTestServer(&Config{
		NotificationStyle: "clean",
		EnabledEvents:     []string{"start", "stop"},
		FutureOverrides:   map[string]string{},
	})
	notifier := &recordingNotifier{}
	server.notifier = notifier

	body := `{"agent":"claude","event":"start","project":"agent-notification"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.calls))
	}
	if notifier.calls[0].style != "clean" {
		t.Fatalf("style = %q, want clean", notifier.calls[0].style)
	}
}

func TestNotifyHandler_UsesWorkdirAliasInMessage(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	body := `{"agent":"codex","event":"stop","workdir":"/Users/me/project"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.NotifyHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	notifier := server.notifier.(*recordingNotifier)
	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.calls))
	}
	if !strings.Contains(notifier.calls[0].message, "/Users/me/project") {
		t.Fatalf("message = %q, want workdir path", notifier.calls[0].message)
	}
}

// === Config Read/Write Tests ===

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NotificationStyle != "clean" {
		t.Errorf("expected notificationStyle 'clean', got %q", cfg.NotificationStyle)
	}

	if len(cfg.EnabledEvents) != 2 {
		t.Errorf("expected 2 enabled events, got %d", len(cfg.EnabledEvents))
	}

	if !cfg.IsEventEnabled("start") {
		t.Error("start should be enabled by default")
	}

	if !cfg.IsEventEnabled("stop") {
		t.Error("stop should be enabled by default")
	}
}

func TestIsEventEnabled(t *testing.T) {
	cfg := &Config{
		NotificationStyle: "clean",
		EnabledEvents:     []string{"start"},
		FutureOverrides:   make(map[string]string),
	}

	if cfg.IsEventEnabled("start") != true {
		t.Error("start should be enabled")
	}

	if cfg.IsEventEnabled("stop") != false {
		t.Error("stop should not be enabled")
	}

	if cfg.IsEventEnabled("") != false {
		t.Error("empty event should not be enabled")
	}
}

func TestConfigJSONRoundTrip(t *testing.T) {
	original := &Config{
		NotificationStyle: "agent-badge",
		EnabledEvents:     []string{"start", "stop"},
		FutureOverrides:   map[string]string{"key": "value"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored Config
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if restored.NotificationStyle != original.NotificationStyle {
		t.Errorf("style mismatch: got %q, want %q", restored.NotificationStyle, original.NotificationStyle)
	}

	if len(restored.EnabledEvents) != len(original.EnabledEvents) {
		t.Errorf("events count mismatch: got %d, want %d", len(restored.EnabledEvents), len(original.EnabledEvents))
	}
}

// === Style Selection Logic Tests ===

func TestFormatTitle(t *testing.T) {
	tests := []struct {
		agent  string
		event  string
		expect string
	}{
		{"claude", "start", "START · claude"},
		{"claude", "stop", "STOP · claude"},
		{"test-agent", "start", "START · test-agent"},
		{"", "start", "START · unknown"},
		{"claude", "unknown", "EVENT · claude"},
	}

	for _, tt := range tests {
		got := formatTitle(tt.agent, tt.event)
		if got != tt.expect {
			t.Errorf("formatTitle(%q, %q) = %q, want %q", tt.agent, tt.event, got, tt.expect)
		}
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name     string
		payload  NotifyPayload
		contains string
	}{
		{
			name:     "with project",
			payload:  NotifyPayload{Project: "my-project"},
			contains: "PROJECT: my-project",
		},
		{
			name:     "with cwd",
			payload:  NotifyPayload{Cwd: "~/code"},
			contains: "DIR: ~/code",
		},
		{
			name:     "with message",
			payload:  NotifyPayload{Message: "hello world"},
			contains: "hello world",
		},
		{
			name:     "with timestamp",
			payload:  NotifyPayload{Timestamp: "2024-01-15T10:30:00Z"},
			contains: "10:30:00",
		},
		{
			name:     "empty returns default",
			payload:  NotifyPayload{},
			contains: "Agent event occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMessage(tt.payload)
			if tt.contains != "" && !bytes.Contains([]byte(got), []byte(tt.contains)) {
				t.Errorf("formatMessage() = %q, want to contain %q", got, tt.contains)
			}
		})
	}
}

func TestFormatMessage_MultipleFields(t *testing.T) {
	payload := NotifyPayload{
		Project: "test-project",
		Cwd:     "/home/user",
		Message: "test message",
	}

	got := formatMessage(payload)
	want := "DIR: /home/user | PROJECT: test-project | test message"
	if got != want {
		t.Fatalf("formatMessage() = %q, want %q", got, want)
	}

	if !bytes.Contains([]byte(got), []byte("test-project")) {
		t.Error("should contain project")
	}
	if !bytes.Contains([]byte(got), []byte("/home/user")) {
		t.Error("should contain cwd")
	}
	if !bytes.Contains([]byte(got), []byte("test message")) {
		t.Error("should contain message")
	}
	if !bytes.Contains([]byte(got), []byte(" | ")) {
		t.Error("parts should be joined with ' | '")
	}
}

func TestFormatMessage_TreatsPathProjectAsDirectoryWhenCwdMissing(t *testing.T) {
	payload := NotifyPayload{
		Project: "/Users/me/project",
		Message: "done",
	}

	got := formatMessage(payload)
	want := "DIR: /Users/me/project | done"
	if got != want {
		t.Fatalf("formatMessage() = %q, want %q", got, want)
	}
}

func TestFormatMessage_DoesNotTreatRepoSlugProjectAsDirectory(t *testing.T) {
	payload := NotifyPayload{
		Project: "TaurusGGBOY/agent-notification",
		Message: "done",
	}

	got := formatMessage(payload)
	want := "PROJECT: TaurusGGBOY/agent-notification | done"
	if got != want {
		t.Fatalf("formatMessage() = %q, want %q", got, want)
	}
}

func TestFormatMessage_CompactsDeepDirectoryButKeepsTail(t *testing.T) {
	payload := NotifyPayload{
		Cwd:     "/Users/me/work/company/platform/services/agent-notification/tauri-app/src",
		Message: "done",
	}

	got := formatMessage(payload)
	if !strings.Contains(got, "DIR: /Users/me/.../agent-notification/tauri-app/src") {
		t.Fatalf("formatMessage() = %q, want compact directory with tail", got)
	}
}

// === Discovery Response Formatting Tests ===

func TestHealthHandler(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Version != "1.0.1" {
		t.Errorf("expected version '1.0.1', got %q", resp.Version)
	}
}

func TestManifestHandler(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/manifest", nil)
	w := httptest.NewRecorder()

	server.ManifestHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ManifestResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Name != "Agent Notify Server" {
		t.Errorf("expected name 'Agent Notify Server', got %q", resp.Name)
	}
	if resp.Protocol != "mdns-dns-sd" {
		t.Errorf("expected protocol 'mdns-dns-sd', got %q", resp.Protocol)
	}
	if resp.Version != "1.0.1" {
		t.Errorf("expected version '1.0.1', got %q", resp.Version)
	}
}

func TestHealthHandler_WrongMethod(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	server.HealthHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// === mDNS Discovery TXT Tests ===

func TestManifestResponse_Fields(t *testing.T) {
	resp := ManifestResponse{
		Name:            "Agent Notify Server",
		Version:         "1.0.1",
		URL:             "http://localhost:17891",
		Hostname:        "test-host",
		Protocol:        "mdns-dns-sd",
		ServiceType:     "_agent-notify._tcp.local.",
		Description:     "Windows notification server",
		SupportedEvents: []string{"start", "stop"},
		SupportedStyles: []string{"clean"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored map[string]interface{}
	json.Unmarshal(data, &restored)

	if url, ok := restored["url"].(string); !ok || url == "" {
		t.Error("url field missing or empty")
	}
	if hostname, ok := restored["hostname"].(string); !ok || hostname == "" {
		t.Error("hostname field missing or empty")
	}
	if serviceType, ok := restored["serviceType"].(string); !ok || serviceType == "" {
		t.Error("serviceType field missing or empty")
	}
}

// === Settings Handler Tests ===

func TestSettingsHandler_GET(t *testing.T) {
	handler := NewSettingsHandler()

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected content-type 'text/html; charset=utf-8', got %q", contentType)
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("AgentNotify 设置")) {
		t.Error("expected HTML title in response")
	}
	if bytes.Contains([]byte(body), []byte("通知样式")) {
		t.Error("settings page should not expose notification style controls")
	}
	if bytes.Contains([]byte(body), []byte("预览")) {
		t.Error("settings page should not expose notification preview")
	}
}

func TestSettingsHandler_POST_SaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	var configPath string

	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)
		configPath = filepath.Join(tmpDir, "Library", "Application Support", "AgentNotify", "config.json")
	} else {
		oldAppData := os.Getenv("APPDATA")
		os.Setenv("APPDATA", tmpDir)
		defer os.Setenv("APPDATA", oldAppData)
		configPath = filepath.Join(tmpDir, "AgentNotify", "config.json")
	}

	handler := NewSettingsHandler()

	body := `{"notificationStyle":"agent-badge","enabledEvents":["start"]}`
	req := httptest.NewRequest(http.MethodPost, "/settings", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify config file was written
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var cfg Config
	json.Unmarshal(data, &cfg)

	if cfg.NotificationStyle != "clean" {
		t.Errorf("expected style 'clean', got %q", cfg.NotificationStyle)
	}
}

func TestConfigHandler_GET(t *testing.T) {
	tmpDir := t.TempDir()
	oldAppData := os.Getenv("APPDATA")
	os.Setenv("APPDATA", tmpDir)
	defer os.Setenv("APPDATA", oldAppData)

	handler := &ConfigHandler{}

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["notificationStyle"] != "clean" {
		t.Errorf("expected default style 'clean', got %v", resp["notificationStyle"])
	}
}

func TestSettingsHandler_WrongMethod(t *testing.T) {
	handler := NewSettingsHandler()

	req := httptest.NewRequest(http.MethodDelete, "/settings", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// === Style Validation Tests ===

func TestIsValidStyle(t *testing.T) {
	validStyles := []string{"clean"}
	invalidStyles := []string{"invalid", "fancy", "", "CLEAN", "status-color", "agent-badge", "compact", "custom-card"}

	for _, s := range validStyles {
		if !isValidStyle(s) {
			t.Errorf("isValidStyle(%q) = false, want true", s)
		}
	}

	for _, s := range invalidStyles {
		if isValidStyle(s) {
			t.Errorf("isValidStyle(%q) = true, want false", s)
		}
	}
}

func TestIsValidEvent(t *testing.T) {
	validEvents := []string{"start", "stop"}
	invalidEvents := []string{"invalid", "pause", "", "START"}

	for _, e := range validEvents {
		if !isValidEvent(e) {
			t.Errorf("isValidEvent(%q) = false, want true", e)
		}
	}

	for _, e := range invalidEvents {
		if isValidEvent(e) {
			t.Errorf("isValidEvent(%q) = true, want false", e)
		}
	}
}

// === Settings Save/Load Round Trip Tests ===

func TestSettingsHandler_SaveLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)
	} else {
		oldAppData := os.Getenv("APPDATA")
		os.Setenv("APPDATA", tmpDir)
		defer os.Setenv("APPDATA", oldAppData)
	}

	handler := NewSettingsHandler()

	// Save config
	body := `{"notificationStyle":"compact","enabledEvents":["stop"]}`
	req := httptest.NewRequest(http.MethodPost, "/settings", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("save failed: expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Load saved config
	cfgHandler := &ConfigHandler{}
	req2 := httptest.NewRequest(http.MethodGet, "/config", nil)
	w2 := httptest.NewRecorder()
	cfgHandler.ServeHTTP(w2, req2)

	var loaded map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&loaded)

	if loaded["notificationStyle"] != "clean" {
		t.Errorf("style mismatch: got %v, want clean", loaded["notificationStyle"])
	}

	events := loaded["enabledEvents"].([]interface{})
	if len(events) != 1 || events[0] != "stop" {
		t.Errorf("events mismatch: got %v, want [stop]", events)
	}
}

// === Manifest Extended Fields Tests ===

func TestManifestHandler_ExtendedFields(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/manifest", nil)
	req.Host = "localhost:17891"
	w := httptest.NewRecorder()

	server.ManifestHandler(w, req)

	var resp ManifestResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !strings.HasPrefix(resp.URL, "http://") || !strings.HasSuffix(resp.URL, ":17891") {
		t.Errorf("expected LAN-style url on port 17891, got %q", resp.URL)
	}

	if resp.Hostname == "" {
		t.Error("hostname should not be empty")
	}

	if resp.ServiceType == "" {
		t.Error("serviceType should not be empty")
	}

	if len(resp.SupportedEvents) == 0 {
		t.Error("supportedEvents should not be empty")
	}

	if len(resp.SupportedStyles) == 0 {
		t.Error("supportedStyles should not be empty")
	}

	// Check that start and stop are in supported events
	foundStart := false
	foundStop := false
	for _, e := range resp.SupportedEvents {
		if e == "start" {
			foundStart = true
		}
		if e == "stop" {
			foundStop = true
		}
	}
	if !foundStart || !foundStop {
		t.Error("supportedEvents should contain start and stop")
	}
}

func TestNotifyHandler_RecordsRecentHistory(t *testing.T) {
	cfg := DefaultConfig()
	server := newTestServer(cfg)

	for i := 0; i < 4; i++ {
		body := `{"agent":"claude","event":"start","project":"project-` + string(rune('A'+i)) + `","message":"message"}`
		req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		server.NotifyHandler(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("notify %d status = %d", i, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/history", nil)
	w := httptest.NewRecorder()
	server.HistoryHandler(w, req)

	var resp HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode history failed: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("history length = %d, want 3", len(resp.Items))
	}
	if resp.Items[0].Project != "project-D" {
		t.Fatalf("latest project = %q, want project-D", resp.Items[0].Project)
	}
	if resp.Items[2].Project != "project-B" {
		t.Fatalf("oldest kept project = %q, want project-B", resp.Items[2].Project)
	}
}

func TestBroadcastHandler_GetAndInvalidPost(t *testing.T) {
	controller := NewBroadcastController(17891)
	handler := NewBroadcastHandler(controller)

	req := httptest.NewRequest(http.MethodGet, "/broadcast", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", w.Code, http.StatusOK)
	}

	req = httptest.NewRequest(http.MethodPost, "/broadcast", bytes.NewBufferString(`{}`))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("POST missing enabled status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// === Invalid Style Save Tests ===

func TestSettingsHandler_InvalidStyleNotSaved(t *testing.T) {
	tmpDir := t.TempDir()
	var configPath string

	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)
		configPath = filepath.Join(tmpDir, "Library", "Application Support", "AgentNotify", "config.json")
	} else {
		oldAppData := os.Getenv("APPDATA")
		os.Setenv("APPDATA", tmpDir)
		defer os.Setenv("APPDATA", oldAppData)
		configPath = filepath.Join(tmpDir, "AgentNotify", "config.json")
	}

	handler := NewSettingsHandler()

	// Try to save invalid style
	body := `{"notificationStyle":"invalid-style","enabledEvents":["start"]}`
	req := httptest.NewRequest(http.MethodPost, "/settings", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify config should use default style, not invalid style
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var cfg Config
	json.Unmarshal(data, &cfg)

	// Should not have saved the invalid style
	if cfg.NotificationStyle == "invalid-style" {
		t.Error("invalid style should not have been saved")
	}
}

// === Invalid Event Save Tests ===

func TestSettingsHandler_InvalidEventNotSaved(t *testing.T) {
	tmpDir := t.TempDir()
	var configPath string

	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)
		configPath = filepath.Join(tmpDir, "Library", "Application Support", "AgentNotify", "config.json")
	} else {
		oldAppData := os.Getenv("APPDATA")
		os.Setenv("APPDATA", tmpDir)
		defer os.Setenv("APPDATA", oldAppData)
		configPath = filepath.Join(tmpDir, "AgentNotify", "config.json")
	}

	handler := NewSettingsHandler()

	// Save with invalid event
	body := `{"notificationStyle":"clean","enabledEvents":["start","invalid-event"]}`
	req := httptest.NewRequest(http.MethodPost, "/settings", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify only valid events were saved
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var cfg Config
	json.Unmarshal(data, &cfg)

	for _, e := range cfg.EnabledEvents {
		if e == "invalid-event" {
			t.Error("invalid event should not have been saved")
		}
	}
}

// === XML Generation Tests ===

func TestFormatToastXML_Clean(t *testing.T) {
	xml := formatToastXML("clean", "start", "Agent Started", "message", "claude", "agent-notification", "")
	if !strings.Contains(xml, `template="ToastGeneric"`) {
		t.Error("clean style should use ToastGeneric template")
	}
	if strings.Contains(xml, `placement="hero"`) {
		t.Error("clean style should not use hero image")
	}
}

func TestFormatToastXML_CleanIncludesAppLogoOverride(t *testing.T) {
	xml := formatToastXML("clean", "start", "Agent Started", "message", "claude", "agent-notification", `C:\Temp\agentnotify-logo.png`)
	if !strings.Contains(xml, `placement="appLogoOverride"`) {
		t.Fatalf("clean toast XML missing app logo override: %s", xml)
	}
	if !strings.Contains(xml, `hint-crop="circle"`) {
		t.Fatalf("clean toast XML missing circular crop: %s", xml)
	}
	if !strings.Contains(xml, `agentnotify-logo.png`) {
		t.Fatalf("clean toast XML missing logo path: %s", xml)
	}
}

func TestFormatToastXML_UnsupportedStylesCollapseToClean(t *testing.T) {
	message := "Project: agent-notification | Build completed"
	styles := []string{"clean", "status-color", "agent-badge", "compact", "custom-card"}
	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			xml := formatToastXML(style, "start", "Agent Started", message, "claude", "agent-notification", `C:\Temp\agentnotify-logo.png`)
			for _, want := range []string{
				`<text hint-style="base" hint-wrap="false">Agent Started</text>`,
				`<text hint-style="captionSubtle" hint-wrap="true">Project: agent-notification | Build completed</text>`,
				`placement="appLogoOverride"`,
				`hint-crop="circle"`,
			} {
				if !strings.Contains(xml, want) {
					t.Fatalf("%s XML missing %q in %s", style, want, xml)
				}
			}
			for _, absent := range []string{`<group>`, `STATUS ·`, `START · Agent Started`, `placement="hero"`} {
				if strings.Contains(xml, absent) {
					t.Fatalf("%s XML should not contain %q in %s", style, absent, xml)
				}
			}
		})
	}
}

func TestFormatToastXML_EscapesFields(t *testing.T) {
	xml := formatToastXML("clean", "start", `Agent <Started> & "Ready"`, "done", "claude", "a'b", "")
	for _, raw := range []string{`<Started>`, `"Ready"`, "a'b"} {
		if strings.Contains(xml, raw) {
			t.Fatalf("xml contains unescaped raw field %q: %s", raw, xml)
		}
	}
	for _, escaped := range []string{"&lt;Started&gt;", "&amp;", "&quot;Ready&quot;", "a&apos;b"} {
		if !strings.Contains(xml, escaped) {
			t.Fatalf("xml missing escaped field %q: %s", escaped, xml)
		}
	}
}
