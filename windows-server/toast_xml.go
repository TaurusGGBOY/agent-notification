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