//go:build windows

package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var dnsSDCommand = exec.Command

func StartMDNSAdvertisement(ctx context.Context, port uint16) error {
	host := localHostname()
	// Use hyphens instead of spaces to avoid PowerShell argument parsing issues.
	instance := "Agent-Notify-" + host

	args := []string{
		"-R",
		instance,
		mdnsServiceType,
		"local.",
		strconv.Itoa(int(port)),
	}
	args = append(args, discoveryTXT()...)

	cmd := dnsSDCommand("dns-sd", args...)
	cmd.Stdout = logWriter{}
	cmd.Stderr = logWriter{}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("dns-sd stdin pipe failed: %w", err)
	}

	log.Printf("starting dns-sd command: dns-sd %q", args)
	startedAt := time.Now()
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("dns-sd start failed: %w", err)
	}
	log.Printf("dns-sd started with pid %d", cmd.Process.Pid)

	done := make(chan error, 1)

	// Wait for the process to exit (either killed or unexpected failure).
	go func() {
		err := cmd.Wait()
		done <- err

		elapsed := time.Since(startedAt).Round(time.Millisecond)
		if ctx.Err() != nil {
			log.Printf("dns-sd stopped after %s: %v", elapsed, err)
			return
		}
		if err != nil {
			log.Printf("dns-sd exited unexpectedly after %s: %v", elapsed, err)
			return
		}
		log.Printf("dns-sd exited unexpectedly after %s without error", elapsed)
	}()

	go func() {
		<-ctx.Done()
		_ = stdin.Close()

		select {
		case <-done:
			return
		case <-time.After(2 * time.Second):
			log.Printf("dns-sd did not stop after stdin close; killing pid %d", cmd.Process.Pid)
			_ = cmd.Process.Kill()
		}
	}()

	log.Printf("mDNS advertising %s as %q on port %d (via dns-sd)", mdnsServiceType, instance, port)
	return nil
}

// logWriter is an io.Writer that logs each line to the standard logger.
type logWriter struct{}

func (logWriter) Write(p []byte) (n int, err error) {
	log.Printf("[dns-sd] %s", strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
