//go:build windows

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStartMDNSAdvertisementKeepsDNSSDStdinOpen(t *testing.T) {
	if os.Getenv("GO_WANT_DNSSD_HELPER_PROCESS") == "1" {
		runDNSSDHelperProcess()
		return
	}

	tempDir := t.TempDir()
	startedPath := filepath.Join(tempDir, "started")
	exitedPath := filepath.Join(tempDir, "exited")

	originalCommand := dnsSDCommand
	t.Cleanup(func() { dnsSDCommand = originalCommand })
	dnsSDCommand = func(name string, args ...string) *exec.Cmd {
		if name != "dns-sd" {
			t.Fatalf("command name = %q, want dns-sd", name)
		}
		cmd := exec.Command(os.Args[0], "-test.run=TestStartMDNSAdvertisementKeepsDNSSDStdinOpen")
		cmd.Env = append(os.Environ(),
			"GO_WANT_DNSSD_HELPER_PROCESS=1",
			"DNSSD_HELPER_STARTED="+startedPath,
			"DNSSD_HELPER_EXITED="+exitedPath,
		)
		return cmd
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if err := StartMDNSAdvertisement(ctx, 17891); err != nil {
		t.Fatalf("StartMDNSAdvertisement failed: %v", err)
	}
	waitForFile(t, startedPath)

	time.Sleep(250 * time.Millisecond)
	if _, err := os.Stat(exitedPath); err == nil {
		t.Fatal("dns-sd helper exited before advertisement context was cancelled")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat helper exit marker failed: %v", err)
	}

	cancel()
}

func runDNSSDHelperProcess() {
	startedPath := os.Getenv("DNSSD_HELPER_STARTED")
	exitedPath := os.Getenv("DNSSD_HELPER_EXITED")
	if startedPath != "" {
		_ = os.WriteFile(startedPath, []byte("started"), 0o600)
	}

	_, _ = os.Stdin.Read(make([]byte, 1))

	if exitedPath != "" {
		_ = os.WriteFile(exitedPath, []byte("exited"), 0o600)
	}
	os.Exit(0)
}

func waitForFile(t *testing.T, path string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s failed: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}
