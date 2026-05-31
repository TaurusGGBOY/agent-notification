//go:build darwin

package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// ToastNotifier sends native macOS notifications via osascript.
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
	sound := "Glass"
	if event == "start" {
		sound = "Hero"
	}

	// osascript display notification: title = event label, subtitle = agent, body = message
	subtitle := strings.TrimSpace(agent)
	if subtitle == "" {
		subtitle = n.appName
	}

	script := fmt.Sprintf(
		`display notification %s with title %s subtitle %s sound name %q`,
		appleScriptQuote(message),
		appleScriptQuote(title),
		appleScriptQuote(subtitle),
		sound,
	)

	log.Printf("sending macOS notification: title=%q subtitle=%q", title, subtitle)
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("osascript failed: %v, output: %s", err, string(output))
		return fmt.Errorf("osascript: %w", err)
	}
	log.Printf("macOS notification sent successfully")
	return nil
}

// appleScriptQuote wraps a string in double quotes for AppleScript,
// escaping any embedded double quotes and backslashes.
func appleScriptQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
