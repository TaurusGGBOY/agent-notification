package main

// Notifier defines the interface for sending notifications
type Notifier interface {
	Notify(title, message string) error
	NotifyWithStyle(style, event, title, message, agent string) error
}