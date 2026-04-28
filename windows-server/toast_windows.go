//go:build windows
// +build windows

package main

import (
	"github.com/go-toast/toast"
)

// ToastNotifier sends Windows toast notifications
type ToastNotifier struct {
	appName string
}

func NewToastNotifier(appName string) *ToastNotifier {
	return &ToastNotifier{appName: appName}
}

func (n *ToastNotifier) Notify(title, message string) error {
	notification := toast.Notification{
		AppID:   n.appName,
		Title:   title,
		Message: message,
	}
	return notification.Push()
}
