package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestToastCardPathUsesLocalAppDataAndFallback(t *testing.T) {
	tmpDir := t.TempDir()

	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)

		path, err := toastCardPath()
		if err != nil {
			t.Fatalf("toastCardPath failed: %v", err)
		}
		want := filepath.Join(tmpDir, "Library", "Caches", "AgentNotify", "toast-card.png")
		if path != want {
			t.Fatalf("path = %q, want %q", path, want)
		}
	} else {
		oldLocalAppData := os.Getenv("LOCALAPPDATA")
		defer os.Setenv("LOCALAPPDATA", oldLocalAppData)
		os.Setenv("LOCALAPPDATA", tmpDir)

		path, err := toastCardPath()
		if err != nil {
			t.Fatalf("toastCardPath with LOCALAPPDATA failed: %v", err)
		}
		want := filepath.Join(tmpDir, "AgentNotify", "toast-card.png")
		if path != want {
			t.Fatalf("path = %q, want %q", path, want)
		}
		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			t.Fatalf("expected directory to exist: %v", err)
		}

		os.Setenv("LOCALAPPDATA", "")
		path, err = toastCardPath()
		if err != nil {
			t.Fatalf("toastCardPath fallback failed: %v", err)
		}
		if !strings.HasSuffix(path, filepath.Join("AgentNotify", "toast-card.png")) {
			t.Fatalf("fallback path = %q, want AgentNotify toast card suffix", path)
		}
	}
}

func TestToastAppLogoPathAndRenderWritesPNG(t *testing.T) {
	tmpDir := t.TempDir()
	var logoDir string

	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)
		logoDir = filepath.Join(tmpDir, "Library", "Caches", "AgentNotify")
	} else {
		oldLocalAppData := os.Getenv("LOCALAPPDATA")
		defer os.Setenv("LOCALAPPDATA", oldLocalAppData)
		os.Setenv("LOCALAPPDATA", tmpDir)
		logoDir = filepath.Join(tmpDir, "AgentNotify")
	}

	path, err := toastAppLogoPath()
	if err != nil {
		t.Fatalf("toastAppLogoPath failed: %v", err)
	}
	if want := filepath.Join(logoDir, "app-logo.png"); path != want {
		t.Fatalf("logo path = %q, want %q", path, want)
	}

	if err := renderToastAppLogo(path); err != nil {
		t.Fatalf("renderToastAppLogo failed: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open rendered logo failed: %v", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode rendered logo failed: %v", err)
	}
	if got, want := img.Bounds().Dx(), 256; got != want {
		t.Fatalf("logo width = %d, want %d", got, want)
	}
	if got, want := img.Bounds().Dy(), 256; got != want {
		t.Fatalf("logo height = %d, want %d", got, want)
	}
	if _, _, _, alpha := img.At(0, 0).RGBA(); alpha == 0 {
		t.Fatal("logo background should be opaque")
	}
}

func TestRenderToastCardWritesScaledPNGWithEventAccent(t *testing.T) {
	testCases := []struct {
		name   string
		event  string
		accent [3]uint32
	}{
		{name: "start", event: "start", accent: [3]uint32{74, 222, 128}},
		{name: "stop", event: "stop", accent: [3]uint32{248, 113, 113}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "toast-card.png")
			card := ToastCard{
				Event:   tc.event,
				Title:   "  Very Long Agent Notification Title  ",
				Agent:   " codex ",
				Project: "agent-notification",
				Message: "message body for coverage",
			}

			if err := renderToastCard(path, card); err != nil {
				t.Fatalf("renderToastCard failed: %v", err)
			}

			file, err := os.Open(path)
			if err != nil {
				t.Fatalf("open rendered png failed: %v", err)
			}
			defer file.Close()

			img, err := png.Decode(file)
			if err != nil {
				t.Fatalf("decode rendered png failed: %v", err)
			}
			if got, want := img.Bounds().Dx(), scale(toastCardWidth); got != want {
				t.Fatalf("png width = %d, want %d", got, want)
			}
			if got, want := img.Bounds().Dy(), scale(toastCardHeight); got != want {
				t.Fatalf("png height = %d, want %d", got, want)
			}

			r, g, b, _ := img.At(scale(4), scale(20)).RGBA()
			if got := [3]uint32{r >> 8, g >> 8, b >> 8}; got != tc.accent {
				t.Fatalf("accent pixel = %v, want %v", got, tc.accent)
			}
		})
	}
}

func TestToastCardSummaryLinesPrioritizeEventAgentAndDirectory(t *testing.T) {
	card := ToastCard{
		Event:   "stop",
		Title:   "STOP · codex",
		Agent:   "codex",
		Message: "DIR: /Users/me/project | PROJECT: agent-notification | DIR: done",
	}

	eventAgent, directory, detail := toastCardSummaryLines(card)
	if eventAgent != "STOP · codex" {
		t.Fatalf("eventAgent = %q, want STOP · codex", eventAgent)
	}
	if directory != "/Users/me/project" {
		t.Fatalf("directory = %q, want /Users/me/project", directory)
	}
	if detail != "PROJECT: agent-notification | DIR: done" {
		t.Fatalf("detail = %q, want project detail", detail)
	}
}

func TestCompactDirectoryForDisplayPreservesTailForDeepPaths(t *testing.T) {
	unixPath := "/Users/me/work/company/platform/services/agent-notification/tauri-app/src"
	got := compactDirectoryForDisplay(unixPath, 46)
	if got != "/Users/me/.../agent-notification/tauri-app/src" {
		t.Fatalf("unix compact path = %q", got)
	}

	windowsPath := `C:\Users\me\work\company\platform\agent-notification\windows-server`
	got = compactDirectoryForDisplay(windowsPath, 50)
	if got != `C:\Users\me\...\agent-notification\windows-server` {
		t.Fatalf("windows compact path = %q", got)
	}
}

func TestCompactDirectoryForDisplayFallsBackToTailTruncation(t *testing.T) {
	if got := compactDirectoryForDisplay("abcdef", 3); got != "def" {
		t.Fatalf("small max compact path = %q, want def", got)
	}
	if got := compactDirectoryForDisplay("abcdef", 5); got != "...ef" {
		t.Fatalf("tail compact path = %q, want ...ef", got)
	}
	if got := compactDirectoryForDisplay("abc", 5); got != "abc" {
		t.Fatalf("short compact path = %q, want abc", got)
	}
}

func TestHasWindowsDriveRootValidatesDriveAndSeparator(t *testing.T) {
	valid := []string{`C:\Users\me`, "d:/work"}
	for _, path := range valid {
		if !hasWindowsDriveRoot(path) {
			t.Fatalf("hasWindowsDriveRoot(%q) = false, want true", path)
		}
	}

	invalid := []string{"", "C:", "1:/work", "C|/work", "C:work"}
	for _, path := range invalid {
		if hasWindowsDriveRoot(path) {
			t.Fatalf("hasWindowsDriveRoot(%q) = true, want false", path)
		}
	}
}

func TestDrawCircleOnlyPaintsInsideBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	drawCircle(img, 0, 0, 5, image.Black)

	if _, _, _, a := img.At(0, 0).RGBA(); a == 0 {
		t.Fatal("center pixel should be painted")
	}
	if _, _, _, a := img.At(19, 19).RGBA(); a != 0 {
		t.Fatal("far outside pixel should stay transparent")
	}
}

func TestTruncateTextHandlesWhitespaceRuneAndSmallLimits(t *testing.T) {
	if got := truncateText("  hello  ", 10); got != "hello" {
		t.Fatalf("trimmed text = %q, want hello", got)
	}
	if got := truncateText("abcdef", 3); got != "abc" {
		t.Fatalf("small limit text = %q, want abc", got)
	}
	if got := truncateText("你好世界abc", 5); got != "你好..." {
		t.Fatalf("rune truncation = %q, want 你好...", got)
	}
}

func TestToastXMLCleanLogoAndImageAttributes(t *testing.T) {
	xml := formatToastXML("custom-card", "start", `Title & "quoted"`, "message", "codex", "Project", `C:\Temp\app-logo.png`)

	for _, want := range []string{
		`<image placement="appLogoOverride"`,
		`hint-crop="circle"`,
		`src="C:\Temp\app-logo.png"`,
		`Title &amp; &quot;quoted&quot;`,
		`placement="attribution">Project`,
	} {
		if !strings.Contains(xml, want) {
			t.Fatalf("clean XML missing %q in %s", want, xml)
		}
	}

	xml = buildToastXML("title", "project", "https://example.com/card.png", "appLogoOverride", "circle", "Agent C")
	for _, want := range []string{
		`src="https://example.com/card.png"`,
		`hint-crop="circle"`,
		`placement="appLogoOverride"`,
		`placement="attribution">Agent C`,
	} {
		if !strings.Contains(xml, want) {
			t.Fatalf("image XML missing %q in %s", want, xml)
		}
	}
}

func TestNormalizeToastImagePathPreservesEmptyAndURLs(t *testing.T) {
	if got := normalizeToastImagePath(""); got != "" {
		t.Fatalf("empty path = %q, want empty", got)
	}
	if got := normalizeToastImagePath("file:///tmp/card.png"); got != "file:///tmp/card.png" {
		t.Fatalf("url path = %q, want unchanged", got)
	}
	if got := normalizeToastImagePath(filepath.Join("a", "..", "b", "card.png")); got != filepath.Join("b", "card.png") {
		t.Fatalf("cleaned path = %q, want b/card.png", got)
	}
}

func TestAgentInitialHandlesEmptyWhitespaceAndUnicode(t *testing.T) {
	testCases := map[string]string{
		"":      "?",
		"   ":   "?",
		"codex": "C",
		" 飞书":   "飞",
	}
	for input, want := range testCases {
		if got := agentInitial(input); got != want {
			t.Fatalf("agentInitial(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatToastXMLAlwaysClean(t *testing.T) {
	xml := formatToastXML("status-color", "start", "Agent Started", "message", "codex", "agent-notification", "")
	if strings.Contains(xml, `STATUS ·`) || strings.Contains(xml, `<group>`) || strings.Contains(xml, `placement="hero"`) {
		t.Fatalf("toast XML should stay clean: %s", xml)
	}
	if !strings.Contains(xml, `placement="attribution">agent-notification`) {
		t.Fatalf("clean toast XML missing attribution: %s", xml)
	}
}

func TestManifestAndHistoryWrongMethods(t *testing.T) {
	server := newTestServer(DefaultConfig())

	req := httptest.NewRequest(http.MethodPost, "/manifest", nil)
	w := httptest.NewRecorder()
	server.ManifestHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /manifest status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}

	req = httptest.NewRequest(http.MethodPost, "/history", nil)
	w = httptest.NewRecorder()
	server.HistoryHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /history status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHistoryHandlerEmptyAndRecordHistoryDefaultsAgent(t *testing.T) {
	server := newTestServer(DefaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/history", nil)
	w := httptest.NewRecorder()
	server.HistoryHandler(w, req)

	var empty HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&empty); err != nil {
		t.Fatalf("decode empty history failed: %v", err)
	}
	if empty.Items == nil {
		t.Fatal("empty history should encode as [] not null")
	}
	if len(empty.Items) != 0 {
		t.Fatalf("empty history length = %d, want 0", len(empty.Items))
	}

	server.recordHistory(NotifyPayload{Agent: "   ", Event: "start", Project: "  p  ", Message: "  m  "}, "start")
	req = httptest.NewRequest(http.MethodGet, "/history", nil)
	w = httptest.NewRecorder()
	server.HistoryHandler(w, req)

	var resp HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode history failed: %v", err)
	}
	if got := resp.Items[0].Agent; got != "unknown" {
		t.Fatalf("history agent = %q, want unknown", got)
	}
	if got := resp.Items[0].Project; got != "p" {
		t.Fatalf("history project = %q, want p", got)
	}
	if got := resp.Items[0].Message; got != "m" {
		t.Fatalf("history message = %q, want m", got)
	}
}

func TestCurrentConfigFallsBackToCachedConfigOnLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	var cfgDir string
	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)
		cfgDir = filepath.Join(tmpDir, "Library", "Application Support", "AgentNotify")
	} else {
		t.Setenv("APPDATA", tmpDir)
		cfgDir = filepath.Join(tmpDir, "AgentNotify")
	}
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir config dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{bad json`), 0644); err != nil {
		t.Fatalf("write invalid config failed: %v", err)
	}

	cached := &Config{
		NotificationStyle: "clean",
		EnabledEvents:     []string{"stop"},
		FutureOverrides:   map[string]string{},
	}
	server := newTestServer(cached)

	if got := server.currentConfig(); got != cached {
		t.Fatal("currentConfig should return cached config when LoadConfig fails")
	}
	if server.config.NotificationStyle != "clean" {
		t.Fatalf("cached style = %q, want clean", server.config.NotificationStyle)
	}
}

func TestEnvOrDefaultTrimsAndFallsBack(t *testing.T) {
	t.Setenv("AGENT_NOTIFY_TEST_ENV", "")
	if got := envOrDefault("AGENT_NOTIFY_TEST_ENV", "fallback"); got != "fallback" {
		t.Fatalf("empty env = %q, want fallback", got)
	}

	t.Setenv("AGENT_NOTIFY_TEST_ENV", "  value  ")
	if got := envOrDefault("AGENT_NOTIFY_TEST_ENV", "fallback"); got != "value" {
		t.Fatalf("trimmed env = %q, want value", got)
	}
}

func TestWithCORSHandlesOptionsAndDelegatesRequests(t *testing.T) {
	called := false
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/notify", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if called {
		t.Fatal("OPTIONS request should not call next handler")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("CORS origin = %q, want *", got)
	}

	req = httptest.NewRequest(http.MethodPost, "/notify", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d, want %d", w.Code, http.StatusAccepted)
	}
	if !called {
		t.Fatal("non-OPTIONS request should call next handler")
	}
}

func TestBroadcastControllerDisableCancelsRunningAdvertisement(t *testing.T) {
	cancelled := false
	controller := NewBroadcastController(17891)
	controller.enabled = true
	controller.cancel = func() { cancelled = true }

	if err := controller.SetEnabled(false); err != nil {
		t.Fatalf("SetEnabled(false) failed: %v", err)
	}
	if controller.Enabled() {
		t.Fatal("broadcast should be disabled")
	}
	if !cancelled {
		t.Fatal("cancel function should be called")
	}
	if controller.cancel != nil {
		t.Fatal("cancel function should be cleared")
	}
}

func TestBroadcastHandlerPostFalseInvalidJSONAndWrongMethod(t *testing.T) {
	controller := NewBroadcastController(17891)
	handler := NewBroadcastHandler(controller)

	req := httptest.NewRequest(http.MethodPost, "/broadcast", bytes.NewBufferString(`{"enabled":false}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST false status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode broadcast response failed: %v", err)
	}
	if resp["enabled"] {
		t.Fatal("broadcast enabled response should be false")
	}

	req = httptest.NewRequest(http.MethodPost, "/broadcast", bytes.NewBufferString(`{bad json`))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	req = httptest.NewRequest(http.MethodPut, "/broadcast", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestSettingsHandlerInvalidJSONAndDirectoryFailure(t *testing.T) {
	handler := &SettingsHandler{configPath: filepath.Join(t.TempDir(), "config.json")}

	req := httptest.NewRequest(http.MethodPost, "/settings", bytes.NewBufferString(`{bad json`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "not-a-directory")
	if err := os.WriteFile(blockingFile, []byte("file"), 0644); err != nil {
		t.Fatalf("write blocking file failed: %v", err)
	}
	handler = &SettingsHandler{configPath: filepath.Join(blockingFile, "config.json")}
	req = httptest.NewRequest(http.MethodPost, "/settings", bytes.NewBufferString(`{"notificationStyle":"clean"}`))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("directory failure status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestConfigHandlerWrongMethodAndFallbackPath(t *testing.T) {
	tmpDir := t.TempDir()
	var wantPath string

	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)
		wantPath = filepath.Join(tmpDir, "Library", "Application Support", "AgentNotify", "config.json")
	} else {
		oldAppData := os.Getenv("APPDATA")
		oldUserProfile := os.Getenv("USERPROFILE")
		defer os.Setenv("APPDATA", oldAppData)
		defer os.Setenv("USERPROFILE", oldUserProfile)
		os.Setenv("APPDATA", "")
		os.Setenv("USERPROFILE", tmpDir)
		wantPath = filepath.Join(tmpDir, "AppData", "Roaming", "AgentNotify", "config.json")
	}

	handler := &ConfigHandler{}
	req := httptest.NewRequest(http.MethodPost, "/config", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /config status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}

	req = httptest.NewRequest(http.MethodGet, "/config", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /config status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode config response failed: %v", err)
	}
	if resp["_path"] != wantPath {
		t.Fatalf("config path = %q, want %q", resp["_path"], wantPath)
	}
}

func TestLoadConfigInvalidJSONReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	var cfgDir string
	if runtime.GOOS == "darwin" {
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)
		cfgDir = filepath.Join(tmpDir, "Library", "Application Support", "AgentNotify")
	} else {
		t.Setenv("APPDATA", tmpDir)
		cfgDir = filepath.Join(tmpDir, "AgentNotify")
	}
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir config dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{bad json`), 0644); err != nil {
		t.Fatalf("write invalid config failed: %v", err)
	}

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig should fail on invalid JSON")
	}
}
