//go:build windows
// +build windows

package main

import (
	"encoding/base64"
	"log"
	"os/exec"
	"syscall"
)

type ToastNotifier struct {
	appName string
}

var (
	toastLogoPathFunc   = toastAppLogoPath
	renderToastLogoFn   = renderToastAppLogo
	sendToastPowerShell = sendToastViaPowerShell
)

const createNoWindow = 0x08000000

func NewToastNotifier(appName string) *ToastNotifier {
	return &ToastNotifier{appName: appName}
}

func (n *ToastNotifier) Notify(title, message string) error {
	return n.NotifyWithStyle("clean", "stop", title, message, "")
}

func (n *ToastNotifier) NotifyWithStyle(style, event, title, message, agent string) error {
	logoPath := ""
	path, err := toastLogoPathFunc()
	if err != nil {
		log.Printf("toast logo path failed: %v", err)
	} else if err := renderToastLogoFn(path); err != nil {
		log.Printf("render toast logo failed: %v", err)
	} else {
		logoPath = path
	}
	xml := formatToastXML(style, event, title, message, agent, "", logoPath)
	log.Printf("sending toast via PowerShell, XML length: %d", len(xml))
	if err := sendToastPowerShell(n.appName, xml); err != nil {
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
	cmd := newHiddenPowerShellCommand(script)
	return cmd.Run()
}

func newHiddenPowerShellCommand(script string) *exec.Cmd {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return cmd
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
