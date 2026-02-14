package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
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
		"name" TEXT NOT NULL PRIMARY KEY,
		"id" TEXT,
		"working_dir" TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
}

func restartDaemon() {
	// Try to stop existing daemon
	pidData, err := os.ReadFile(config.PidFile)
	if err == nil {
		var pid int
		fmt.Sscanf(string(pidData), "%d", &pid)

		// Check if process exists and kill it
		process, err := os.FindProcess(pid)
		if err == nil {
			fmt.Printf("Stopping existing daemon (PID: %d)...\n", pid)
			process.Signal(syscall.SIGTERM)

			// Wait a bit for graceful shutdown
			for i := 0; i < 10; i++ {
				if err := process.Signal(syscall.Signal(0)); err != nil {
					// Process no longer exists
					break
				}
				fmt.Print(".")
				os.Stdout.Sync()
				time.Sleep(100 * time.Millisecond)
			}
			fmt.Println()
		}
	}

	// Start new daemon
	fmt.Println("Starting new daemon...")
	d := daemon.NewServer()
	d.Start()
}

func main() {
	initDB()
	defer db.Close()

	isDaemon := flag.Bool("daemon", false, "Run as daemon")
	agent := flag.String("agent", config.DefaultAgent, "Agent name")
	model := flag.String("model", "zai-coding-plan/glm-5", "Model ID")

	flag.Parse()

	if *isDaemon {
		d := daemon.NewServer()
		d.Start()
		return
	}

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
		if len(args) < 3 {
			fmt.Println("Usage: opencode_wrapper init-session <SESSION_NAME> <WORKING_DIR>")
			os.Exit(1)
		}
		sessionName := args[1]
		workingDir := args[2]

		absDir, err := filepath.Abs(workingDir)
		if err != nil {
			log.Fatalf("Invalid working directory: %v", err)
		}

		initSession(sessionName, absDir)
		return
	}

	// Normal run: <SESSION_NAME> [MESSAGE...]
	sessionName := args[0]
	messageParts := args[1:]

	sessionData, ok := getSession(sessionName)
	if !ok {
		fmt.Printf("Session '%s' not found.\n", sessionName)
		listSessions()
		fmt.Println("\nTo create a new session, run:")
		fmt.Println("  opencode_wrapper init-session <SESSION_NAME> <WORKING_DIR>")
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

func getSession(name string) (SessionData, bool) {
	var id, workingDir string
	row := db.QueryRow("SELECT id, working_dir FROM sessions WHERE name = ?", name)
	err := row.Scan(&id, &workingDir)
	if err == sql.ErrNoRows {
		return SessionData{}, false
	} else if err != nil {
		log.Printf("Error querying session: %v", err)
		return SessionData{}, false
	}
	return SessionData{ID: id, WorkingDir: workingDir}, true
}

func initSession(name, workingDir string) {
	// Check if exists and abort old session if needed
	oldSession, exists := getSession(name)
	if exists {
		fmt.Printf("[INFO] Session '%s' already exists (ID: %s, Dir: %s)\n", name, oldSession.ID, oldSession.WorkingDir)

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

	fmt.Printf("[INFO] Creating new session '%s' in %s...\n", name, workingDir)
	apiClient := api.NewClient(workingDir)
	id, err := apiClient.CreateSession(name)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	// Upsert
	statement, err := db.Prepare("INSERT OR REPLACE INTO sessions (name, id, working_dir) VALUES (?, ?, ?)")
	if err != nil {
		log.Fatalf("Failed to prepare statement: %v", err)
	}
	_, err = statement.Exec(name, id, workingDir)
	if err != nil {
		log.Fatalf("Failed to save session: %v", err)
	}

	fmt.Printf("[SUCCESS] Session '%s' initialized with ID: %s in %s\n", name, id, workingDir)
}

func listSessions() {
	rows, err := db.Query("SELECT name, working_dir FROM sessions ORDER BY name")
	if err != nil {
		log.Printf("Failed to list sessions: %v", err)
		return
	}
	defer rows.Close()

	var sessions []struct {
		Name       string
		WorkingDir string
	}

	for rows.Next() {
		var s struct {
			Name       string
			WorkingDir string
		}
		if err := rows.Scan(&s.Name, &s.WorkingDir); err == nil {
			sessions = append(sessions, s)
		}
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions found.")
		return
	}

	fmt.Println("Recent sessions:")
	for _, s := range sessions {
		fmt.Printf("  - %-20s (Dir: %s)\n", s.Name, s.WorkingDir)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  opencode_wrapper restart")
	fmt.Println("  opencode_wrapper init-session <SESSION_NAME> <WORKING_DIR>")
	fmt.Println("  opencode_wrapper <SESSION_NAME> <MESSAGE> [options]")
	fmt.Println("  opencode_wrapper <SESSION_NAME> /wait")
	fmt.Println("  opencode_wrapper <SESSION_NAME> /status")
}
