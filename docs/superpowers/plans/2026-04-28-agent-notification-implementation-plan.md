# Agent Notification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and debug a LAN-only Windows notification server in this repo, plus a separate Claude/Codex-compatible discovery skill in a sibling project folder.

**Architecture:** `/Users/<username>/project/agent-notification` contains only the Windows server source, scripts, and docs. The discovery skill lives outside this repo at `/Users/<username>/project/agent-notify-discovery-skill` so this repo stays focused on the Windows project. MVP runs from command line for debugging; no release executable or installer packaging is required yet.

**Tech Stack:** Go for the Windows server, mDNS/DNS-SD for automatic LAN discovery, OS-specific toast implementation behind a small interface, Python scripts for the skill.

---

## Current State and Constraints

- Existing draft implementation is in `windows-server/`.
- Existing spec is at `docs/superpowers/specs/2026-04-28-agent-notification-design.md`.
- Do not implement Codex hooks yet.
- Do design the sender/skill interfaces so Codex and other agents can be added later.
- Do not require a Windows `.exe` deliverable in MVP.
- Use `go run .` for local server debugging.
- Keep the skill in sibling folder `/Users/<username>/project/agent-notify-discovery-skill`.
- Current repo should contain only Windows server project files and docs.
- Discovery acceptance test must start a local server process, then run the installed skill discovery script and verify that this local server is found. Remote Windows LAN discovery is useful, but it is not enough for CI/debug acceptance because multicast may be blocked between machines.

Known issues from status check:

- `go test ./...` fails on macOS because `github.com/go-toast/toast` is Windows-specific and currently compiled on non-Windows.
- `go vet ./...` fails for the same reason.
- `POST /settings` JavaScript currently filters out the fields it needs to save.
- Discovery must be redesigned away from UDP broadcast and subnet scanning. Use one automatic discovery protocol only: mDNS/DNS-SD. Manual URL entry is the only fallback.
- `/manifest` does not yet include `url`, `supportedEvents`, and `supportedStyles`.
- Existing `windows-server/agent-notify-server.exe` should be removed from source control scope and ignored.

## Target Layout

Windows server repo:

```text
/Users/<username>/project/agent-notification/
  .gitignore
  docs/
    superpowers/
      specs/
      plans/
  windows-server/
    go.mod
    go.sum
    main.go
    config.go
    handlers.go
    manifest.go
    notify.go
    settings.go
    udp.go
    toast.go
    toast_windows.go
    toast_stub.go
    server_test.go
    config_test.go
    discovery_test.go
    scripts/
      start-dev.bat
      restart-dev.bat
      install-startup-dev.bat
      uninstall-startup-dev.bat
```

Sibling skill project:

```text
/Users/<username>/project/agent-notify-discovery-skill/
  SKILL.md
  scripts/
    discover.py
    send.py
    configure_claude.py
  references/
    hook-formats.md
```

## Task 1: Clean Server Build Boundary

**Files:**

- Modify: `windows-server/toast.go`
- Create: `windows-server/toast_windows.go`
- Create: `windows-server/toast_stub.go`
- Modify: `windows-server/go.mod`
- Modify: `.gitignore`

- [ ] **Step 1: Remove direct Windows toast dependency from cross-platform file**

Change `windows-server/toast.go` to define only the interface and constructor:

```go
package main

type Notifier interface {
	Notify(title, message string) error
}
```

- [ ] **Step 2: Add Windows toast implementation**

Create `windows-server/toast_windows.go`:

```go
//go:build windows

package main

import "github.com/go-toast/toast"

type ToastNotifier struct {
	appName string
}

func NewToastNotifier(appName string) Notifier {
	return &ToastNotifier{appName: appName}
}

func (n *ToastNotifier) Notify(title, message string) error {
	notification := toast.Notification{
		AppID:   n.appName,
		Title:   title,
		Message: message,
	}
	return notification.Push()
}
```

- [ ] **Step 3: Add non-Windows stub**

Create `windows-server/toast_stub.go`:

```go
//go:build !windows

package main

import "log"

type ToastNotifier struct {
	appName string
}

func NewToastNotifier(appName string) Notifier {
	return &ToastNotifier{appName: appName}
}

func (n *ToastNotifier) Notify(title, message string) error {
	log.Printf("[toast stub:%s] %s - %s", n.appName, title, message)
	return nil
}
```

- [ ] **Step 4: Update `Server` to depend on interface**

In `windows-server/handlers.go`, change:

```go
type Server struct {
	config   *Config
	notifier *ToastNotifier
}
```

to:

```go
type Server struct {
	config   *Config
	notifier Notifier
}
```

- [ ] **Step 5: Ignore generated binaries**

Add to `.gitignore`:

```gitignore
windows-server/*.exe
windows-server/*.log
```

Remove any generated executable from working tree tracking scope:

```bash
rm -f windows-server/agent-notify-server.exe
```

- [ ] **Step 6: Verify**

Run:

```bash
cd /Users/<username>/project/agent-notification/windows-server
go test ./...
go vet ./...
```

Expected:

```text
ok  	agent-notify-server
```

- [ ] **Step 7: Commit**

```bash
git add .gitignore windows-server/toast.go windows-server/toast_windows.go windows-server/toast_stub.go windows-server/handlers.go windows-server/go.mod windows-server/go.sum
git commit -m "fix: isolate windows toast implementation"
```

## Task 2: Replace UDP Discovery with mDNS/DNS-SD

**Files:**

- Modify: `windows-server/go.mod`
- Modify: `windows-server/handlers.go`
- Delete or stop using: `windows-server/udp.go`
- Create: `windows-server/mdns.go`
- Create: `windows-server/discovery_test.go`

**Decision:** Use mDNS/DNS-SD as the only automatic discovery protocol. Service type: `_agent-notify._tcp.local.`. Server advertises itself. Skill browses this service type and lists every server it sees. If mDNS fails, the only fallback is manual URL entry.

**Why this design:**

- UDP broadcast failed inside Claude Code/macOS with permission errors.
- Subnet scanning is slow, noisy, and timed out in real use.
- mDNS/DNS-SD is built for "many local services, browse by service type" workflows.
- Windows server can publish with a pure Go library; Mac and Windows clients can browse with Python `zeroconf` or platform tools.

- [ ] **Step 1: Add mDNS dependency**

Run:

```bash
cd /Users/<username>/project/agent-notification/windows-server
go get github.com/betamos/zeroconf
```

Rationale: this library is pure Go, supports publishing and browsing, and is documented as tested on Windows, macOS, and Linux.

- [ ] **Step 2: Add manifest response structs**

In `windows-server/handlers.go`, replace `ManifestResponse` with:

```go
type ManifestResponse struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	URL             string   `json:"url"`
	Hostname        string   `json:"hostname"`
	Protocol        string   `json:"protocol"`
	ServiceType     string   `json:"serviceType"`
	Description     string   `json:"description"`
	SupportedEvents []string `json:"supportedEvents"`
	SupportedStyles []string `json:"supportedStyles"`
}
```

- [ ] **Step 3: Add helper functions**

Add near manifest code:

```go
const mdnsServiceType = "_agent-notify._tcp"

func supportedEvents() []string {
	return []string{"start", "stop"}
}

func supportedStyles() []string {
	return []string{"clean", "status-color", "agent-badge", "compact"}
}

func localHostname() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "unknown"
	}
	return host
}
```

Add `os` import to `handlers.go`.

- [ ] **Step 4: Update `/manifest` response**

Build response like:

```go
resp := ManifestResponse{
	Name:            "Agent Notify Server",
	Version:         version,
	URL:             "http://" + r.Host,
	Hostname:        localHostname(),
	Protocol:        "mdns-dns-sd",
	ServiceType:     mdnsServiceType + ".local.",
	Description:     "Windows notification server for agent start/stop events",
	SupportedEvents: supportedEvents(),
	SupportedStyles: supportedStyles(),
}
```

- [ ] **Step 5: Add mDNS advertiser**

Create `windows-server/mdns.go`:

```go
package main

import (
	"context"
	"log"
	"strings"

	"github.com/betamos/zeroconf"
)

type DiscoveryTXT struct {
	Version string
	Events  string
	Styles  string
	Path    string
	Settings string
}

func discoveryTXT() []string {
	return []string{
		"version=" + version,
		"events=" + strings.Join(supportedEvents(), ","),
		"styles=" + strings.Join(supportedStyles(), ","),
		"path=/notify",
		"settings=/settings",
	}
}

func StartMDNSAdvertisement(ctx context.Context, port uint16) error {
	host := localHostname()
	instance := "Agent Notify " + host
	service := zeroconf.NewService(zeroconf.NewType(mdnsServiceType), instance, port)
	service.Text = discoveryTXT()

	client, err := zeroconf.New().
		Publish(service).
		Open()
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		if err := client.Close(); err != nil {
			log.Printf("mDNS close error: %v", err)
		}
	}()

	log.Printf("mDNS advertising %s as %q on port %d", mdnsServiceType, instance, port)
	return nil
}
```

- [ ] **Step 6: Start mDNS instead of UDP**

In `windows-server/main.go`, replace:

```go
go StartUDPDiscovery()
```

with:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := StartMDNSAdvertisement(ctx, 17891); err != nil {
	log.Printf("Warning: mDNS advertisement failed: %v", err)
}
```

Add `context` import.

Do not call `StartUDPDiscovery()` in MVP. Keep `udp.go` out of build or delete it after this task.

- [ ] **Step 7: Write discovery metadata test**

Create `windows-server/discovery_test.go`:

```go
package main

import (
	"strings"
	"testing"
)

func TestDiscoveryTXTContainsContractFields(t *testing.T) {
	txt := strings.Join(discoveryTXT(), "\n")
	for _, want := range []string{
		"version=",
		"events=start,stop",
		"styles=clean,status-color,agent-badge,compact",
		"path=/notify",
		"settings=/settings",
	} {
		if !strings.Contains(txt, want) {
			t.Fatalf("discovery TXT missing %q in %q", want, txt)
		}
	}

func TestMDNSServiceType(t *testing.T) {
	if mdnsServiceType != "_agent-notify._tcp" {
		t.Fatalf("mdnsServiceType = %q", mdnsServiceType)
	}
}
```

- [ ] **Step 8: Verify**

Run:

```bash
cd /Users/<username>/project/agent-notification/windows-server
go test ./...
go vet ./...
```

- [ ] **Step 9: Commit**

```bash
git add windows-server/go.mod windows-server/go.sum windows-server/handlers.go windows-server/main.go windows-server/mdns.go windows-server/discovery_test.go
git rm windows-server/udp.go
git commit -m "feat: advertise notification server with mdns"
```

## Task 3: Fix Settings Persistence and Preset Validation

**Files:**

- Modify: `windows-server/config.go`
- Modify: `windows-server/settings.go`
- Create: `windows-server/config_test.go`

- [ ] **Step 1: Add config validation**

In `windows-server/config.go`, add:

```go
func IsSupportedStyle(style string) bool {
	for _, supported := range []string{"clean", "status-color", "agent-badge", "compact"} {
		if style == supported {
			return true
		}
	}
	return false
}

func IsSupportedEvent(event string) bool {
	return event == "start" || event == "stop"
}

func (c *Config) Normalize() {
	if !IsSupportedStyle(c.NotificationStyle) {
		c.NotificationStyle = "clean"
	}
	if len(c.EnabledEvents) == 0 {
		c.EnabledEvents = []string{"start", "stop"}
		return
	}
	filtered := make([]string, 0, len(c.EnabledEvents))
	seen := map[string]bool{}
	for _, event := range c.EnabledEvents {
		if IsSupportedEvent(event) && !seen[event] {
			filtered = append(filtered, event)
			seen[event] = true
		}
	}
	c.EnabledEvents = filtered
	if c.FutureOverrides == nil {
		c.FutureOverrides = map[string]string{}
	}
}
```

Call `cfg.Normalize()` before returning from `LoadConfig()` and before saving in `settings.go`.

- [ ] **Step 2: Fix JavaScript save payload**

In `settingsHTML`, replace current `saveSettings()` with:

```javascript
async function saveSettings() {
  const saveData = {
    notificationStyle: config.notificationStyle,
    enabledEvents: config.enabledEvents,
    futureOverrides: config.futureOverrides || {}
  };

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
```

- [ ] **Step 3: Validate settings POST**

In `handlePost`, after applying updates:

```go
cfg.Normalize()
```

If the posted style is unsupported, either normalize to `clean` or return `400`. MVP should normalize to `clean` to keep UI recovery simple.

- [ ] **Step 4: Add config tests**

Create `windows-server/config_test.go`:

```go
package main

import "testing"

func TestConfigNormalizeUnsupportedStyle(t *testing.T) {
	cfg := &Config{
		NotificationStyle: "unknown",
		EnabledEvents:     []string{"start", "stop"},
	}
	cfg.Normalize()
	if cfg.NotificationStyle != "clean" {
		t.Fatalf("NotificationStyle = %q", cfg.NotificationStyle)
	}
}

func TestConfigNormalizeFiltersEvents(t *testing.T) {
	cfg := &Config{
		NotificationStyle: "compact",
		EnabledEvents:     []string{"start", "bad", "stop", "stop"},
	}
	cfg.Normalize()
	want := []string{"start", "stop"}
	if len(cfg.EnabledEvents) != len(want) {
		t.Fatalf("EnabledEvents = %#v", cfg.EnabledEvents)
	}
	for i := range want {
		if cfg.EnabledEvents[i] != want[i] {
			t.Fatalf("EnabledEvents = %#v", cfg.EnabledEvents)
		}
	}
}
```

- [ ] **Step 5: Verify**

Run:

```bash
cd /Users/<username>/project/agent-notification/windows-server
go test ./...
go vet ./...
```

- [ ] **Step 6: Commit**

```bash
git add windows-server/config.go windows-server/settings.go windows-server/config_test.go
git commit -m "fix: persist settings presets"
```

## Task 4: Make Command-Line Debug Flow First-Class

**Files:**

- Modify: `windows-server/main.go`
- Create: `windows-server/README.md`
- Modify: `windows-server/scripts/start-dev.bat`
- Modify: `windows-server/scripts/restart-dev.bat`

- [ ] **Step 1: Add env-configurable addresses**

In `main.go`, replace constant-only address use with:

```go
func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}
```

Use:

```go
httpAddr := envOrDefault("AGENT_NOTIFY_HTTP_ADDR", "0.0.0.0:17891")
```

Keep the mDNS service port aligned with the HTTP port. MVP can hardcode `17891` for mDNS advertisement if parsing `AGENT_NOTIFY_HTTP_ADDR` is not worth the complexity.

- [ ] **Step 2: Add debug README**

Create `windows-server/README.md`:

```markdown
# Agent Notify Windows Server

Run for development:

```powershell
go run .
```

Open settings:

```text
http://localhost:17891/settings
```

Test health:

```powershell
curl http://localhost:17891/health
```

Send test notification:

```powershell
curl -Method POST http://localhost:17891/notify `
  -ContentType "application/json" `
  -Body '{"agent":"claude","event":"stop","project":"agent-notification","message":"manual test"}'
```

LAN access:

```text
http://<windows-lan-ip>:17891/settings
```
```

- [ ] **Step 3: Rename scripts for dev mode**

Use `start-dev.bat`:

```bat
@echo off
cd /d "%~dp0.."
go run .
```

Use `restart-dev.bat`:

```bat
@echo off
taskkill /F /IM go.exe >nul 2>&1
timeout /t 1 /nobreak >nul
cd /d "%~dp0.."
go run .
```

Keep existing startup scripts if desired, but plan should not depend on exe packaging.

- [ ] **Step 4: Verify**

Run:

```bash
cd /Users/<username>/project/agent-notification/windows-server
go test ./...
```

Manual on Windows:

```powershell
cd C:\path\to\agent-notification\windows-server
go run .
```

Open:

```text
http://localhost:17891/settings
```

- [ ] **Step 5: Commit**

```bash
git add windows-server/main.go windows-server/README.md windows-server/scripts/start-dev.bat windows-server/scripts/restart-dev.bat
git commit -m "docs: add command line debug flow"
```

## Task 5: Create Separate Discovery Skill Project

**Files:**

- Create outside current repo: `/Users/<username>/project/agent-notify-discovery-skill/SKILL.md`
- Create: `/Users/<username>/project/agent-notify-discovery-skill/install-skill.sh`
- Create: `/Users/<username>/project/agent-notify-discovery-skill/install-skill.ps1`
- Create: `/Users/<username>/project/agent-notify-discovery-skill/scripts/discover.py`
- Create: `/Users/<username>/project/agent-notify-discovery-skill/scripts/send.py`
- Create: `/Users/<username>/project/agent-notify-discovery-skill/scripts/configure_claude.py`
- Create: `/Users/<username>/project/agent-notify-discovery-skill/scripts/smoke_discover_local.sh`
- Create: `/Users/<username>/project/agent-notify-discovery-skill/references/hook-formats.md`

- [ ] **Step 1: Create skill directories**

Run:

```bash
mkdir -p /Users/<username>/project/agent-notify-discovery-skill/scripts
mkdir -p /Users/<username>/project/agent-notify-discovery-skill/references
```

- [ ] **Step 2: Create install script**

Create `/Users/<username>/project/agent-notify-discovery-skill/install-skill.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST_DIR="${HOME}/.claude/skills/agent-notify-discovery"

mkdir -p "${HOME}/.claude/skills"
rm -rf "${DEST_DIR}"
cp -R "${SRC_DIR}" "${DEST_DIR}"
python3 -m venv "${DEST_DIR}/.venv"
"${DEST_DIR}/.venv/bin/python" -m pip install --upgrade pip
"${DEST_DIR}/.venv/bin/python" -m pip install zeroconf

echo "Installed agent-notify-discovery skill at ${DEST_DIR}"
```

Make it executable so `./install-skill.sh` does not fail with exit code `126`:

```bash
chmod +x /Users/<username>/project/agent-notify-discovery-skill/install-skill.sh
```

Also document the fallback command:

```bash
bash /Users/<username>/project/agent-notify-discovery-skill/install-skill.sh
```

Create `/Users/<username>/project/agent-notify-discovery-skill/install-skill.ps1` for Claude Code on Windows:

```powershell
$ErrorActionPreference = "Stop"

$src = Split-Path -Parent $MyInvocation.MyCommand.Path
$dest = Join-Path $HOME ".claude\skills\agent-notify-discovery"

New-Item -ItemType Directory -Force (Join-Path $HOME ".claude\skills") | Out-Null
if (Test-Path $dest) {
  Remove-Item -Recurse -Force $dest
}
Copy-Item -Recurse $src $dest

py -3 -m venv (Join-Path $dest ".venv")
& (Join-Path $dest ".venv\Scripts\python.exe") -m pip install --upgrade pip
& (Join-Path $dest ".venv\Scripts\python.exe") -m pip install zeroconf

Write-Host "Installed agent-notify-discovery skill at $dest"
```

- [ ] **Step 3: Create `SKILL.md`**

Create:

```markdown
---
name: agent-notify-discovery
description: Discover LAN Agent Notify Windows servers with mDNS/DNS-SD, choose a server, and configure Claude Code start/stop hooks. Use when users ask to set up agent task notifications to Windows.
---

# Agent Notify Discovery

Use this skill to discover the Windows Agent Notify server and configure agent hooks.

Workflow:

1. Run `scripts/discover.py` to find LAN servers.
2. Present every discovered `Agent Notify Server` instance.
3. If automatic discovery fails, ask user for `http://<windows-ip>:17891`.
4. Fetch `/manifest`.
5. Choose enabled events: `start`, `stop`, or both.
6. For Claude Code, run `scripts/configure_claude.py`.
7. Send a test notification with `scripts/send.py`.

MVP supports Claude Code only. Codex and other agents are future integrations; see `references/hook-formats.md`.
```

- [ ] **Step 4: Create `discover.py`**

Create a Python script that uses exactly one automatic discovery protocol: mDNS/DNS-SD. It browses `_agent-notify._tcp.local.` and prints every discovered server. It must not do UDP broadcast or subnet scanning. Manual URL entry is the only fallback.

Minimum behavior:

```python
#!/usr/bin/env python3
import argparse
import json
import socket
import sys
import time
import urllib.error
import urllib.request

from zeroconf import ServiceBrowser, ServiceListener, Zeroconf

SERVICE_TYPE = "_agent-notify._tcp.local."
HTTP_PORT = 17891

def fetch_manifest(url, timeout=2.0):
    manifest_url = url.rstrip("/") + "/manifest"
    req = urllib.request.Request(manifest_url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        payload = json.loads(resp.read().decode("utf-8"))
    payload.setdefault("url", url.rstrip("/"))
    return payload

def url_from_service_info(info):
    addresses = []
    for raw in info.addresses:
        try:
            addresses.append(socket.inet_ntoa(raw))
        except OSError:
            continue
    if not addresses:
        return None
    return f"http://{addresses[0]}:{info.port or HTTP_PORT}"

class AgentNotifyListener(ServiceListener):
    def __init__(self):
        self.results = {}

    def add_service(self, zeroconf, service_type, name):
        self._record(zeroconf, service_type, name)

    def update_service(self, zeroconf, service_type, name):
        self._record(zeroconf, service_type, name)

    def remove_service(self, zeroconf, service_type, name):
        self.results.pop(name, None)

    def _record(self, zeroconf, service_type, name):
        info = zeroconf.get_service_info(service_type, name, timeout=1500)
        if not info:
            return
        url = url_from_service_info(info)
        if not url:
            return
        props = {}
        for key, value in info.properties.items():
            key_text = key.decode("utf-8", errors="replace")
            value_text = value.decode("utf-8", errors="replace") if value else ""
            props[key_text] = value_text
        self.results[name] = {
            "name": name.rstrip("."),
            "url": url,
            "hostname": info.server.rstrip(".") if info.server else "",
            "serviceType": service_type,
            "properties": props,
        }

def discover_mdns(timeout=3.0):
    zeroconf = Zeroconf()
    listener = AgentNotifyListener()
    try:
        ServiceBrowser(zeroconf, SERVICE_TYPE, listener)
        time.sleep(timeout)
        return list(listener.results.values())
    finally:
        zeroconf.close()

def check_manual_url(host_or_url):
    if host_or_url.startswith("http://") or host_or_url.startswith("https://"):
        url = host_or_url
    else:
        url = f"http://{host_or_url}:{HTTP_PORT}"
    return [fetch_manifest(url, timeout=2.0)]

def discover(args):
    if args.manual:
        return check_manual_url(args.manual)
    return discover_mdns(timeout=args.timeout)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--timeout", type=float, default=2.0)
    parser.add_argument("--manual", help="Manual URL or host. This is the only fallback.")
    parser.add_argument("--json", action="store_true", help="Print JSON. Default is JSON too; kept for Claude Code prompts.")
    args = parser.parse_args()

    try:
        print(json.dumps(discover(args), indent=2))
    except Exception as err:
        print(f"Discovery failed: {err}", file=sys.stderr)
        print("[]")
```

- [ ] **Step 5: Create `send.py`**

Create a script that posts normalized payload to `/notify`, exits `0` on network failure by default, and supports strict mode.

Required CLI:

```bash
python scripts/send.py --url http://localhost:17891 --agent claude --event stop --project agent-notification
```

- [ ] **Step 6: Create `configure_claude.py`**

Create a script that writes Claude Code hooks for `SessionStart` and `Stop` using `send.py`.

MVP can emit the JSON snippet and ask user/agent to apply it if automatic settings editing is risky. Prefer not to overwrite existing hooks without merging.

- [ ] **Step 7: Create `hook-formats.md`**

Document:

```markdown
# Hook Formats

## Claude Code

`start` maps to `SessionStart`.
`stop` maps to `Stop`.

## Codex

Future work. Keep sender interface stable:

```bash
python scripts/send.py --url <url> --agent codex --event start
python scripts/send.py --url <url> --agent codex --event stop
```
```

- [ ] **Step 8: Verify discovery skill manually**

Create `/Users/<username>/project/agent-notify-discovery-skill/scripts/smoke_discover_local.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

SERVER_DIR="/Users/<username>/project/agent-notification/windows-server"
SKILL_DIR="/Users/<username>/project/agent-notify-discovery-skill"
LOG_FILE="${TMPDIR:-/tmp}/agent-notify-local-smoke.log"

cd "${SERVER_DIR}"
go run . >"${LOG_FILE}" 2>&1 &
SERVER_PID=$!

cleanup() {
  kill "${SERVER_PID}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

for _ in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:17891/health >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done

if ! curl -fsS http://127.0.0.1:17891/health >/dev/null 2>&1; then
  echo "server did not start; log follows" >&2
  cat "${LOG_FILE}" >&2
  exit 1
fi

cd "${SKILL_DIR}"
DISCOVER_OUTPUT="$(python3 scripts/discover.py --timeout 5)"
echo "${DISCOVER_OUTPUT}"

if ! printf '%s' "${DISCOVER_OUTPUT}" | grep -q '"serviceType": "_agent-notify._tcp.local."'; then
  echo "local mDNS discovery did not find local server" >&2
  cat "${LOG_FILE}" >&2
  exit 1
fi

if ! printf '%s' "${DISCOVER_OUTPUT}" | grep -q '17891'; then
  echo "local discovery output did not include port 17891" >&2
  cat "${LOG_FILE}" >&2
  exit 1
fi
```

Make it executable:

```bash
chmod +x /Users/<username>/project/agent-notify-discovery-skill/scripts/smoke_discover_local.sh
```

This smoke test is mandatory. It proves the server advertises mDNS locally and the skill can discover the advertised service without manual URL fallback.

Run:

```bash
cd /Users/<username>/project/agent-notify-discovery-skill
./install-skill.sh
./scripts/smoke_discover_local.sh
python3 scripts/discover.py
python3 scripts/discover.py --manual http://<windows-ip>:17891
python3 scripts/send.py --url http://localhost:17891 --agent claude --event stop --project manual-test --strict
```

Expected:

- `./scripts/smoke_discover_local.sh` starts the local Go server and exits `0` only if `scripts/discover.py` finds it through mDNS.
- `python3 scripts/discover.py` returns every mDNS-advertised Agent Notify server.
- No subnet scan happens.
- No UDP broadcast happens.
- `--manual http://<windows-ip>:17891` validates and returns one manually supplied server.

- [ ] **Step 9: Do not commit skill in Windows repo**

The sibling skill folder is not part of `/Users/<username>/project/agent-notification`. If a separate git repo is desired, initialize it in `/Users/<username>/project/agent-notify-discovery-skill`.

## Task 6: Final Verification Checklist

**Files:**

- Modify only if needed: `windows-server/README.md`
- Modify only if needed: `docs/superpowers/specs/2026-04-28-agent-notification-design.md`

- [ ] **Step 1: Verify current repo contains only Windows project**

Run:

```bash
cd /Users/<username>/project/agent-notification
find . -maxdepth 2 -type d | sort
```

Expected: no `skills/` directory in this repo.

- [ ] **Step 2: Verify server tests**

Run:

```bash
cd /Users/<username>/project/agent-notification/windows-server
go test ./...
go vet ./...
```

- [ ] **Step 3: Verify dev run**

Run on Windows:

```powershell
cd C:\path\to\agent-notification\windows-server
go run .
```

Open:

```text
http://localhost:17891/settings
```

- [ ] **Step 4: Verify local discovery smoke test**

Run from Mac:

```bash
cd /Users/<username>/project/agent-notify-discovery-skill
./scripts/smoke_discover_local.sh
```

Expected: the script starts `/Users/<username>/project/agent-notification/windows-server` with `go run .`, waits for `http://127.0.0.1:17891/health`, runs `python3 scripts/discover.py --timeout 5`, and finds the local mDNS service.

- [ ] **Step 5: Verify LAN discovery from Mac to Windows if network permits**

Run from Mac:

```bash
cd /Users/<username>/project/agent-notify-discovery-skill
python3 scripts/discover.py
```

Expected: JSON array includes all reachable Windows servers advertising `_agent-notify._tcp.local.`. If this fails while the local smoke test passes, treat it as a LAN multicast/network reachability issue, not a skill/server implementation failure.

- [ ] **Step 6: Verify notification**

Run from Mac:

```bash
python3 scripts/send.py --url http://<windows-ip>:17891 --agent claude --event stop --project agent-notification --strict
```

Expected: Windows desktop toast appears using selected preset style.

- [ ] **Step 7: Commit Windows repo changes**

```bash
cd /Users/<username>/project/agent-notification
git status --short
git add .gitignore docs/superpowers windows-server
git commit -m "feat: add windows notification server"
```

## Self-Review

- Spec coverage: Windows server, mDNS/DNS-SD discovery, settings UI, style presets, restart/dev scripts, and future agent extensibility are covered.
- Scope adjustment: skill project is explicitly outside this repo at `/Users/<username>/project/agent-notify-discovery-skill`.
- MVP adjustment: no release executable required; command-line `go run .` debug flow is primary.
- Discovery acceptance adjusted: local server + skill discovery smoke test is mandatory; remote LAN multicast discovery is an additional environment check.
- Known implementation risks are captured as tasks.
- Codex hook implementation remains out of scope.
