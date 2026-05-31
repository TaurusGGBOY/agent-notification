//go:build darwin

package main

import (
	"log"
	"strings"
)

// ToastNotifier sends native macOS notifications.
type ToastNotifier struct {
	appName string
}

type darwinNotificationRequest struct {
	Title    string
	Subtitle string
	Body     string
	Sound    string
}

func NewToastNotifier(appName string) *ToastNotifier {
	return &ToastNotifier{appName: appName}
}

func (n *ToastNotifier) Notify(title, message string) error {
	return n.NotifyWithStyle("clean", "stop", title, message, "")
}

func (n *ToastNotifier) NotifyWithStyle(style, event, title, message, agent string) error {
	req := newDarwinNotificationRequest(event, title, message, agent, n.appName)
	log.Printf("sending native macOS notification: title=%q subtitle=%q", req.Title, req.Subtitle)
	if err := deliverDarwinNotification(req); err != nil {
		log.Printf("native macOS notification failed: %v", err)
		return err
	}
	log.Printf("native macOS notification sent successfully")
	return nil
}

func newDarwinNotificationRequest(event, title, body, agent, appName string) darwinNotificationRequest {
	subtitle := strings.TrimSpace(agent)
	if subtitle == "" {
		subtitle = strings.TrimSpace(appName)
	}
	if subtitle == "" {
		subtitle = "AgentNotify"
	}

	sound := "Glass"
	if strings.EqualFold(strings.TrimSpace(event), "start") {
		sound = "Hero"
	}

	return darwinNotificationRequest{
		Title:    strings.TrimSpace(title),
		Subtitle: subtitle,
		Body:     strings.TrimSpace(body),
		Sound:    sound,
	}
}
