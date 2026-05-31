//go:build !windows && !darwin

package main

import (
	"log"
)

// ToastNotifier stub for non-Windows platforms
type ToastNotifier struct {
	appName string
}

func NewToastNotifier(appName string) *ToastNotifier {
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
