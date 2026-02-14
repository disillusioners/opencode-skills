package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"opencode_skill/internal/api"
	"opencode_skill/internal/client"
	"opencode_skill/internal/config"
	"opencode_skill/internal/daemon"
)

func stopDaemon() bool {
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
			fmt.Printf("Stopping daemon (PID: %d)...\n", pid)
			process, err := os.FindProcess(pid)
			if err == nil {
				process.Signal(syscall.SIGKILL)
				time.Sleep(500 * time.Millisecond)
				fmt.Println("Daemon stopped.")
				return true
			}
		}
	}

	// Clean up PID file if exists
	os.Remove(config.PidFile)
	fmt.Println("No running daemon found.")
	return false
}

func startDaemon() {
	// Check if daemon is already running
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", config.DaemonPort))
	output, err := cmd.Output()

	if err == nil && len(output) > 0 {
		fmt.Println("Daemon is already running.")
		return
	}

	// Start daemon in background
	fmt.Println("Starting daemon in background...")
	executable, _ := os.Executable()
	cmd = exec.Command(executable, "--daemon")
	cmd.Dir = config.ProjectRoot
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start daemon: %v\n", err)
		return
	}
	fmt.Printf("Daemon started (PID: %d).\n", cmd.Process.Pid)
}

func restartDaemon() {
	stopDaemon()
	// Clean up PID file if exists
	os.Remove(config.PidFile)
	startDaemon()
}

func main() {
	isDaemon := flag.Bool("daemon", false, "Run as daemon")
	agent := flag.String("agent", config.DefaultAgent, "Agent name")
	model := flag.String("model", "zai-coding-plan/glm-5", "Model ID")

	flag.Parse()

	// Daemon doesn't need database access
	if *isDaemon {
		registry, err := daemon.NewRegistry(config.SessionMapFile)
		if err != nil {
			log.Fatalf("Failed to create registry: %v", err)
		}
		d := daemon.NewServer(registry)
		if err := d.Start(); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "start":
		startDaemon()
		return
	case "stop":
		stopDaemon()
		return
	case "restart":
		restartDaemon()
		return
	}

	if command == "init-session" {
		if len(args) < 4 {
			fmt.Println("Usage: opencode_skill init-session <PROJECT> <SESSION_NAME> <WORKING_DIR>")
			os.Exit(1)
		}
		project := args[1]
		sessionName := args[2]
		workingDir := args[3]

		absDir, err := filepath.Abs(workingDir)
		if err != nil {
			log.Fatalf("Invalid working directory: %v", err)
		}

		c := client.NewClient("") // No session ID needed for init
		sessionData, err := c.InitSession(project, sessionName, absDir)
		if err != nil {
			log.Fatalf("Failed to initialize session: %v", err)
		}
		fmt.Printf("[SUCCESS] Session '%s:%s' initialized with ID: %s in %s\n", project, sessionName, sessionData.ID, absDir)
		return
	}

	// Normal run: <PROJECT> <SESSION_NAME> [MESSAGE...]
	if len(args) < 2 {
		fmt.Println("Usage: opencode_skill <PROJECT> <SESSION_NAME> <MESSAGE> [options]")
		fmt.Println("   or: opencode_skill <PROJECT> <SESSION_NAME> /wait")
		fmt.Println("   or: opencode_skill <PROJECT> <SESSION_NAME> /status")
		os.Exit(1)
	}

	project := args[0]
	sessionName := args[1]
	fullSessionName := fmt.Sprintf("%s:%s", project, sessionName)
	messageParts := args[2:]

	c := client.NewClient("") // Temp client for lookup
	sessionData, err := c.GetSession(project, sessionName)
	if err != nil {
		fmt.Printf("Session '%s' not found: %v\n", fullSessionName, err)
		sessions, _ := c.ListSessions()
		if len(sessions) == 0 {
			fmt.Println("No active sessions found.")
		} else {
			fmt.Println("Recent sessions:")
			for _, s := range sessions {
				fullName := fmt.Sprintf("%s:%s", s.Project, s.SessionName)
				fmt.Printf("  - %-30s (Dir: %s)\n", fullName, s.WorkingDir)
			}
		}
		fmt.Println("\nTo create a new session, run:")
		fmt.Println("  opencode_skill init-session <PROJECT> <SESSION_NAME> <WORKING_DIR>")
		os.Exit(1)
	}

	// Now create the real client with session ID and metadata
	c = client.NewClientWithMeta(sessionData.ID, project, sessionName)

	// Ensure session is started in daemon with correct working dir
	_, err = c.SendRequest("START_SESSION", map[string]string{"working_dir": sessionData.WorkingDir})
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

		if status, ok := res["status"].(string); ok && status == "error" {
			fmt.Printf("Error: %v\n", res["message"])
			return
		}

		fmt.Printf("Answer status: %v\n", res["message"])
		fmt.Printf("[SUBMITTED] Run: opencode_skill %s %s /wait\n", project, sessionName)

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

		if status, ok := res["status"].(string); ok && status == "error" {
			fmt.Printf("Error: %v\n", res["message"])
			return
		}

		fmt.Printf("Command sent: %v\n", res["message"])
		fmt.Printf("[SUBMITTED] Run: opencode_skill %s %s /wait\n", project, sessionName)

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

		if status, ok := res["status"].(string); ok && status == "error" {
			fmt.Printf("Error: %v\n", res["message"])
			return
		}

		fmt.Printf("Prompt sent: %v\n", res["message"])
		fmt.Printf("[SUBMITTED] Run: opencode_skill %s %s /wait\n", project, sessionName)
	}
}

func parseModel(m string) api.ModelDetails {
	if strings.Contains(m, "/") {
		parts := strings.SplitN(m, "/", 2)
		return api.ModelDetails{ProviderID: parts[0], ModelID: parts[1]}
	}
	return api.ModelDetails{ProviderID: "zai-coding-plan", ModelID: m}
}

func formatSubmittedMessage(project, session string) string {
	return fmt.Sprintf("[SUBMITTED] Run: opencode_skill %s %s /wait", project, session)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  opencode_skill start")
	fmt.Println("  opencode_skill stop")
	fmt.Println("  opencode_skill restart")
	fmt.Println("  opencode_skill init-session <PROJECT> <SESSION_NAME> <WORKING_DIR>")
	fmt.Println("  opencode_skill <PROJECT> <SESSION_NAME> <MESSAGE> [options]")
	fmt.Println("  opencode_skill <PROJECT> <SESSION_NAME> /wait")
	fmt.Println("  opencode_skill <PROJECT> <SESSION_NAME> /status")
}
