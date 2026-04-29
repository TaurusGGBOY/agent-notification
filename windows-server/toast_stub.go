//go:build !windows

package main

import (
	"log"
	"strings"
)

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

func (n *ToastNotifier) NotifyWithStyle(style, event, title, message, agent string) error {
	log.Printf("[toast stub:%s] style=%s event=%s %s - %s", n.appName, style, event, title, message)
	return nil
}

// formatToastXML generates ToastGeneric XML based on style (stub for non-Windows)
func formatToastXML(style, event, title, agent, extra string) string {
	switch style {
	case "status-color":
		return buildStatusColorXML(event, title, agent)
	case "agent-badge":
		return buildAgentBadgeXML(event, title, agent)
	case "compact":
		return buildCompactXML(event, title, agent)
	case "clean":
		return buildCleanXML(title, agent, extra)
	default:
		return buildCleanXML(title, agent, extra)
	}
}

func buildCleanXML(title, agent, extra string) string {
	return `<toast><visual><binding template="ToastGeneric"><text>` + escapeXML(title) + `</text></binding></visual></toast>`
}

func buildStatusColorXML(event, title, agent string) string {
	attribution := "Started"
	if event == "stop" {
		attribution = "Stopped"
	}
	iconPath := "ms-appx:///assets/status.png"
	return `<toast><visual><binding template="ToastGeneric"><text>` + escapeXML(title) + `</text><image placement="appLogoOverride" src="` + iconPath + `"/><text placement="attribution">` + escapeXML(attribution) + `</text></binding></visual></toast>`
}

func buildAgentBadgeXML(event, title, agent string) string {
	initial := "?"
	if agent != "" {
		initial = strings.ToUpper(string(agent[0]))
	}
	iconPath := "ms-appx:///assets/agent.png"
	return `<toast><visual><binding template="ToastGeneric"><text>` + escapeXML(title) + `</text><image placement="appLogoOverride" src="` + iconPath + `"/><text placement="attribution">` + escapeXML(initial) + `</text></binding></visual></toast>`
}

func buildCompactXML(event, title, agent string) string {
	return `<toast><visual><binding template="ToastGeneric"><text>` + escapeXML(title) + `</text></binding></visual></toast>`
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}