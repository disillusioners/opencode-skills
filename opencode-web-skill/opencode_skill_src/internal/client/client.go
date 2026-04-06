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
	Quiet       bool
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

func (c *Client) SetQuiet(quiet bool) {
	c.Quiet = quiet
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

func (c *Client) getDaemonInfo() (startTime string, err error) {
	resp, err := c.SendRequest("PING", nil)
	if err != nil {
		return "", err
	}
	if status, ok := resp["status"].(string); !ok || status != "ok" {
		return "", fmt.Errorf("%v", resp["message"])
	}
	if t, ok := resp["start_time"].(string); ok {
		return t, nil
	}
	return "", nil
}

func (c *Client) WaitForResult() {
	start := time.Now()

	// Get daemon start time first
	daemonStartTime, _ := c.getDaemonInfo()

	if !c.Quiet {
		fmt.Printf("[TOOL_INFO] daemon last start time: %s\n", daemonStartTime)
		fmt.Printf("Waiting for result (Timeout: %v)...\n", config.ClientTimeout)
	}

	for time.Since(start) < config.ClientTimeout {
		resp, err := c.SendRequest("GET_STATUS", nil)
		if err != nil {
			fmt.Printf("Error checking status: %v\n", err)
			time.Sleep(config.PollInterval)
			continue
		}

		if status, ok := resp["status"].(string); !ok || status != "ok" {
			fmt.Printf("Daemon error: %v\n", resp["message"])
			time.Sleep(config.PollInterval)
			continue
		}

		data, _ := resp["data"].(map[string]interface{})
		state, _ := data["state"].(string)

		// Check questions
		if questionsRaw, ok := data["questions"].([]interface{}); ok && len(questionsRaw) > 0 {
			c.printQuestions(questionsRaw)
			return
		}

		// Check result - return immediately if session is IDLE
		latestResp, _ := data["latest_response"].(map[string]interface{})

		if state == string(manager.StateIdle) {
			if latestResp != nil {
				if errStr, ok := latestResp["error"].(string); ok && errStr != "" {
					fmt.Printf("Error: %s\n", errStr)
				} else if res, ok := latestResp["result"]; ok {
					formatted, _ := json.MarshalIndent(res, "", "  ")
					if c.Quiet {
						fmt.Println(string(formatted))
					} else {
						fmt.Println("Response received:")
						fmt.Println(string(formatted))
					}
				}
			} else {
				if c.Quiet {
					fmt.Println("Session is idle with no result available")
				} else {
					fmt.Println("Session completed but no result was captured.")
					fmt.Println("Run `/status` for more details.")
				}
			}
			return
		}

		time.Sleep(config.PollInterval)
	}

	if c.Quiet {
		fmt.Println("Error: Timeout waiting for result")
	} else {
		fmt.Printf("\n[TIMEOUT] Message is taking longer than %v.\n", config.ClientTimeout)
		fmt.Println("Daemon is still running in background.")
		fmt.Printf("Run: `opencode_skill %s /wait` to check again.\n", c.fullSessionRef())
	}
}

func (c *Client) printQuestions(questions []interface{}) {
	if !c.Quiet {
		fmt.Println("\n" + strings.Repeat("=", 40))
		fmt.Println("  ACTION REQUIRED")
		fmt.Println(strings.Repeat("=", 40))
	}

	// We need to decode map[string]interface{} to api.Question manually or just traverse
	for _, qRaw := range questions {
		q, _ := qRaw.(map[string]interface{})
		if c.Quiet {
			fmt.Printf("[?] Request ID: %v\n", q["id"])
		} else {
			fmt.Printf("[?] Request ID: %v\n", q["id"])
		}

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
	if !c.Quiet {
		fmt.Printf("\nRun: `opencode_skill %s /answer ...`\n", c.fullSessionRef())
	}
}

func (c *Client) Status() {
	// Get daemon start time first
	daemonStartTime, _ := c.getDaemonInfo()
	fmt.Printf("[TOOL_INFO] daemon last start time: %s\n", daemonStartTime)

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

// SessionStatus holds the status of a single session for wait_any
type SessionStatus struct {
	Project     string
	SessionName string
	ID          string
	State       string
	Completed   bool
	Response    interface{}
	Questions   []interface{}
}

// WaitAny polls multiple sessions and returns when any completes.
// Returns all completed sessions and statuses of all sessions.
func (c *Client) WaitAny(sessionPairs []SessionData) ([]SessionStatus, []SessionStatus, error) {
	start := time.Now()
	daemonStartTime, _ := c.getDaemonInfo()

	if !c.Quiet {
		fmt.Printf("[TOOL_INFO] daemon last start time: %s\n", daemonStartTime)
		fmt.Printf("Waiting for any session to complete (Timeout: %v)...\n", config.ClientTimeout)
		fmt.Printf("Monitoring %d sessions: ", len(sessionPairs))
		for i, s := range sessionPairs {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s/%s", s.Project, s.SessionName)
		}
		fmt.Println()
	}

	// Build session ID list for GET_MULTI_STATUS
	sessionIDs := make([]string, len(sessionPairs))
	for i, s := range sessionPairs {
		sessionIDs[i] = s.ID
	}

	for time.Since(start) < config.ClientTimeout {
		// Use GET_MULTI_STATUS to fetch all sessions in one call
		resp, err := c.SendRequest("GET_MULTI_STATUS", map[string]interface{}{
			"session_ids": sessionIDs,
		})
		if err != nil {
			fmt.Printf("Error checking status: %v\n", err)
			time.Sleep(config.PollInterval)
			continue
		}

		if status, ok := resp["status"].(string); !ok || status != "ok" {
			fmt.Printf("Daemon error: %v\n", resp["message"])
			time.Sleep(config.PollInterval)
			continue
		}

		// Parse status map: session_id -> status snapshot
		statusesRaw, _ := resp["results"].(map[string]interface{})
		allCurrentStatuses := make(map[string]*SessionStatus)
		allStatuses := make([]SessionStatus, 0, len(sessionPairs))
		completedThisRound := []SessionStatus{}

		// First pass: collect all statuses
		for _, session := range sessionPairs {
			sessionStatus, ok := statusesRaw[session.ID].(map[string]interface{})
			if !ok {
				// Session not found in response, skip
				continue
			}

			state, _ := sessionStatus["state"].(string)
			questions, _ := sessionStatus["questions"].([]interface{})
			latestResp, _ := sessionStatus["latest_response"].(map[string]interface{})

			// Check completion by state
			completed := state == string(manager.StateIdle) || state == string(manager.StateWaitingForInput) || len(questions) > 0

			currentStatus := &SessionStatus{
				Project:     session.Project,
				SessionName: session.SessionName,
				ID:          session.ID,
				State:       state,
				Completed:   completed,
				Response:    latestResp,
				Questions:   questions,
			}
			allCurrentStatuses[session.ID] = currentStatus

			// Collect completed sessions for this round
			if completed {
				completedThisRound = append(completedThisRound, *currentStatus)
			}
		}

		// Build full allStatuses list from collected statuses
		for _, session := range sessionPairs {
			if status, exists := allCurrentStatuses[session.ID]; exists {
				allStatuses = append(allStatuses, *status)
			}
		}

		// After checking all sessions, if any completed this round, return them all
		if len(completedThisRound) > 0 {
			return completedThisRound, allStatuses, nil
		}

		time.Sleep(config.PollInterval)
	}

	// Timeout - build final status snapshot using GET_MULTI_STATUS
	resp, _ := c.SendRequest("GET_MULTI_STATUS", map[string]interface{}{
		"session_ids": sessionIDs,
	})

	allStatuses := make([]SessionStatus, 0, len(sessionPairs))
	completed := []SessionStatus{}
	if statusMap, ok := resp["results"].(map[string]interface{}); ok {
		for _, session := range sessionPairs {
			sessionStatus, ok := statusMap[session.ID].(map[string]interface{})
			if !ok {
				continue
			}
			state, _ := sessionStatus["state"].(string)
			questions, _ := sessionStatus["questions"].([]interface{})
			latestResp, _ := sessionStatus["latest_response"].(map[string]interface{})

			isCompleted := state == string(manager.StateIdle) || state == string(manager.StateWaitingForInput) || len(questions) > 0
			status := SessionStatus{
				Project:     session.Project,
				SessionName: session.SessionName,
				ID:          session.ID,
				State:       state,
				Completed:   isCompleted,
				Response:    latestResp,
				Questions:   questions,
			}
			allStatuses = append(allStatuses, status)
			if isCompleted {
				completed = append(completed, status)
			}
		}
	}

	return completed, allStatuses, nil
}

// PrintWaitAnyResults prints the results of WaitAny in a formatted way
// Part 1 - Summary: Full list showing status of ALL sessions (done or not)
// Part 2 - Completed Response: The actual response content from completed sessions
func PrintWaitAnyResults(completed []SessionStatus, allStatuses []SessionStatus, quiet bool) {
	if quiet {
		// Quiet mode: just print responses
		for _, comp := range completed {
			printCompletedResponse(comp, quiet)
		}
		return
	}

	// Non-quiet mode: structured output with summary and responses required
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  WAIT_ANY RESULT")
	fmt.Println(strings.Repeat("=", 60))

	// Part 1: Summary of ALL sessions
	completedCount := len(completed)
	totalCount := len(allStatuses)
	fmt.Printf("\n[SUMMARY] %d/%d sessions completed\n\n", completedCount, totalCount)

	for _, s := range allStatuses {
		if s.Completed {
			fmt.Printf("  ✓ %s: completed (state: %s)\n", s.SessionName, s.State)
		} else {
			fmt.Printf("  - %s: still running (state: %s)\n", s.SessionName, s.State)
		}
	}

	// Part 2: Completed responses (if any)
	if len(completed) > 0 {
		fmt.Printf("\n%s\n", strings.Repeat("─", 60))
		fmt.Println("  COMPLETED RESPONSES")
		fmt.Println(strings.Repeat("─", 60))

		for _, comp := range completed {
			printCompletedResponse(comp, quiet)
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}

// printCompletedResponse prints the response content for a single completed session
func printCompletedResponse(comp SessionStatus, quiet bool) {
	// Check for questions
	if len(comp.Questions) > 0 {
		if !quiet {
			fmt.Printf("\n[%s]\n", comp.SessionName)
			fmt.Println("\n[QUESTIONS PENDING]")
		}
		for _, qRaw := range comp.Questions {
			q, _ := qRaw.(map[string]interface{})
			if !quiet {
				fmt.Printf("[?] Request ID: %v\n", q["id"])
			} else {
				fmt.Printf("[?] %s/%s - Request ID: %v\n", comp.Project, comp.SessionName, q["id"])
			}

			if subQs, ok := q["questions"].([]interface{}); ok {
				for _, subQRaw := range subQs {
					subQ, _ := subQRaw.(map[string]interface{})
					if !quiet {
						fmt.Printf("    %v\n", subQ["question"])
					} else {
						fmt.Printf("    Question: %v\n", subQ["question"])
					}

					if opts, ok := subQ["options"].([]interface{}); ok {
						for _, optRaw := range opts {
							opt, _ := optRaw.(map[string]interface{})
							label := opt["label"]
							desc := opt["description"]
							if desc != nil && desc != "" {
								if !quiet {
									fmt.Printf("      - %v: %v\n", label, desc)
								}
							}
						}
					}
				}
			}
		}
		return
	}

	// Print response
	if comp.Response == nil {
		return
	}

	if respMap, ok := comp.Response.(map[string]interface{}); ok {
		if errStr, ok := respMap["error"].(string); ok && errStr != "" {
			if !quiet {
				fmt.Printf("\n[%s]\nError: %s\n", comp.SessionName, errStr)
			} else {
				fmt.Printf("[ERROR] %s/%s: %s\n", comp.Project, comp.SessionName, errStr)
			}
			return
		}

		if res, ok := respMap["result"]; ok {
			formatted, _ := json.MarshalIndent(res, "", "  ")
			if quiet {
				fmt.Printf("[COMPLETED] %s/%s:\n%s\n", comp.Project, comp.SessionName, string(formatted))
			} else {
				fmt.Printf("\n[%s]\n", comp.SessionName)
				fmt.Println("Response:")
				fmt.Println(string(formatted))
			}
		}
	}
}
