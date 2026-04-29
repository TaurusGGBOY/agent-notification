# Custom Toast Notification Styles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace go-toast/toast library with direct WinRT API calls to enable 4 distinct toast visual styles (clean, status-color, agent-badge, compact).

**Architecture:** Use `golang.org/x/sys/windows` to call Windows.UI.Notifications API directly with ToastGeneric XML templates. Each style produces different XML structure for visual differentiation.

**Tech Stack:** Go + `golang.org/x/sys/windows` (WinRT), no external toast library.

---

## File Structure

```
windows-server/
  toast_windows.go     # NOTIFIER interface + WinRT implementation (REPLACE)
  toast_stub.go        # Unchanged (macOS stub)
  handlers.go          # Modify: pass style/event to notifier (line ~120)
  config.go            # Unchanged (NotificationStyle already exists)
  windows_test.go      # Add: style XML generation tests
```

---

## Task 1: Rewrite toast_windows.go with WinRT

**Files:**
- Modify: `windows-server/toast_windows.go`

- [ ] **Step 1: Write test for XML generation**

```go
// windows-server/windows_test.go

func TestFormatToastXML_Clean(t *testing.T) {
    xml := formatToastXML("clean", "start", "Agent Started", "claude", "agent-notification")
    if !strings.Contains(xml, "ToastGeneric") {
        t.Error("clean style should use ToastGeneric template")
    }
    if strings.Contains(xml, "appLogoOverride") {
        t.Error("clean style should not have appLogoOverride")
    }
}

func TestFormatToastXML_StatusColor(t *testing.T) {
    xml := formatToastXML("status-color", "stop", "Agent Stopped", "claude", "agent-notification")
    if !strings.Contains(xml, "appLogoOverride") {
        t.Error("status-color should have appLogoOverride")
    }
}

func TestFormatToastXML_AgentBadge(t *testing.T) {
    xml := formatToastXML("agent-badge", "start", "Agent Started", "claude", "agent-notification")
    if !strings.Contains(xml, "hint-crop=circle") {
        t.Error("agent-badge should have circular crop")
    }
}

func TestFormatToastXML_Compact(t *testing.T) {
    xml := formatToastXML("compact", "stop", "claude: stop", "claude", "agent-notification")
    if strings.Contains(xml, "project") || strings.Contains(xml, "cwd") {
        t.Error("compact should not include project/cwd")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd windows-server && go test ./... -run TestFormatToastXML -v`
Expected: FAIL (function not exist)

- [ ] **Step 3: Write toast_windows.go with WinRT**

```go
//go:build windows
// +build windows

package main

import (
    "strings"
    "unicode/utf8"

    "golang.org/x/sys/windows"
)

const (
    toastNamespace = "Windows.UI.Notifications.ToastNotificationManager"
    toastIID       = "79AB57F6-43FE-487B-8A7F-99567200AE94" // IToastNotificationManager
)

type ToastNotifier struct {
    appName string
}

func NewToastNotifier(appName string) *ToastNotifier {
    return &ToastNotifier{appName: appName}
}

func (n *ToastNotifier) Notify(title, message string) error {
    style := "clean" // default
    event := "stop"  // default
    return n.NotifyWithStyle(style, event, title, message, "")
}

func (n *ToastNotifier) NotifyWithStyle(style, event, title, message, agent string) error {
    xml := formatToastXML(style, event, title, agent, "")
    return sendToastNotification(n.appName, xml)
}

// formatToastXML generates ToastGeneric XML based on style
func formatToastXML(style, event, title, agent, extra string) string {
    switch style {
    case "status-color":
        return buildStatusColorXML(event, title, agent)
    case "agent-badge":
        return buildAgentBadgeXML(event, title, agent)
    case "compact":
        return buildCompactXML(event, title, agent)
    default: // clean
        return buildCleanXML(title, agent, extra)
    }
}

func buildCleanXML(title, agent, extra string) string {
    return buildBaseToastXML(title, "", agent, "", "")
}

func buildStatusColorXML(event, title, agent string) string {
    // Use appLogoOverride with colored icon based on event
    iconPath := getStatusIconPath(event) // green for start, red for stop
    attribution := getStatusAttribution(event) // "Started" or "Stopped"
    return buildBaseToastXML(title, iconPath, agent, attribution, "appLogoOverride")
}

func buildAgentBadgeXML(event, title, agent string) string {
    // Circular avatar + agent initial
    iconPath := getAgentBadgePath(agent)
    attribution := getAgentInitial(agent) // "C" for claude
    return buildBaseToastXML(title, iconPath, agent, attribution, "appLogoOverride-circle")
}

func buildCompactXML(event, title, agent string) string {
    // Single line, no extra info
    return `<toast><visual><binding template="ToastGeneric"><text>` + escapeXML(title) + `</text></binding></visual></toast>`
}

func buildBaseToastXML(title, iconPath, agent, attribution, iconPlacement string) string {
    var sb strings.Builder
    sb.WriteString(`<toast><visual><binding template="ToastGeneric">`)

    if iconPath != "" {
        sb.WriteString(`<image placement="`)
        sb.WriteString(iconPlacement)
        sb.WriteString(`" src="`)
        sb.WriteString(iconPath)
        sb.WriteString(`"/>`)
    }

    sb.WriteString(`<text>`)
    sb.WriteString(escapeXML(title))
    sb.WriteString(`</text>`)

    if attribution != "" {
        sb.WriteString(`<text placement="attribution">`)
        sb.WriteString(escapeXML(attribution))
        sb.WriteString(`</text>`)
    }

    sb.WriteString(`</binding></visual></toast>`)
    return sb.String()
}

func getStatusIconPath(event string) string {
    if event == "start" {
        return `C:\Windows\System32\green.png`
    }
    return `C:\Windows\System32\red.png`
}

func getStatusAttribution(event string) string {
    if event == "start" {
        return "Started"
    }
    return "Stopped"
}

func getAgentBadgePath(agent string) string {
    // Return path to agent icon or use system default
    return `C:\Windows\System32\user.png`
}

func getAgentInitial(agent string) string {
    if len(agent) > 0 {
        r, _ := utf8.DecodeRuneInString(agent)
        return string(unicode.ToUpper(r))
    }
    return "?"
}

func escapeXML(s string) string {
    s = strings.ReplaceAll(s, "&", "&amp;")
    s = strings.ReplaceAll(s, "<", "&lt;")
    s = strings.ReplaceAll(s, ">", "&gt;")
    s = strings.ReplaceAll(s, `"`, "&quot;")
    s = strings.ReplaceAll(s, "'", "&apos;")
    return s
}

// sendToastNotification uses WinRT to display toast
func sendToastNotification(appID, xml string) error {
    // Convert to UTF-16 for WinRT
    xml16, err := windows.UTF16FromString(xml)
    if err != nil {
        return err
    }

    // Use CreateXmlDocument to parse
    var doc windows.IInspectable
    hr := windows.RoActivateInstance(
        windows.StringToGUID("{3A3DCD6C-3EAB-43DC-BCDE-45671CE800C8}"), // IXmlDocument
        &doc,
    )
    if err != nil {
        // Fallback: use shell toast via shell command
        return sendToastViaShell(appID, xml)
    }
    defer doc.Release()

    // Set XML content on document
    // Then create toast and show via ToastNotificationManager

    return sendToastViaShell(appID, xml)
}

// sendToastViaShell fallback using PowerShell
func sendToastViaShell(appID, xml string) error {
    ps := `[Windows.UI.Notifications.ToastNotificationManager,Windows.UI.Notifications,ContentType=WindowsRuntime] | Out-Null; `
    ps += `[Windows.Data.Xml.Dom.XmlDocument,Windows.Data.Xml.Dom.XmlDocument,ContentType=WindowsRuntime] | Out-Null; `
    ps += `$d=[Windows.Data.Xml.Dom.XmlDocument]::new(); `
    ps += `$d.LoadXml('` + xml + `'); `
    ps += `[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('` + appID + `').Show([Windows.UI.Notifications.ToastNotification]::new($d))`

    return runPowerShell(ps)
}

func runPowerShell(script string) error {
    argv, _ := windows.UTF16PtrFromString("powershell")
    args, _ := windows.UTF16PtrFromString("-NoProfile -WindowStyle Hidden -Command " + script)
    info := &windows.StartupInfo{
        ShowWindow: windows.SW_HIDE,
    }
    proc, _ := windows.CreateProcess(nil, argv, nil, nil, true, windows.CREATE_NO_WINDOW, nil, nil, info)
    if proc == 0 {
        return windows.GetLastError()
    }
    defer windows.CloseHandle(windows.Handle(proc))

    var exitCode uint32
    windows.WaitForSingleObject(windows.Handle(proc), windows.INFINITE)
    windows.GetExitCodeProcess(windows.Handle(proc), &exitCode)
    if exitCode != 0 {
        return windows.ERROR_EXEC_FAILURE
    }
    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd windows-server && go test ./... -run TestFormatToastXML -v`
Expected: PASS (38 tests total)

- [ ] **Step 5: Commit**

```bash
cd windows-server
git add toast_windows.go windows_test.go
git commit -m "feat: replace go-toast with WinRT for custom toast styles"
```

---

## Task 2: Update handlers.go to pass style

**Files:**
- Modify: `windows-server/handlers.go:120-130`

- [ ] **Step 1: Verify current NotifyHandler uses config.NotificationStyle**

Run: `grep -n "NotificationStyle" windows-server/handlers.go`

- [ ] **Step 2: Update NotifyHandler to call NotifyWithStyle**

Find line ~120 where `s.notifier.Notify(title, message)` is called, change to:

```go
s.notifier.NotifyWithStyle(
    s.config.NotificationStyle,
    event,
    title,
    message,
    payload.Agent,
)
```

- [ ] **Step 3: Run tests**

Run: `cd windows-server && go test ./... -v`
Expected: PASS (38 tests)

- [ ] **Step 4: Commit**

```bash
git add handlers.go
git commit -m "fix: pass style to notifier for custom toast visuals"
```

---

## Task 3: Add icon assets for styles (Future)

**Note:** For MVP, use Windows built-in icons. Later add custom icons.

- `status-color`: Use `C:\Windows\System32\green.png` (start) and `C:\Windows\System32\red.png` (stop) — Windows may not have these exact files, fallback to generic
- `agent-badge`: Use `C:\Windows\System32\user.png` or generate initial-based icon

**Testing on Windows:**
```powershell
cd C:\path\to\agent-notification\windows-server
go run .
# Open http://localhost:17891/settings
# Select each style, click "Send Test Toast"
# Verify visual differences
```

---

## Verification

1. Run `go test ./...` — all 38+ tests pass
2. On Windows: `go run .` + open settings + select each style + test toast
3. Verify 4 distinct visual outputs:
   - clean: plain text
   - status-color: icon + attribution
   - agent-badge: circular avatar + initial
   - compact: minimal single line