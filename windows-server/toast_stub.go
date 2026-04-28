//go:build !windows

package main

import "log"

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