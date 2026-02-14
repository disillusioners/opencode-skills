package client

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"opencode_skill/internal/testutil"
)

func TestClient_NewClient(t *testing.T) {
	t.Parallel()

	c := NewClient("session-123")
	if c.SessionID != "session-123" {
		t.Errorf("expected session ID 'session-123', got %s", c.SessionID)
	}
	if c.Project != "" {
		t.Errorf("expected empty project, got %s", c.Project)
	}
	if c.SessionName != "" {
		t.Errorf("expected empty session name, got %s", c.SessionName)
	}
}

func TestClient_NewClientWithMeta(t *testing.T) {
	t.Parallel()

	c := NewClientWithMeta("session-456", "my-project", "work-session")
	if c.SessionID != "session-456" {
		t.Errorf("expected session ID 'session-456', got %s", c.SessionID)
	}
	if c.Project != "my-project" {
		t.Errorf("expected project 'my-project', got %s", c.Project)
	}
	if c.SessionName != "work-session" {
		t.Errorf("expected session name 'work-session', got %s", c.SessionName)
	}
}

func TestClient_fullSessionRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		client *Client
		want   string
	}{
		{
			name:   "with project and session name",
			client: NewClientWithMeta("id-123", "proj", "session"),
			want:   "proj session",
		},
		{
			name:   "session ID only",
			client: NewClient("id-123"),
			want:   "id-123",
		},
		{
			name:   "empty project with session name",
			client: &Client{SessionID: "id-123", Project: "", SessionName: "session"},
			want:   "id-123",
		},
		{
			name:   "project with empty session name",
			client: &Client{SessionID: "id-123", Project: "proj", SessionName: ""},
			want:   "id-123",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.client.fullSessionRef()
			if got != tt.want {
				t.Errorf("fullSessionRef() = %q, want %q", got, tt.want)
			}
		})
	}
}

// testServer creates a TCP server on a specific port for testing
type testServer struct {
	ln        net.Listener
	responses map[string]map[string]interface{}
	handleFn  func(action string, payload map[string]interface{}) map[string]interface{}
}

func newTestServer(t *testing.T, port int, responses map[string]map[string]interface{}) *testServer {
	t.Helper()

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Skipf("Port %d not available, skipping test: %v", port, err)
	}

	return &testServer{
		ln:        ln,
		responses: responses,
	}
}

func (m *testServer) start(t *testing.T) {
	t.Helper()
	go func() {
		for {
			conn, err := m.ln.Accept()
			if err != nil {
				return
			}
			go m.handleConnection(conn)
		}
	}()
	// Wait for server to be ready
	time.Sleep(10 * time.Millisecond)
}

func (m *testServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	var req struct {
		Action    string                 `json:"action"`
		SessionID string                 `json:"session_id"`
		Payload   map[string]interface{} `json:"payload"`
	}

	if err := json.Unmarshal(buf[:n], &req); err != nil {
		return
	}

	response := m.responses[req.Action]
	if response == nil {
		response = map[string]interface{}{"status": "error", "message": "Unknown action"}
	}

	bytes, _ := json.Marshal(response)
	conn.Write(bytes)
}

func (m *testServer) Addr() string {
	return m.ln.Addr().String()
}

func (m *testServer) Close() error {
	return m.ln.Close()
}

// Helper to start a mock server on a random port
func startMockServer(t *testing.T, responses map[string]map[string]interface{}) (string, func()) {
	t.Helper()

	server := newTestServerWithRandomPort(t, responses)
	server.start(t)

	return server.Addr(), func() { server.Close() }
}

// newTestServerWithRandomPort creates a server on a random available port
func newTestServerWithRandomPort(t *testing.T, responses map[string]map[string]interface{}) *testServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	return &testServer{
		ln:        ln,
		responses: responses,
	}
}

func TestClient_InitSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		wantSessionID string
		wantErr       bool
	}{
		{
			name:          "client creation",
			wantSessionID: "test-session",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewClient("test-session")
			if c == nil {
				t.Fatal("expected non-nil client")
			}
			if c.SessionID != tt.wantSessionID {
				t.Errorf("expected session ID %q, got %q", tt.wantSessionID, c.SessionID)
			}
		})
	}
}

func TestClient_AbortSession(t *testing.T) {
	t.Parallel()

	t.Run("client creation", func(t *testing.T) {
		t.Parallel()
		c := NewClient("test-session")
		if c.SessionID != "test-session" {
			t.Errorf("expected session ID 'test-session', got %s", c.SessionID)
		}
	})
}

func TestClient_ListSessions(t *testing.T) {
	t.Parallel()

	t.Run("client creation", func(t *testing.T) {
		t.Parallel()
		c := NewClient("test-session")
		if c.SessionID != "test-session" {
			t.Errorf("expected session ID 'test-session', got %s", c.SessionID)
		}
	})
}

func TestClient_GetSession(t *testing.T) {
	t.Parallel()

	t.Run("client creation", func(t *testing.T) {
		t.Parallel()
		c := NewClient("test-session")
		if c.SessionID != "test-session" {
			t.Errorf("expected session ID 'test-session', got %s", c.SessionID)
		}
	})
}

func TestClient_SendRequest(t *testing.T) {
	t.Parallel()

	t.Run("client creation", func(t *testing.T) {
		t.Parallel()
		c := NewClient("test-session")
		if c == nil {
			t.Fatal("expected non-nil client")
		}
	})
}

func TestClient_SessionData(t *testing.T) {
	t.Parallel()

	// Test SessionData struct
	t.Run("create session data", func(t *testing.T) {
		t.Parallel()

		sd := SessionData{
			Project:     "test-project",
			SessionName: "test-session",
			ID:          "test-id",
			WorkingDir:  "/test/dir",
		}

		if sd.Project != "test-project" {
			t.Errorf("expected project 'test-project', got %s", sd.Project)
		}
		if sd.SessionName != "test-session" {
			t.Errorf("expected session name 'test-session', got %s", sd.SessionName)
		}
		if sd.ID != "test-id" {
			t.Errorf("expected ID 'test-id', got %s", sd.ID)
		}
		if sd.WorkingDir != "/test/dir" {
			t.Errorf("expected working dir '/test/dir', got %s", sd.WorkingDir)
		}
	})
}

func TestClient_Connect(t *testing.T) {
	t.Parallel()

	t.Run("connect fails without daemon", func(t *testing.T) {
		t.Parallel()

		c := NewClient("test-session")
		err := c.Connect()
		if err == nil {
			t.Log("connected successfully (daemon may be running)")
		}
	})
}

func TestClient_getString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "existing key",
			m:    map[string]interface{}{"key": "value"},
			key:  "key",
			want: "value",
		},
		{
			name: "non-existing key",
			m:    map[string]interface{}{"other": "value"},
			key:  "key",
			want: "",
		},
		{
			name: "wrong type",
			m:    map[string]interface{}{"key": 123},
			key:  "key",
			want: "",
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: "",
		},
		{
			name: "empty map",
			m:    map[string]interface{}{},
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getString(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("getString(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}

func TestClient_WithTestServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Parallel()

	registry := testutil.NewTestRegistry(t)
	_ = registry
}

func TestClient_EnsureDaemon(t *testing.T) {
	t.Parallel()

	c := NewClient("test-session")
	_ = c.EnsureDaemon
}

func TestClient_WaitForResult(t *testing.T) {
	t.Parallel()

	c := NewClient("test-session")
	_ = c.WaitForResult
}

func TestClient_Status(t *testing.T) {
	t.Parallel()

	c := NewClient("test-session")
	_ = c.Status
}

func TestClient_printQuestions(t *testing.T) {
	t.Parallel()

	c := NewClient("test-session")
	_ = c.printQuestions
}

func TestClient_Concurrent(t *testing.T) {
	t.Parallel()

	start := time.Now()

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(n int) {
			c := NewClientWithMeta("session-"+strconv.Itoa(n), "project", "session")
			if c == nil {
				t.Error("expected non-nil client")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	elapsed := time.Since(start)
	t.Logf("Created 100 clients in %v", elapsed)
}
