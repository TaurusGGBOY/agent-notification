//go:build !windows
// +build !windows

package main

// ToastNotifier stub for non-Windows platforms
type ToastNotifier struct {
	appName string
}

func NewToastNotifier(appName string) *ToastNotifier {
	return &ToastNotifier{appName: appName}
}

func (n *ToastNotifier) Notify(title, message string) error {
	return nil
}
