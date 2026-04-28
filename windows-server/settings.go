package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const settingsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>AgentNotify Settings</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#1a1a2e;color:#eee;min-height:100vh;padding:2rem}
h1{color:#00d4ff;margin-bottom:.5rem}
h2{color:#fff;font-size:1.1rem;margin:1.5rem 0 .75rem}
.status{display:flex;gap:2rem;background:#16213e;padding:1rem;border-radius:8px;margin-bottom:1.5rem}
.status-item span{display:block;color:#888;font-size:.85rem}
.status-item strong{color:#00d4ff;font-size:1.2rem}
.card-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:1rem;margin-bottom:1.5rem}
.preset-card{background:#16213e;border:2px solid #333;border-radius:8px;padding:1rem;cursor:pointer;transition:all .2s}
.preset-card:hover{border-color:#555}
.preset-card.selected{border-color:#00d4ff;background:#1a2a4a}
.preset-card h3{color:#fff;margin-bottom:.5rem}
.preset-card p{color:#888;font-size:.85rem}
.toggles{display:flex;gap:1rem;margin-bottom:1.5rem}
.toggle-btn{padding:.75rem 1.5rem;border:none;border-radius:6px;cursor:pointer;font-size:1rem;transition:all .2s}
.toggle-btn.start{background:#1e5128;color:#4ade80}
.toggle-btn.start:disabled{opacity:.4;cursor:not-allowed}
.toggle-btn.stop{background:#4a1e1e;color:#f87171}
.toggle-btn.stop:disabled{opacity:.4;cursor:not-allowed}
.toggle-btn.active.start{background:#4ade80;color:#1e5128}
.toggle-btn.active.stop{background:#f87171;color:#4a1e1e}
.actions{display:flex;gap:1rem;margin-bottom:1.5rem}
.btn{padding:.75rem 1.5rem;border:none;border-radius:6px;cursor:pointer;font-size:1rem;transition:all .2s}
.btn-primary{background:#00d4ff;color:#1a1a2e;font-weight:600}
.btn-primary:hover{background:#00b8d9}
.btn-secondary{background:#333;color:#fff}
.btn-secondary:hover{background:#444}
.preview{background:#0a0a1a;border-radius:8px;padding:1.5rem;margin-bottom:1.5rem}
.preview h3{color:#888;margin-bottom:1rem;font-size:.9rem}
.toast-preview{padding:1rem;border-radius:6px;margin-bottom:.75rem;transition:all .3s}
.toast-preview.clean{background:#1e293b}
.toast-preview.status-color{background:#1e293b;border-left:4px solid #4ade80}
.toast-preview.agent-badge{background:#1e293b;display:flex;align-items:center;gap:.75rem}
.toast-preview.agent-badge .avatar{width:40px;height:40px;border-radius:50%;background:#00d4ff;display:flex;align-items:center;justify-content:center;font-weight:bold;color:#1a1a2e}
.toast-preview.compact{background:#1e293b;padding:.5rem 1rem;font-size:.9rem}
.toast-title{color:#fff;font-weight:600;margin-bottom:.25rem}
.toast-message{color:#aaa;font-size:.9rem}
.toast-preview.agent-badge .toast-title{color:#00d4ff}
.hidden{display:none}
#toastResult{margin-top:1rem;padding:1rem;border-radius:6px}
#toastResult.success{background:#1e5128;color:#4ade80}
#toastResult.error{background:#4a1e1e;color:#f87171}
</style>
</head>
<body>
<h1>AgentNotify Settings</h1>

<div class="status">
<div class="status-item"><span>Status</span><strong id="serverStatus">Loading...</strong></div>
<div class="status-item"><span>HTTP URL</span><strong>http://localhost:17891</strong></div>
<div class="status-item"><span>UDP Discovery</span><strong>Port 17892</strong></div>
<div class="status-item"><span>Config Path</span><strong id="configPath">Loading...</strong></div>
</div>

<h2>Notification Style</h2>
<div class="card-grid" id="styleGrid">
<div class="preset-card selected" data-style="clean" onclick="selectStyle('clean')">
<h3>🧹 Clean</h3>
<p>Minimal, distraction-free</p>
</div>
<div class="preset-card" data-style="status-color" onclick="selectStyle('status-color')">
<h3>🎨 Status Color</h3>
<p>Color-coded by event type</p>
</div>
<div class="preset-card" data-style="agent-badge" onclick="selectStyle('agent-badge')">
<h3>🏷️ Agent Badge</h3>
<p>Shows agent avatar</p>
</div>
<div class="preset-card" data-style="compact" onclick="selectStyle('compact')">
<h3>📦 Compact</h3>
<p>Minimal footprint</p>
</div>
</div>

<h2>Enabled Events</h2>
<div class="toggles">
<button class="toggle-btn start active" id="toggleStart" onclick="toggleEvent('start')">▶ Start Events</button>
<button class="toggle-btn stop active" id="toggleStop" onclick="toggleEvent('stop')">⏹ Stop Events</button>
</div>

<h2>Preview</h2>
<div class="preview">
<div class="toast-preview clean" id="previewStart">
<div class="toast-title">🚀 Agent Start: claude</div>
<div class="toast-message">Project: my-project | CWD: ~/code</div>
</div>
<div class="toast-preview clean" id="previewStop">
<div class="toast-title">⏹️ Agent Stop: claude</div>
<div class="toast-message">Project: my-project | CWD: ~/code</div>
</div>
</div>

<div class="actions">
<button class="btn btn-primary" onclick="sendTestToast()">🔔 Send Test Toast</button>
<button class="btn btn-secondary" onclick="saveSettings()">💾 Save Settings</button>
</div>

<div id="toastResult" class="hidden"></div>

<script>
let config = {notificationStyle:'clean', enabledEvents:['start','stop'], futureOverrides:{}};

async function loadConfig() {
  try {
    const res = await fetch('/config');
    if (res.ok) config = await res.json();
  } catch(e) {}

  document.getElementById('serverStatus').textContent = 'Running';
  document.getElementById('configPath').textContent = config._path || '%APPDATA%\\AgentNotify\\config.json';

  document.querySelectorAll('.preset-card').forEach(c => {
    c.classList.toggle('selected', c.dataset.style === config.notificationStyle);
  });

  updatePreview();
  updateToggles();
}

function selectStyle(style) {
  config.notificationStyle = style;
  document.querySelectorAll('.preset-card').forEach(c => {
    c.classList.toggle('selected', c.dataset.style === style);
  });
  updatePreview();
}

function toggleEvent(event) {
  const idx = config.enabledEvents.indexOf(event);
  if (idx >= 0) {
    config.enabledEvents.splice(idx, 1);
  } else {
    config.enabledEvents.push(event);
  }
  updateToggles();
}

function updateToggles() {
  const startActive = config.enabledEvents.includes('start');
  const stopActive = config.enabledEvents.includes('stop');
  document.getElementById('toggleStart').classList.toggle('active', startActive);
  document.getElementById('toggleStop').classList.toggle('active', stopActive);
}

function updatePreview() {
  const style = config.notificationStyle;
  const previewStart = document.getElementById('previewStart');
  const previewStop = document.getElementById('previewStop');

  previewStart.className = 'toast-preview ' + style;
  previewStop.className = 'toast-preview ' + style;

  if (style === 'agent-badge') {
    previewStart.innerHTML = '<div class="avatar">C</div><div><div class="toast-title">🚀 Agent Start: claude</div><div class="toast-message">Project: my-project</div></div>';
    previewStop.innerHTML = '<div class="avatar">C</div><div><div class="toast-title">⏹️ Agent Stop: claude</div><div class="toast-message">Project: my-project</div></div>';
  } else if (style === 'status-color') {
    previewStart.innerHTML = '<div class="toast-title" style="color:#4ade80">🚀 Agent Start: claude</div><div class="toast-message">Project: my-project | CWD: ~/code</div>';
    previewStop.innerHTML = '<div class="toast-title" style="color:#f87171">⏹️ Agent Stop: claude</div><div class="toast-message">Project: my-project | CWD: ~/code</div>';
  } else {
    previewStart.innerHTML = '<div class="toast-title">🚀 Agent Start: claude</div><div class="toast-message">Project: my-project | CWD: ~/code</div>';
    previewStop.innerHTML = '<div class="toast-title">⏹️ Agent Stop: claude</div><div class="toast-message">Project: my-project | CWD: ~/code</div>';
  }
}

async function saveSettings() {
  const saveKeys = {notificationStyle:true, enabledEvents:true};
  const saveData = {};
  for (const [k,v] of Object.entries(config)) if (saveKeys[k]) saveData[k] = v;

  try {
    const res = await fetch('/settings', {
      method:'POST',
      headers:{'Content-Type':'application/json'},
      body: JSON.stringify(saveData)
    });
    showResult(res.ok ? 'Settings saved successfully!' : 'Failed to save settings', !res.ok);
  } catch(e) {
    showResult('Error: ' + e.message, true);
  }
}

async function sendTestToast() {
  try {
    const res = await fetch('/notify', {
      method:'POST',
      headers:{'Content-Type':'application/json'},
      body: JSON.stringify({
        agent:'test-agent',
        event:'start',
        project:'Settings UI',
        cwd:'C:\\test',
        message:'Test notification from settings UI',
        timestamp: new Date().toISOString(),
        sourcePayload:{}
      })
    });
    showResult(res.ok ? 'Test toast sent!' : 'Failed to send toast', !res.ok);
  } catch(e) {
    showResult('Error: ' + e.message, true);
  }
}

function showResult(msg, isError) {
  const el = document.getElementById('toastResult');
  el.textContent = msg;
  el.className = isError ? 'error' : 'success';
  el.classList.remove('hidden');
  setTimeout(() => el.classList.add('hidden'), 3000);
}

loadConfig();
</script>
</body>
</html>`

type SettingsHandler struct {
	configPath string
}

func NewSettingsHandler() *SettingsHandler {
	cfgDir := os.Getenv("APPDATA")
	if cfgDir == "" {
		cfgDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	return &SettingsHandler{
		configPath: filepath.Join(cfgDir, "AgentNotify", "config.json"),
	}
}

func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SettingsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, settingsHTML)
}

func (h *SettingsHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Load existing config
	cfg, err := LoadConfig()
	if err != nil {
		cfg = DefaultConfig()
	}

	// Apply updates
	if style, ok := updates["notificationStyle"].(string); ok {
		if isValidStyle(style) {
			cfg.NotificationStyle = style
		}
	}
	if events, ok := updates["enabledEvents"].([]interface{}); ok {
		cfg.EnabledEvents = make([]string, 0, len(events))
		for _, e := range events {
			if s, ok := e.(string); ok && isValidEvent(s) {
				cfg.EnabledEvents = append(cfg.EnabledEvents, s)
			}
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(h.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, "Failed to create config directory", http.StatusInternalServerError)
		return
	}

	// Save config
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal config", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(h.configPath, data, 0644); err != nil {
		http.Error(w, "Failed to write config", http.StatusInternalServerError)
		return
	}

	log.Printf("Config saved to %s", h.configPath)
	w.WriteHeader(http.StatusNoContent)
}

// ConfigHandler returns current config as JSON for the settings UI
type ConfigHandler struct{}

func (h *ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := LoadConfig()
	if err != nil {
		cfg = DefaultConfig()
	}

	cfgMap := map[string]interface{}{
		"notificationStyle": cfg.NotificationStyle,
		"enabledEvents":      cfg.EnabledEvents,
		"futureOverrides":    cfg.FutureOverrides,
		"_path":              h.getConfigPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfgMap)
}

func (h *ConfigHandler) getConfigPath() string {
	cfgDir := os.Getenv("APPDATA")
	if cfgDir == "" {
		cfgDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	return filepath.Join(cfgDir, "AgentNotify", "config.json")
}

var validStyles = map[string]bool{
	"clean":        true,
	"status-color": true,
	"agent-badge":  true,
	"compact":      true,
}

var validEvents = map[string]bool{
	"start": true,
	"stop":  true,
}

func isValidStyle(style string) bool {
	return validStyles[style]
}

func isValidEvent(event string) bool {
	return validEvents[event]
}
