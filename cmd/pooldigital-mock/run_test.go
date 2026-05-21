package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// freePort returns an unused localhost TCP port for tests.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

func startServers(t *testing.T) (string, string, func()) {
	t.Helper()
	procPort := freePort(t)
	violetPort := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	var stdout bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, []string{
			"--host=127.0.0.1",
			fmt.Sprintf("--proconip-port=%d", procPort),
			fmt.Sprintf("--violet-port=%d", violetPort),
			"--no-color",
		}, &stdout, io.Discard)
	}()

	// Wait for both ports to accept connections
	procURL := fmt.Sprintf("http://127.0.0.1:%d", procPort)
	violetURL := fmt.Sprintf("http://127.0.0.1:%d", violetPort)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("servers did not start in time")
		}
		if portReady(procPort) && portReady(violetPort) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cleanup := func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("run returned error: %v", err)
			}
		case <-time.After(shutdownTimeout + time.Second):
			t.Error("run did not return after shutdown")
		}
	}
	return procURL, violetURL, cleanup
}

func portReady(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func TestRunStartsBothServers(t *testing.T) {
	procURL, violetURL, cleanup := startServers(t)
	defer cleanup()

	// proconip requires auth
	resp, err := http.Get(procURL + "/GetState.csv")
	if err != nil {
		t.Fatalf("proconip Get: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("proconip status = %d, want 401", resp.StatusCode)
	}

	// violet /getReadings is anonymous
	resp, err = http.Get(violetURL + "/getReadings?ALL")
	if err != nil {
		t.Fatalf("violet Get: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("violet status = %d, want 200", resp.StatusCode)
	}
}

func TestRunShutsDownOnContextCancel(t *testing.T) {
	procPort := freePort(t)
	violetPort := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- run(ctx, []string{
			"--host=127.0.0.1",
			fmt.Sprintf("--proconip-port=%d", procPort),
			fmt.Sprintf("--violet-port=%d", violetPort),
			"--no-color",
			"--quiet",
		}, io.Discard, io.Discard)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && (!portReady(procPort) || !portReady(violetPort)) {
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("run returned error: %v", err)
		}
	case <-time.After(shutdownTimeout + time.Second):
		t.Fatal("run did not exit after cancel")
	}
}

func TestRunVersionFlag(t *testing.T) {
	var out bytes.Buffer
	err := run(context.Background(), []string{"--version"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "pooldigital-mock") {
		t.Errorf("version output missing program name: %q", out.String())
	}
}

func TestRunHelpFlag(t *testing.T) {
	err := run(context.Background(), []string{"--help"}, io.Discard, io.Discard)
	if err != nil {
		t.Errorf("--help should not error: %v", err)
	}
}

func TestRunUnknownFlag(t *testing.T) {
	err := run(context.Background(), []string{"--definitely-not-a-flag"}, io.Discard, io.Discard)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestRunBannerEmitted(t *testing.T) {
	procPort := freePort(t)
	violetPort := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	var stdout bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, []string{
			"--host=127.0.0.1",
			fmt.Sprintf("--proconip-port=%d", procPort),
			fmt.Sprintf("--violet-port=%d", violetPort),
			"--no-color",
			"--quiet",
		}, &stdout, io.Discard)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && (!portReady(procPort) || !portReady(violetPort)) {
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	out := stdout.String()
	for _, want := range []string{"pooldigital-mock", "proconip", "violet"} {
		if !strings.Contains(out, want) {
			t.Errorf("banner missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestEnvOrDefaultFallbacks(t *testing.T) {
	t.Setenv("X_TEST_ENV", "hello")
	if got := envOrDefault("X_TEST_ENV", "world"); got != "hello" {
		t.Errorf("env hit: got %q want hello", got)
	}
	if got := envOrDefault("X_NOT_SET_DEFINITELY", "world"); got != "world" {
		t.Errorf("env miss: got %q want world", got)
	}
}

func TestEnvIntOrDefaultParsing(t *testing.T) {
	t.Setenv("X_TEST_INT", "42")
	if got := envIntOrDefault("X_TEST_INT", 7); got != 42 {
		t.Errorf("parse: got %d want 42", got)
	}
	t.Setenv("X_TEST_INT", "not-a-number")
	if got := envIntOrDefault("X_TEST_INT", 7); got != 7 {
		t.Errorf("bad parse should fall back: got %d", got)
	}
	if got := envIntOrDefault("X_DEFINITELY_UNSET", 9); got != 9 {
		t.Errorf("unset: got %d want 9", got)
	}
}

func TestDisplayURLLocalhostSubst(t *testing.T) {
	cases := []struct {
		host string
		port int
		want string
	}{
		{"0.0.0.0", 8080, "http://localhost:8080"},
		{"::", 8180, "http://localhost:8180"},
		{"", 80, "http://localhost:80"},
		{"127.0.0.1", 9999, "http://127.0.0.1:9999"},
	}
	for _, c := range cases {
		if got := displayURL(c.host, c.port); got != c.want {
			t.Errorf("displayURL(%q,%d) = %q, want %q", c.host, c.port, got, c.want)
		}
	}
}
