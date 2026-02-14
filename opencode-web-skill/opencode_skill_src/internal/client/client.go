package client

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"opencode_skill/internal/config"
	"opencode_skill/internal/manager"
)

type Client struct {
	SessionID   string
	Project     string
	SessionName string
	conn        net.Conn
}

// SessionData represents session information from daemon
type SessionData struct {
	Project     string
	SessionName string
	ID          string
	WorkingDir  string
}

func NewClient(sessionID string) *Client {
	return &Client{
		SessionID: sessionID,
	}
}

func NewClientWithMeta(sessionID, project, sessionName string) *Client {
	return &Client{
		SessionID:   sessionID,
		Project:     project,
		SessionName: sessionName,
	}
}

func (c *Client) fullSessionRef() string {
	if c.Project != "" && c.SessionName != "" {
		return c.Project + " " + c.SessionName
	}
	return c.SessionID
}

func (c *Client) Connect() error {
	addr := net.JoinHostPort(config.DaemonHost, strconv.Itoa(config.DaemonPort))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) EnsureDaemon() error {
	if c.Connect() == nil {
		c.conn.Close()
		return nil
	}

	fmt.Println("Starting daemon...")

	// Spawn daemon
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(executable, "--daemon")
	cmd.Dir = config.ProjectRoot
	cmd.Stdout = nil // or redirect to log
	cmd.Stderr = nil

	// Detach process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %v", err)
	}

	// Wait for daemon to become ready
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if c.Connect() == nil {
			c.conn.Close()
			return nil
		}
	}
	return fmt.Errorf("daemon failed to start")
}

func (c *Client) SendRequest(action string, payload interface{}) (map[string]interface{}, error) {
	if err := c.Connect(); err != nil {
		// Try spawning once
		if err := c.EnsureDaemon(); err != nil {
			return nil, err
		}
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}
	defer c.conn.Close()

	req := map[string]interface{}{
		"action":     action,
		"session_id": c.SessionID,
		"payload":    payload,
	}

	if err := json.NewEncoder(c.conn).Encode(req); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(c.conn).Decode(&resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) WaitForResult() {
	start := time.Now()
	fmt.Printf("Waiting for result (Timeout: %v)...\n", config.ClientTimeout)

	for time.Since(start) < config.ClientTimeout {
		resp, err := c.SendRequest("GET_STATUS", nil)
		if err != nil {
			fmt.Printf("Error checking status: %v\n", err)
			time.Sleep(3 * time.Second)
			continue
		}

		if status, ok := resp["status"].(string); !ok || status != "ok" {
			fmt.Printf("Daemon error: %v\n", resp["message"])
			time.Sleep(3 * time.Second)
			continue
		}

		data, _ := resp["data"].(map[string]interface{})
		state, _ := data["state"].(string)

		// Check questions
		if questionsRaw, ok := data["questions"].([]interface{}); ok && len(questionsRaw) > 0 {
			c.printQuestions(questionsRaw)
			return
		}

		// Check result
		latestResp, _ := data["latest_response"].(map[string]interface{})

		if state == string(manager.StateIdle) && latestResp != nil {
			if errStr, ok := latestResp["error"].(string); ok && errStr != "" {
				fmt.Printf("Error: %s\n", errStr)
			} else if res, ok := latestResp["result"]; ok {
				formatted, _ := json.MarshalIndent(res, "", "  ")
				fmt.Println("Response received:")
				fmt.Println(string(formatted))
			}
			return
		}

		time.Sleep(3 * time.Second)
	}

	fmt.Println("\n[TIMEOUT] Message is taking longer than 10 minutes.")
	fmt.Println("Daemon is still running in background.")
	fmt.Printf("Run: `opencode_skill %s /wait` to check again.\n", c.fullSessionRef())
}

func (c *Client) printQuestions(questions []interface{}) {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("  ACTION REQUIRED")
	fmt.Println(strings.Repeat("=", 40))

	// We need to decode map[string]interface{} to api.Question manually or just traverse
	for _, qRaw := range questions {
		q, _ := qRaw.(map[string]interface{})
		fmt.Printf("[?] Request ID: %v\n", q["id"])

		if subQs, ok := q["questions"].([]interface{}); ok {
			for _, subQRaw := range subQs {
				subQ, _ := subQRaw.(map[string]interface{})
				fmt.Printf("    %v\n", subQ["question"])

				if opts, ok := subQ["options"].([]interface{}); ok {
					fmt.Println("    Options:")
					for _, optRaw := range opts {
						opt, _ := optRaw.(map[string]interface{})
						label := opt["label"]
						desc := opt["description"]
						if desc != nil && desc != "" {
							fmt.Printf("      - %v: %v\n", label, desc)
						} else {
							fmt.Printf("      - %v\n", label)
						}
					}
				}
			}
		}
	}
	fmt.Printf("\nRun: `opencode_skill %s /answer ...`\n", c.fullSessionRef())
}

func (c *Client) Status() {
	resp, err := c.SendRequest("GET_STATUS", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if status, ok := resp["status"].(string); !ok || status != "ok" {
		fmt.Printf("Error: %v\n", resp["message"])
		return
	}

	data, _ := resp["data"].(map[string]interface{})
	state, _ := data["state"].(string)

	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Printf("  SESSION STATUS: %s\n", state)
	fmt.Println(strings.Repeat("=", 40))

	// Safely get questions
	var qs []interface{}
	if qSlice, ok := data["questions"].([]interface{}); ok {
		qs = qSlice
	}

	if len(qs) > 0 {
		fmt.Println("\n[QUESTIONS PENDING]")
		c.printQuestions(qs)
	}

	latestResp, _ := data["latest_response"].(map[string]interface{})
	if latestResp != nil {
		fmt.Println("\n[LATEST RESPONSE]")
		formatted, _ := json.MarshalIndent(latestResp, "", "  ")
		fmt.Println(string(formatted))
	}

	if state == "IDLE" && len(qs) == 0 && latestResp == nil {
		fmt.Println("\nSession is idle with no pending work.")
	} else if state == "BUSY" {
		fmt.Println("\nSession is currently processing...")
		fmt.Println("Run `/wait` to monitor for completion.")
	}
}

func (c *Client) InitSession(project, sessionName, workingDir string) (*SessionData, error) {
	resp, err := c.SendRequest("INIT_SESSION", map[string]string{
		"project":      project,
		"session_name": sessionName,
		"working_dir":  workingDir,
	})
	if err != nil {
		return nil, err
	}

	if status, _ := resp["status"].(string); status != "ok" {
		return nil, fmt.Errorf("%v", resp["message"])
	}

	sessionID, _ := resp["session_id"].(string)
	return &SessionData{
		Project:     project,
		SessionName: sessionName,
		ID:          sessionID,
		WorkingDir:  workingDir,
	}, nil
}

func (c *Client) AbortSession(project, sessionName string) error {
	resp, err := c.SendRequest("ABORT_SESSION", map[string]string{
		"project":      project,
		"session_name": sessionName,
	})
	if err != nil {
		return err
	}

	if status, _ := resp["status"].(string); status != "ok" {
		return fmt.Errorf("%v", resp["message"])
	}

	return nil
}

func (c *Client) ListSessions() ([]SessionData, error) {
	resp, err := c.SendRequest("LIST_SESSIONS", nil)
	if err != nil {
		return nil, err
	}

	if status, _ := resp["status"].(string); status != "ok" {
		return nil, fmt.Errorf("%v", resp["message"])
	}

	sessionsRaw, _ := resp["sessions"].([]interface{})
	sessions := make([]SessionData, 0, len(sessionsRaw))

	for _, sRaw := range sessionsRaw {
		s, _ := sRaw.(map[string]interface{})
		sessions = append(sessions, SessionData{
			Project:     getString(s, "project"),
			SessionName: getString(s, "session_name"),
			ID:          getString(s, "session_id"),
			WorkingDir:  getString(s, "working_dir"),
		})
	}

	return sessions, nil
}

func (c *Client) GetSession(project, sessionName string) (*SessionData, error) {
	resp, err := c.SendRequest("GET_SESSION", map[string]string{
		"project":      project,
		"session_name": sessionName,
	})
	if err != nil {
		return nil, err
	}

	if status, _ := resp["status"].(string); status != "ok" {
		return nil, fmt.Errorf("%v", resp["message"])
	}

	sessionRaw, _ := resp["session"].(map[string]interface{})
	return &SessionData{
		Project:     getString(sessionRaw, "project"),
		SessionName: getString(sessionRaw, "session_name"),
		ID:          getString(sessionRaw, "session_id"),
		WorkingDir:  getString(sessionRaw, "working_dir"),
	}, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
