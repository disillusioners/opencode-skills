package testutil

import (
	"net"
	"path/filepath"
	"testing"

	"opencode_wrapper/internal/daemon"
)

func FreePort() (int, error) {
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

func NewTestRegistry(t *testing.T) *daemon.Registry {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	r, err := daemon.NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func NewTestServer(t *testing.T, registry *daemon.Registry) *daemon.Server {
	t.Helper()
	s := daemon.NewServer(registry)
	return s
}
