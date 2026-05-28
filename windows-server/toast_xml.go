package main

import (
	"path/filepath"
	"strings"
)

func formatToastXML(style, event, title, message, agent, project, logoImagePath string) string {
	_ = style
	_ = event
	_ = agent
	return buildCleanXML(title, message, project, logoImagePath)
}

func buildCleanXML(title, message, project, logoImagePath string) string {
	var sb strings.Builder
	startToastBinding(&sb)
	appendToastImage(&sb, logoImagePath, "appLogoOverride", "circle")
	appendToastText(&sb, title, ` hint-style="base" hint-wrap="false"`)
	appendToastText(&sb, message, ` hint-style="captionSubtle" hint-wrap="true"`)
	appendToastAttribution(&sb, project)
	endToastBinding(&sb)
	return sb.String()
}

func buildToastXML(title, project, imagePath, placement, crop, attribution string) string {
	var sb strings.Builder
	startToastBinding(&sb)
	appendToastText(&sb, title, "")
	if project != "" {
		appendToastText(&sb, project, "")
	}
	if attribution != "" {
		appendToastAttribution(&sb, attribution)
	}
	if imagePath != "" {
		appendToastImage(&sb, imagePath, placement, crop)
	}
	endToastBinding(&sb)
	return sb.String()
}

func startToastBinding(sb *strings.Builder) {
	sb.WriteString(`<toast><visual><binding template="ToastGeneric">`)
}

func endToastBinding(sb *strings.Builder) {
	sb.WriteString(`</binding></visual></toast>`)
}

func appendToastText(sb *strings.Builder, text, attrs string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	sb.WriteString(`<text`)
	sb.WriteString(attrs)
	sb.WriteString(`>`)
	sb.WriteString(escapeXML(text))
	sb.WriteString(`</text>`)
}

func appendToastAttribution(sb *strings.Builder, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	appendToastText(sb, text, ` placement="attribution"`)
}

func appendToastImage(sb *strings.Builder, imagePath, placement, crop string) {
	if strings.TrimSpace(imagePath) == "" {
		return
	}
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
