//go:build windows
// +build windows

package main

import (
	"log"

	"github.com/go-toast/toast"
)

// ToastNotifier sends Windows toast notifications using go-toast library
type ToastNotifier struct {
	appName string
}

func NewToastNotifier(appName string) *ToastNotifier {
	return &ToastNotifier{appName: appName}
}

// Notify sends a basic notification (legacy method)
func (n *ToastNotifier) Notify(title, message string) error {
	notification := toast.Notification{
		AppID:   n.appName,
		Title:   title,
		Message: message,
	}
	return notification.Push()
}

// NotifyWithStyle sends a notification with style-based customization
// Note: go-toast has limited customization. Styles are approximated via title/message formatting.
func (n *ToastNotifier) NotifyWithStyle(style, event, title, message, agent string) error {
	notification := toast.Notification{
		AppID:   n.appName,
		Title:   title,
		Message: message,
	}

	// Style-specific adjustments via go-toast library capabilities
	switch style {
	case "status-color":
		// Status color uses attribution text feature if available
		notification.Audio = toast.Default
	case "agent-badge":
		// Agent badge uses custom icon if available
		notification.Audio = toast.Default
	case "compact":
		// Compact uses no sound
		notification.Audio = toast.Silent
	default:
		notification.Audio = toast.Default
	}

	if err := notification.Push(); err != nil {
		log.Printf("Toast notification failed: %v", err)
		return err
	}
	return nil
}
