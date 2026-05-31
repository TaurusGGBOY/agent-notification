//go:build darwin && !cgo

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func deliverDarwinNotification(req darwinNotificationRequest) error {
	script := fmt.Sprintf(
		`display notification %s with title %s subtitle %s sound name %q`,
		appleScriptQuote(req.Body),
		appleScriptQuote(req.Title),
		appleScriptQuote(req.Subtitle),
		req.Sound,
	)

	output, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func appleScriptQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
