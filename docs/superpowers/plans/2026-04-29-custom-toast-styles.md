# Custom Toast Card Styles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the limited `go-toast/toast` formatting with Windows ToastGeneric XML that can show a generated PNG card as a hero image, preserving the normal Windows notification flow while allowing custom visual UI.

**Architecture:** Keep `/notify`, config, settings UI, Windows Action Center, focus assist, and system toast behavior. Generate a local PNG card for rich custom UI, reference it from ToastGeneric XML as a hero image, and send the toast through a small PowerShell WinRT bridge. Keep XML/card generation in cross-platform Go files so tests run on macOS and Windows.

**Tech Stack:** Go, standard library `image`/`image/png`, `golang.org/x/image/font/basicfont`, Windows ToastGeneric XML, PowerShell WinRT bridge.

---

## Key Decisions

- Windows toast supports images, including inline images, `appLogoOverride`, circular crop, and hero images.
- Use a generated hero image for the most customizable UI. Recommended 100% scale hero size is `364x180`.
- Do not try to build direct WinRT COM calls with `golang.org/x/sys/windows` in this plan. The current Go package does not expose the high-level WinRT helpers used in the old draft.
- For unpackaged desktop apps, keep images local. Use an absolute local file path in XML for desktop toast compatibility.
- Preserve native toast text in the XML for fallback and notification center readability, even when the custom card image is present.
- Keep `clean`, `status-color`, `agent-badge`, and `compact` as lightweight native XML presets. Put the fully custom UI in the new `custom-card` style.

## File Structure

```
windows-server/
  toast_xml.go          # Cross-platform ToastGeneric XML generation and XML escaping.
  toast_card.go         # Cross-platform PNG card generation for custom-card style.
  toast_windows.go      # Windows-only sender using PowerShell WinRT bridge.
  toast_stub.go         # Non-Windows stub; unchanged unless signature drift appears.
  handlers.go           # Already passes style/event/agent; verify only.
  config.go             # Add custom-card to supported styles.
  settings.go           # Add custom-card option and preview.
  mdns.go               # Uses supportedStyles(); no direct change expected.
  windows_test.go       # Add/update XML, card, config, and settings tests.
  go.mod/go.sum         # Remove go-toast; add golang.org/x/image if needed.
```

---

## Task 1: Move Toast XML Generation Into Cross-Platform Code

**Files:**
- Create: `windows-server/toast_xml.go`
- Modify: `windows-server/windows_test.go`

- [ ] **Step 1: Replace XML tests with hero-card coverage**

Replace the existing XML generation tests in `windows-server/windows_test.go` with these tests. The function signature becomes `formatToastXML(style, event, title, agent, project, cardImagePath string)`.

```go
func TestFormatToastXML_Clean(t *testing.T) {
	xml := formatToastXML("clean", "start", "Agent Started", "claude", "agent-notification", "")
	if !strings.Contains(xml, `template="ToastGeneric"`) {
		t.Error("clean style should use ToastGeneric template")
	}
	if strings.Contains(xml, `placement="hero"`) {
		t.Error("clean style should not use hero image")
	}
}

func TestFormatToastXML_StatusColor(t *testing.T) {
	xml := formatToastXML("status-color", "stop", "Agent Stopped", "claude", "agent-notification", "")
	if !strings.Contains(xml, `placement="attribution"`) {
		t.Error("status-color should have attribution text")
	}
	if !strings.Contains(xml, "Stopped") {
		t.Error("status-color should include stopped attribution")
	}
}

func TestFormatToastXML_AgentBadge(t *testing.T) {
	xml := formatToastXML("agent-badge", "start", "Agent Started", "claude", "agent-notification", "")
	if !strings.Contains(xml, "Agent C") {
		t.Error("agent-badge should include agent initial attribution")
	}
}

func TestFormatToastXML_Compact(t *testing.T) {
	xml := formatToastXML("compact", "stop", "claude: stop", "claude", "agent-notification", "")
	if strings.Contains(xml, "agent-notification") {
		t.Error("compact should not include project")
	}
	if strings.Contains(xml, `placement="hero"`) {
		t.Error("compact should not use hero image")
	}
}

func TestFormatToastXML_CustomCardUsesHeroImage(t *testing.T) {
	xml := formatToastXML("custom-card", "start", "Agent Started", "claude", "agent-notification", `C:\Users\me\AppData\Local\AgentNotify\toast-card.png`)
	if !strings.Contains(xml, `template="ToastGeneric"`) {
		t.Error("custom-card should use ToastGeneric template")
	}
	if !strings.Contains(xml, `placement="hero"`) {
		t.Error("custom-card should use a hero image")
	}
	if !strings.Contains(xml, `toast-card.png`) {
		t.Error("custom-card should include generated card image path")
	}
	if !strings.Contains(xml, "Agent Started") {
		t.Error("custom-card should keep native fallback text")
	}
}

func TestFormatToastXML_EscapesFields(t *testing.T) {
	xml := formatToastXML("clean", "start", `Agent <Started> & "Ready"`, "claude", "a'b", "")
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd windows-server
go test ./... -run 'TestFormatToastXML|TestEscapeXML' -v
```

Expected: FAIL with `undefined: formatToastXML` and `undefined: escapeXML`.

- [ ] **Step 3: Add cross-platform XML generation**

Create `windows-server/toast_xml.go`:

```go
package main

import (
	"path/filepath"
	"strings"
)

func formatToastXML(style, event, title, agent, project, cardImagePath string) string {
	switch style {
	case "status-color":
		return buildStatusColorXML(event, title, project)
	case "agent-badge":
		return buildAgentBadgeXML(event, title, agent, project)
	case "compact":
		return buildCompactXML(title)
	case "custom-card":
		return buildCustomCardXML(title, project, cardImagePath)
	default:
		return buildCleanXML(title, project)
	}
}

func buildCleanXML(title, project string) string {
	return buildToastXML(title, project, "", "", "", "")
}

func buildStatusColorXML(event, title, project string) string {
	status := "Stopped"
	if event == "start" {
		status = "Started"
	}
	return buildToastXML(title, project, "", "", "", status)
}

func buildAgentBadgeXML(event, title, agent, project string) string {
	_ = event
	initial := agentInitial(agent)
	return buildToastXML(title, project, "", "", "", "Agent "+initial)
}

func buildCompactXML(title string) string {
	return `<toast><visual><binding template="ToastGeneric"><text>` + escapeXML(title) + `</text></binding></visual></toast>`
}

func buildCustomCardXML(title, project, cardImagePath string) string {
	return buildToastXML(title, project, cardImagePath, "hero", "", "")
}

func buildToastXML(title, project, imagePath, placement, crop, attribution string) string {
	var sb strings.Builder
	sb.WriteString(`<toast><visual><binding template="ToastGeneric">`)
	sb.WriteString(`<text>`)
	sb.WriteString(escapeXML(title))
	sb.WriteString(`</text>`)
	if project != "" {
		sb.WriteString(`<text>`)
		sb.WriteString(escapeXML(project))
		sb.WriteString(`</text>`)
	}
	if attribution != "" {
		sb.WriteString(`<text placement="attribution">`)
		sb.WriteString(escapeXML(attribution))
		sb.WriteString(`</text>`)
	}
	if imagePath != "" {
		sb.WriteString(`<image`)
		if placement != "" {
			sb.WriteString(` placement="`)
			sb.WriteString(escapeXML(placement))
			sb.WriteString(`"`)
		}
		sb.WriteString(` src="`)
		sb.WriteString(escapeXML(normalizeToastImagePath(imagePath)))
		sb.WriteString(`"`)
		if crop != "" {
			sb.WriteString(` hint-crop="`)
			sb.WriteString(escapeXML(crop))
			sb.WriteString(`"`)
		}
		sb.WriteString(`/>`)
	}
	sb.WriteString(`</binding></visual></toast>`)
	return sb.String()
}

func normalizeToastImagePath(path string) string {
	if path == "" || strings.Contains(path, "://") {
		return path
	}
	return filepath.Clean(path)
}

func agentInitial(agent string) string {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		return "?"
	}
	return strings.ToUpper(string([]rune(agent)[0]))
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
```

- [ ] **Step 4: Run XML tests**

Run:

```bash
cd windows-server
go test ./... -run 'TestFormatToastXML|TestEscapeXML' -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add windows-server/toast_xml.go windows-server/windows_test.go
git commit -m "feat: generate toast xml for custom card styles"
```

---

## Task 2: Generate Custom Toast Card PNG

**Files:**
- Create: `windows-server/toast_card.go`
- Modify: `windows-server/windows_test.go`
- Modify: `windows-server/go.mod`

- [ ] **Step 1: Add card rendering tests**

Add these tests to `windows-server/windows_test.go`:

```go
func TestRenderToastCardCreatesPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "toast-card.png")
	card := ToastCard{
		Event:   "start",
		Title:   "Agent Start: claude",
		Agent:   "claude",
		Project: "agent-notification",
		Message: "Task running",
	}
	if err := renderToastCard(path, card); err != nil {
		t.Fatalf("renderToastCard failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read png failed: %v", err)
	}
	if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("file is not a png")
	}
}

func TestToastCardPathUsesLocalAppData(t *testing.T) {
	tmp := t.TempDir()
	oldLocalAppData := os.Getenv("LOCALAPPDATA")
	os.Setenv("LOCALAPPDATA", tmp)
	defer os.Setenv("LOCALAPPDATA", oldLocalAppData)

	path, err := toastCardPath()
	if err != nil {
		t.Fatalf("toastCardPath failed: %v", err)
	}
	if !strings.Contains(path, filepath.Join(tmp, "AgentNotify")) {
		t.Fatalf("card path should be under local app data, got %s", path)
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("card path should be png, got %s", path)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd windows-server
go test ./... -run 'TestRenderToastCard|TestToastCardPath' -v
```

Expected: FAIL with undefined `ToastCard`, `renderToastCard`, and `toastCardPath`.

- [ ] **Step 3: Add image dependency**

Run:

```bash
cd windows-server
go get golang.org/x/image@latest
```

Expected: `go.mod` includes `golang.org/x/image`.

- [ ] **Step 4: Add PNG renderer**

Create `windows-server/toast_card.go`:

```go
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	toastCardWidth  = 364
	toastCardHeight = 180
)

type ToastCard struct {
	Event   string
	Title   string
	Agent   string
	Project string
	Message string
}

func toastCardPath() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = filepath.Join(os.TempDir(), "AgentNotify")
	} else {
		base = filepath.Join(base, "AgentNotify")
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", err
	}
	return filepath.Join(base, "toast-card.png"), nil
}

func renderToastCard(path string, card ToastCard) error {
	img := image.NewRGBA(image.Rect(0, 0, toastCardWidth, toastCardHeight))
	bg := color.RGBA{R: 18, G: 24, B: 38, A: 255}
	accent := color.RGBA{R: 74, G: 222, B: 128, A: 255}
	if card.Event == "stop" {
		accent = color.RGBA{R: 248, G: 113, B: 113, A: 255}
	}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, 8, toastCardHeight), &image.Uniform{C: accent}, image.Point{}, draw.Src)
	drawCircle(img, 44, 44, 24, accent)
	drawText(img, 36, 49, agentInitial(card.Agent), color.RGBA{R: 15, G: 23, B: 42, A: 255})
	drawText(img, 84, 37, truncateText(card.Title, 38), color.RGBA{R: 248, G: 250, B: 252, A: 255})
	drawText(img, 84, 65, truncateText(card.Project, 42), color.RGBA{R: 148, G: 163, B: 184, A: 255})
	drawText(img, 24, 112, truncateText(card.Message, 52), color.RGBA{R: 203, G: 213, B: 225, A: 255})
	drawText(img, 24, 148, strings.ToUpper(card.Event), accent)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func drawText(img *image.RGBA, x, y int, text string, c color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.Color) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r && image.Pt(x, y).In(img.Bounds()) {
				img.Set(x, y, c)
			}
		}
	}
}

func truncateText(s string, max int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
```

- [ ] **Step 5: Run card tests**

Run:

```bash
cd windows-server
go test ./... -run 'TestRenderToastCard|TestToastCardPath' -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add windows-server/toast_card.go windows-server/windows_test.go windows-server/go.mod windows-server/go.sum
git commit -m "feat: render custom toast card image"
```

---

## Task 3: Replace go-toast With PowerShell WinRT Sender

**Files:**
- Modify: `windows-server/toast_windows.go`
- Modify: `windows-server/go.mod`
- Modify: `windows-server/go.sum`

- [ ] **Step 1: Replace Windows toast sender**

Replace `windows-server/toast_windows.go` with:

```go
//go:build windows
// +build windows

package main

import (
	"encoding/base64"
	"log"
	"os/exec"
)

type ToastNotifier struct {
	appName string
}

func NewToastNotifier(appName string) *ToastNotifier {
	return &ToastNotifier{appName: appName}
}

func (n *ToastNotifier) Notify(title, message string) error {
	return n.NotifyWithStyle("clean", "stop", title, message, "")
}

func (n *ToastNotifier) NotifyWithStyle(style, event, title, message, agent string) error {
	project := ""
	cardPath := ""
	if style == "custom-card" {
		path, err := toastCardPath()
		if err != nil {
			return err
		}
		card := ToastCard{
			Event:   event,
			Title:   title,
			Agent:   agent,
			Project: project,
			Message: message,
		}
		if err := renderToastCard(path, card); err != nil {
			return err
		}
		cardPath = path
	}
	xml := formatToastXML(style, event, title, agent, project, cardPath)
	if err := sendToastViaPowerShell(n.appName, xml); err != nil {
		log.Printf("Toast notification failed: %v", err)
		return err
	}
	return nil
}

func sendToastViaPowerShell(appID, xml string) error {
	encodedXML := base64.StdEncoding.EncodeToString([]byte(xml))
	script := `
$xml = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('` + encodedXML + `'))
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$doc = [Windows.Data.Xml.Dom.XmlDocument]::new()
$doc.LoadXml($xml)
$toast = [Windows.UI.Notifications.ToastNotification]::new($doc)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('` + escapePowerShellSingleQuoted(appID) + `').Show($toast)
`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-Command", script)
	return cmd.Run()
}

func escapePowerShellSingleQuoted(s string) string {
	out := ""
	for _, r := range s {
		if r == '\'' {
			out += "''"
		} else {
			out += string(r)
		}
	}
	return out
}
```

- [ ] **Step 2: Remove go-toast dependency**

Run:

```bash
cd windows-server
go mod tidy
```

Expected: `github.com/go-toast/toast` removed from `go.mod` and `go.sum`.

- [ ] **Step 3: Run cross-platform tests**

Run:

```bash
cd windows-server
go test ./... -v
```

Expected: PASS on macOS/non-Windows. The Windows-only file is not compiled there, but XML/card/config tests still run.

- [ ] **Step 4: Cross-compile Windows binary**

Run:

```bash
cd windows-server
GOOS=windows GOARCH=amd64 go build -o agent-notify-server.exe
```

Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add windows-server/toast_windows.go windows-server/go.mod windows-server/go.sum
git commit -m "feat: send custom toast xml through powershell"
```

---

## Task 4: Add `custom-card` To Config, Manifest, Settings UI, And Discovery

**Files:**
- Modify: `windows-server/config.go`
- Modify: `windows-server/handlers.go`
- Modify: `windows-server/settings.go`
- Modify: `windows-server/windows_test.go`
- Modify: `windows-server/discovery_test.go`

- [ ] **Step 1: Update style validation tests**

Update style lists in tests from:

```go
[]string{"clean", "status-color", "agent-badge", "compact"}
```

to:

```go
[]string{"clean", "status-color", "agent-badge", "compact", "custom-card"}
```

Update the manifest expectation to include `custom-card`.

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd windows-server
go test ./... -run 'TestIsValidStyle|TestManifestHandler|TestDiscoveryTXTRecords' -v
```

Expected: FAIL because production style lists do not include `custom-card`.

- [ ] **Step 3: Add `custom-card` to server style lists**

In `windows-server/config.go`, change `IsSupportedStyle` to:

```go
func IsSupportedStyle(style string) bool {
	for _, supported := range []string{"clean", "status-color", "agent-badge", "compact", "custom-card"} {
		if style == supported {
			return true
		}
	}
	return false
}
```

In `windows-server/handlers.go`, change `supportedStyles` to:

```go
func supportedStyles() []string {
	return []string{"clean", "status-color", "agent-badge", "compact", "custom-card"}
}
```

In `windows-server/settings.go`, update `validStyles`:

```go
var validStyles = map[string]bool{
	"clean":        true,
	"status-color": true,
	"agent-badge":  true,
	"compact":      true,
	"custom-card":  true,
}
```

- [ ] **Step 4: Add settings UI card and preview CSS**

Add a fifth preset card in `settingsHTML`:

```html
<div class="preset-card" data-style="custom-card" onclick="selectStyle('custom-card')">
<h3>Custom Card</h3>
<p>Generated image card inside native Windows toast</p>
</div>
```

Add preview CSS:

```css
.toast-preview.custom-card{background:#121826;border-left:8px solid #4ade80;padding:1rem 1rem 1rem 1.25rem}
.toast-preview.custom-card.stop{border-left-color:#f87171}
```

Update `updatePreview()` so `custom-card` uses a richer preview:

```javascript
} else if (style === 'custom-card') {
  previewStart.className = 'toast-preview custom-card';
  previewStop.className = 'toast-preview custom-card stop';
  previewStart.innerHTML = '<div class="toast-title">Agent Start: claude</div><div class="toast-message">agent-notification</div><div class="toast-message">Custom hero card image</div>';
  previewStop.innerHTML = '<div class="toast-title">Agent Stop: claude</div><div class="toast-message">agent-notification</div><div class="toast-message">Custom hero card image</div>';
```

- [ ] **Step 5: Run tests**

Run:

```bash
cd windows-server
go test ./... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add windows-server/config.go windows-server/handlers.go windows-server/settings.go windows-server/windows_test.go windows-server/discovery_test.go
git commit -m "feat: expose custom card notification style"
```

---

## Task 5: Windows Manual Verification

**Files:**
- No source changes expected.

- [ ] **Step 1: Build on Windows or cross-compile**

Run:

```powershell
cd C:\path\to\agent-notification\windows-server
go build -o agent-notify-server.exe
```

Expected: `agent-notify-server.exe` exists.

- [ ] **Step 2: Start server**

Run:

```powershell
.\agent-notify-server.exe
```

Expected: server listens on `http://localhost:17891`.

- [ ] **Step 3: Select `custom-card`**

Open:

```text
http://localhost:17891/settings
```

Select `Custom Card`, click `Save Settings`, then click `Send Test Toast`.

Expected:
- Windows native toast appears.
- Toast includes generated card image.
- Notification still appears through normal Windows toast flow.

- [ ] **Step 4: Verify generated file**

Run:

```powershell
dir "$env:LOCALAPPDATA\AgentNotify\toast-card.png"
```

Expected: PNG exists and updates when sending a test toast.

- [ ] **Step 5: Verify fallback styles**

In settings UI, test:

```text
clean
status-color
agent-badge
compact
custom-card
```

Expected:
- `clean`: simple native text toast.
- `status-color`: native toast with status attribution/icon placement.
- `agent-badge`: native toast with circular badge placement.
- `compact`: one-line native toast.
- `custom-card`: native toast with generated hero image.

---

## Verification Summary

Run before marking complete:

```bash
cd windows-server
go test ./... -v
GOOS=windows GOARCH=amd64 go build -o agent-notify-server.exe
```

Then on Windows:

```powershell
.\agent-notify-server.exe
```

Open `http://localhost:17891/settings`, select `custom-card`, and send a test toast.

Success means: custom card image shows inside a native Windows toast, and all existing notification flow remains system-managed.
