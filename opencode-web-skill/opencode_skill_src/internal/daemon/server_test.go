package daemon

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opencode_skill/internal/config"
)

func freePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func cleanupPID() {
	os.Remove(config.PidFile)
}

func TestServer_SingletonEnforcement(t *testing.T) {
	defer cleanupPID()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	s1 := NewServer(registry)

	errChan := make(chan error, 1)
	go func() {
		errChan <- s1.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("First server failed to start: %v", err)
		}
	default:
	}

	s2 := NewServer(registry)
	err = s2.Start()

	if err == nil {
		t.Fatal("Second server should have failed to start, but succeeded")
	}

	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("Expected 'already running' error, got: %v", err)
	}
}
