package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"opencode_wrapper/internal/api"
	"opencode_wrapper/internal/client"
	"opencode_wrapper/internal/config"
	"opencode_wrapper/internal/daemon"
)

type SessionData struct {
	ID         string
	WorkingDir string
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", config.SessionMapFile)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS sessions (
		"project" TEXT NOT NULL,
		"session_name" TEXT NOT NULL,
		"id" TEXT,
		"working_dir" TEXT,
		PRIMARY KEY (project, session_name)
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
}

func restartDaemon() {
	// Find and kill process using the daemon port
	fmt.Printf("Checking for process on port %d...\n", config.DaemonPort)

	// Use lsof to find process using the port
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", config.DaemonPort))
	output, err := cmd.Output()

	if err == nil && len(output) > 0 {
		pidStr := strings.TrimSpace(string(output))
		var pid int
		fmt.Sscanf(pidStr, "%d", &pid)

		if pid > 0 {
			fmt.Printf("Killing existing daemon (PID: %d)...\n", pid)
			process, err := os.FindProcess(pid)
			if err == nil {
				process.Signal(syscall.SIGKILL)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	// Clean up PID file if exists
	os.Remove(config.PidFile)

	// Start new daemon in background
	fmt.Println("Starting new daemon in background...")
	executable, _ := os.Executable()
	cmd = exec.Command(executable, "--daemon")
	cmd.Dir = config.ProjectRoot
	_ = cmd.Start()
}

func main() {
	isDaemon := flag.Bool("daemon", false, "Run as daemon")
	agent := flag.String("agent", config.DefaultAgent, "Agent name")
	model := flag.String("model", "zai-coding-plan/glm-5", "Model ID")

	flag.Parse()

	// Daemon doesn't need database access
	if *isDaemon {
		d := daemon.NewServer()
		d.Start()
		return
	}

	// Only client commands need database
	initDB()
	defer db.Close()

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	if command == "restart" {
		restartDaemon()
		return
	}

	if command == "init-session" {
		if len(args) < 4 {
			fmt.Println("Usage: opencode_wrapper init-session <PROJECT> <SESSION_NAME> <WORKING_DIR>")
			os.Exit(1)
		}
		project := args[1]
		sessionName := args[2]
		workingDir := args[3]

		absDir, err := filepath.Abs(workingDir)
		if err != nil {
			log.Fatalf("Invalid working directory: %v", err)
		}

		initSession(project, sessionName, absDir)
		return
	}

	// Normal run: <PROJECT> <SESSION_NAME> [MESSAGE...]
	if len(args) < 2 {
		fmt.Println("Usage: opencode_wrapper <PROJECT> <SESSION_NAME> <MESSAGE> [options]")
		fmt.Println("   or: opencode_wrapper <PROJECT> <SESSION_NAME> /wait")
		fmt.Println("   or: opencode_wrapper <PROJECT> <SESSION_NAME> /status")
		os.Exit(1)
	}

	project := args[0]
	sessionName := args[1]
	fullSessionName := fmt.Sprintf("%s:%s", project, sessionName)
	messageParts := args[2:]

	sessionData, ok := getSession(project, sessionName)
	if !ok {
		fmt.Printf("Session '%s' not found.\n", fullSessionName)
		listSessions()
		fmt.Println("\nTo create a new session, run:")
		fmt.Println("  opencode_wrapper init-session <PROJECT> <SESSION_NAME> <WORKING_DIR>")
		os.Exit(1)
	}

	c := client.NewClient(sessionData.ID)

	// Ensure session is started in daemon with correct working dir
	_, err := c.SendRequest("START_SESSION", map[string]string{"working_dir": sessionData.WorkingDir})
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}

	if len(messageParts) == 0 {
		fmt.Println("No message provided.")
		return
	}

	cmd := messageParts[0]

	if cmd == "/wait" {
		c.WaitForResult()
	} else if cmd == "/status" {
		c.Status()
	} else if cmd == "/answer" {
		answers := messageParts[1:]
		if len(answers) == 0 {
			fmt.Println("Usage: /answer <answer_text> ...")
			return
		}

		// Get status to find Question ID
		resp, _ := c.SendRequest("GET_STATUS", nil)
		data, _ := resp["data"].(map[string]interface{})
		qs, _ := data["questions"].([]interface{})
		if len(qs) == 0 {
			fmt.Println("No pending questions.")
			return
		}

		q, _ := qs[0].(map[string]interface{})
		reqID, _ := q["id"].(string)

		formattedAnswers := [][]string{}
		for _, a := range answers {
			formattedAnswers = append(formattedAnswers, []string{a})
		}

		payload := api.AnswerRequest{
			RequestID: reqID,
			Answers:   formattedAnswers,
		}

		res, err := c.SendRequest("ANSWER", payload)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Answer status: %v\n", res["message"])
		c.WaitForResult()

	} else if strings.HasPrefix(cmd, "/") {
		// Command
		command := cmd[1:]
		arguments := strings.Join(messageParts[1:], " ")

		payload := api.CommandRequest{
			Agent:     *agent,
			Model:     parseModel(*model),
			Command:   command,
			Arguments: arguments,
		}

		res, err := c.SendRequest("COMMAND", payload)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Command sent: %v\n", res["message"])
		c.WaitForResult()

	} else {
		// Prompt
		fullMessage := strings.Join(messageParts, " ")
		payload := api.PromptRequest{
			Agent: *agent,
			Model: parseModel(*model),
			Parts: []api.Part{{Type: "text", Text: fullMessage}},
		}

		res, err := c.SendRequest("PROMPT", payload)
		if err != nil {
			fmt.Printf("Error: %v\n", err) // e.g. "Session is busy"
			return
		}
		fmt.Printf("Prompt sent: %v\n", res["message"])
		c.WaitForResult()
	}
}

func parseModel(m string) api.ModelDetails {
	if strings.Contains(m, "/") {
		parts := strings.SplitN(m, "/", 2)
		return api.ModelDetails{ProviderID: parts[0], ModelID: parts[1]}
	}
	return api.ModelDetails{ProviderID: "zai-coding-plan", ModelID: m}
}

func getSession(project, sessionName string) (SessionData, bool) {
	var id, workingDir string
	row := db.QueryRow("SELECT id, working_dir FROM sessions WHERE project = ? AND session_name = ?", project, sessionName)
	err := row.Scan(&id, &workingDir)
	if err == sql.ErrNoRows {
		return SessionData{}, false
	} else if err != nil {
		log.Printf("Error querying session: %v", err)
		return SessionData{}, false
	}
	return SessionData{ID: id, WorkingDir: workingDir}, true
}

func initSession(project, sessionName, workingDir string) {
	// Create full session name with project prefix for display
	fullSessionName := fmt.Sprintf("%s:%s", project, sessionName)

	// Check if exists and abort old session if needed
	oldSession, exists := getSession(project, sessionName)
	if exists {
		fmt.Printf("[INFO] Session '%s' already exists (ID: %s, Dir: %s)\n", fullSessionName, oldSession.ID, oldSession.WorkingDir)

		// Abort the old OpenCode session to clean up resources
		fmt.Printf("[INFO] Aborting old OpenCode session %s...\n", oldSession.ID)
		oldApiClient := api.NewClient(oldSession.WorkingDir)
		if err := oldApiClient.AbortSession(oldSession.ID); err != nil {
			log.Printf("[WARN] Failed to abort old session: %v", err)
			// Continue anyway - we'll create the new session
		} else {
			fmt.Println("[INFO] Old session aborted successfully")
		}

		// Wait a moment for cleanup to complete
		fmt.Println("[INFO] Waiting for cleanup...")
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("[INFO] Creating new session '%s' in %s...\n", fullSessionName, workingDir)
	apiClient := api.NewClient(workingDir)
	id, err := apiClient.CreateSession(fullSessionName)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	// Upsert
	statement, err := db.Prepare("INSERT OR REPLACE INTO sessions (project, session_name, id, working_dir) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Fatalf("Failed to prepare statement: %v", err)
	}
	_, err = statement.Exec(project, sessionName, id, workingDir)
	if err != nil {
		log.Fatalf("Failed to save session: %v", err)
	}

	fmt.Printf("[SUCCESS] Session '%s' initialized with ID: %s in %s\n", fullSessionName, id, workingDir)
}

func listSessions() {
	rows, err := db.Query("SELECT project, session_name, working_dir FROM sessions ORDER BY project, session_name")
	if err != nil {
		log.Printf("Failed to list sessions: %v", err)
		return
	}
	defer rows.Close()

	var sessions []struct {
		Project     string
		SessionName string
		WorkingDir  string
	}

	for rows.Next() {
		var s struct {
			Project     string
			SessionName string
			WorkingDir  string
		}
		if err := rows.Scan(&s.Project, &s.SessionName, &s.WorkingDir); err == nil {
			sessions = append(sessions, s)
		}
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions found.")
		return
	}

	fmt.Println("Recent sessions:")
	for _, s := range sessions {
		fullName := fmt.Sprintf("%s:%s", s.Project, s.SessionName)
		fmt.Printf("  - %-30s (Dir: %s)\n", fullName, s.WorkingDir)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  opencode_wrapper restart")
	fmt.Println("  opencode_wrapper init-session <PROJECT> <SESSION_NAME> <WORKING_DIR>")
	fmt.Println("  opencode_wrapper <PROJECT> <SESSION_NAME> <MESSAGE> [options]")
	fmt.Println("  opencode_wrapper <PROJECT> <SESSION_NAME> /wait")
	fmt.Println("  opencode_wrapper <PROJECT> <SESSION_NAME> /status")
}
