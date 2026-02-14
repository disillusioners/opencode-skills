package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"opencode_wrapper/internal/api"
	"opencode_wrapper/internal/client"
	"opencode_wrapper/internal/config"
	"opencode_wrapper/internal/daemon"
)

func main() {
	isDaemon := flag.Bool("daemon", false, "Run as daemon")
	agent := flag.String("agent", config.DefaultAgent, "Agent name")
	// Model handling might need refinement but simple string for now
	// In python: --model default implies internal logic.
	// Here we can accept --model provider/model or just model
	model := flag.String("model", "zai-coding-plan/glm-5", "Model ID")

	flag.Parse()

	if *isDaemon {
		d := daemon.NewServer()
		d.Start()
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: opencode_wrapper <SESSION_NAME> <MESSAGE> [options]")
		os.Exit(1)
	}

	sessionName := args[0]
	messageParts := args[1:]

	// Resolve Session ID
	// We need a helper for this (reading/writing session map)
	sessionID := resolveSession(sessionName)

	c := client.NewClient(sessionID)

	// Ensure session is started in daemon
	_, err := c.SendRequest("START_SESSION", nil)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}

	if len(messageParts) == 0 {
		// Just checking? Or error?
		// If no message, maybe just status?
		// Python wrapper demanded message.
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

		// We need the request ID. Get Status first.
		// Similar logic to python client...
		// For brevity, skipping full implementation of answer logic here
		// But "Wait for result" loop needs to know if it should display questions.
		// Let's implement basics.

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

func resolveSession(name string) string {
	// Read map file
	// Ideally lock file, but for now simple read/write
	// Logic similar to python: read, check, if not exists -> create -> write

	// Note: Creating session requires API call.
	// Client struct has method? No, api client has.

	// We can use api.NewClient() directly here.
	apiClient := api.NewClient()

	// Load existing
	sessions := make(map[string]string)
	content, err := os.ReadFile(config.SessionMapFile)
	if err == nil {
		json.Unmarshal(content, &sessions)
	}

	if id, ok := sessions[name]; ok {
		return id
	}

	// Create new
	id, err := apiClient.CreateSession(name)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	sessions[name] = id
	bytes, _ := json.MarshalIndent(sessions, "", "  ")
	os.WriteFile(config.SessionMapFile, bytes, 0644)

	return id
}
