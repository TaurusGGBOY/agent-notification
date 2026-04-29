//go:build windows
// +build windows

package main

import (
	"bytes"
	"os/exec"
	"strings"
)

// ToastNotifier sends Windows toast notifications using PowerShell + WinRT API
type ToastNotifier struct {
	appName string
}

// Valid styles
const (
	StyleClean       = "clean"
	StyleStatusColor = "status-color"
	StyleAgentBadge  = "agent-badge"
	StyleCompact     = "compact"
)

func NewToastNotifier(appName string) *ToastNotifier {
	return &ToastNotifier{appName: appName}
}

// Notify sends a basic notification (legacy method)
func (n *ToastNotifier) Notify(title, message string) error {
	return n.NotifyWithStyle(StyleClean, "start", title, message, n.appName)
}

// NotifyWithStyle sends a notification with custom style
func (n *ToastNotifier) NotifyWithStyle(style, event, title, message, agent string) error {
	xml := formatToastXML(style, event, title, agent, message)
	return sendToastNotification(n.appName, xml)
}

// formatToastXML generates ToastGeneric XML based on style
func formatToastXML(style, event, title, agent, extra string) string {
	switch style {
	case StyleStatusColor:
		return buildStatusColorXML(event, title, agent)
	case StyleAgentBadge:
		return buildAgentBadgeXML(event, title, agent)
	case StyleCompact:
		return buildCompactXML(event, title, agent)
	case StyleClean:
	default:
		return buildCleanXML(title, agent, extra)
	}
	return buildCleanXML(title, agent, extra)
}

// buildCleanXML - plain text, no icon
func buildCleanXML(title, agent, extra string) string {
	return buildBaseToastXML(title, "", agent, "", "hide")
}

// buildStatusColorXML - icon + attribution ("Started"/"Stopped")
func buildStatusColorXML(event, title, agent string) string {
	iconPath := getStatusIconPath(event)
	attribution := getStatusAttribution(event)
	return buildBaseToastXML(title, iconPath, agent, attribution, "hide")
}

// buildAgentBadgeXML - circular avatar + agent initial
func buildAgentBadgeXML(event, title, agent string) string {
	iconPath := getAgentBadgePath(agent)
	initial := getAgentInitial(agent)
	attribution := initial
	return buildBaseToastXML(title, iconPath, agent, attribution, "hide")
}

// buildCompactXML - single line, minimal
func buildCompactXML(event, title, agent string) string {
	return buildBaseToastXML(title, "", agent, "", "hide")
}

// buildBaseToastXML creates the shared XML structure
func buildBaseToastXML(title, iconPath, agent, attribution, iconPlacement string) string {
	var buf strings.Builder
	buf.WriteString(`<toast><visual><binding template="ToastGeneric">`)
	buf.WriteString(`<text>` + escapeXML(title) + `</text>`)

	if iconPath != "" {
		buf.WriteString(`<image placement="appLogoOverride" src="` + escapeXML(iconPath) + `"/>`)
	}

	if attribution != "" {
		buf.WriteString(`<text placement="attribution">` + escapeXML(attribution) + `</text>`)
	}

	buf.WriteString(`</binding></visual></toast>`)
	return buf.String()
}

// sendToastNotification sends toast via PowerShell + WinRT
func sendToastNotification(appID, xml string) error {
	script := sendToastViaShell(appID, xml)
	return runPowerShell(script)
}

// sendToastViaShell generates PowerShell script for WinRT toast
func sendToastViaShell(appID, xml string) string {
	// Escape XML for embedding in PowerShell string
	escapedXML := strings.ReplaceAll(xml, "`", "``")
	escapedXML = strings.ReplaceAll(escapedXML, `"`, "`" + `"`)
	escapedXML = strings.ReplaceAll(escapedXML, "$", "`$")

	return `
Add-Type -AssemblyName System.Runtime.WindowsRuntime
Add-Type -AssemblyName Windows.UI.Notifications

$as = [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]
$manager = $as::CreateToastNotifier("` + appID + `")

$xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastGeneric)
$xml.LoadXml(` + "`" + escapedXML + "`" + `)

$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
$manager.Show($toast)
`
}

// runPowerShell executes PowerShell script
func runPowerShell(script string) error {
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-Command", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return err
		}
		return err
	}
	return nil
}

// getStatusIconPath returns icon path for start/stop events
func getStatusIconPath(event string) string {
	// Use placeholder icons - in production these would be actual file paths
	if event == "start" {
		return "ms-appx:///assets/start.png"
	}
	return "ms-appx:///assets/stop.png"
}

// getStatusAttribution returns attribution text for event
func getStatusAttribution(event string) string {
	if event == "start" {
		return "Started"
	}
	return "Stopped"
}

// getAgentBadgePath returns badge icon path for agent
func getAgentBadgePath(agent string) string {
	// Agent badge icons would be in app assets
	return "ms-appx:///assets/agent-" + strings.ToLower(agent) + ".png"
}

// getAgentInitial returns the first character uppercase
func getAgentInitial(agent string) string {
	if agent == "" {
		return "?"
	}
	return strings.ToUpper(string(agent[0]))
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}