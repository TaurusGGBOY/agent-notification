//go:build windows
// +build windows

package main

import (
	"encoding/base64"
	"log"
	"os/exec"
)

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
	project := ""
	cardPath := ""
	if style == "custom-card" {
		log.Printf("custom-card style detected, generating toast card")
		path, err := toastCardPath()
		if err != nil {
			log.Printf("toastCardPath failed: %v", err)
			return err
		}
		log.Printf("card path: %s", path)
		card := ToastCard{
			Event:   event,
			Title:   title,
			Agent:   agent,
			Project: project,
			Message: message,
		}
		if err := renderToastCard(path, card); err != nil {
			log.Printf("renderToastCard failed: %v", err)
			return err
		}
		log.Printf("card rendered successfully")
		cardPath = path
	}
	xml := formatToastXML(style, event, title, agent, project, cardPath)
	log.Printf("sending toast via PowerShell, XML length: %d", len(xml))
	if err := sendToastViaPowerShell(n.appName, xml); err != nil {
		log.Printf("Toast notification failed: %v", err)
		return err
	}
	log.Printf("toast sent successfully")
	return nil
}

func sendToastViaPowerShell(appID, xml string) error {
	encodedXML := base64.StdEncoding.EncodeToString([]byte(xml))
	script := `
$xml = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('` + encodedXML + `'))
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$doc = [Windows.Data.Xml.Dom.XmlDocument]::new()
$doc.LoadXml($xml)
$toast = [Windows.UI.Notifications.ToastNotification]::new($doc)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('` + escapePowerShellSingleQuoted(appID) + `').Show($toast)
`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-Command", script)
	return cmd.Run()
}

func escapePowerShellSingleQuoted(s string) string {
	out := ""
	for _, r := range s {
		if r == '\'' {
			out += "''"
		} else {
			out += string(r)
		}
	}
	return out
}