package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"opencode_skill/internal/api"
	"opencode_skill/internal/config"
	"opencode_skill/internal/manager"
	"opencode_skill/internal/types"
)

type Server struct {
	sessions map[string]*manager.SessionManager
	listener net.Listener
	registry *Registry
	port     int
}

func NewServer(registry *Registry) *Server {
	return &Server{
		sessions: make(map[string]*manager.SessionManager),
		registry: registry,
		port:     config.DaemonPort,
	}
}

func NewServerWithPort(registry *Registry, port int) *Server {
	return &Server{
		sessions: make(map[string]*manager.SessionManager),
		registry: registry,
		port:     port,
	}
}

func (s *Server) Start() error {
	// Clean up stale PID file if process is dead
	cleanupStalePID()

	// Check if daemon is already running
	if isDaemonRunning() {
		return fmt.Errorf("daemon is already running on port %d", s.port)
	}

	if err := s.writePID(); err != nil {
		return fmt.Errorf("failed to write PID file: %v", err)
	}

	// Auto-recover sessions from registry
	sessions, err := s.registry.List()
	if err != nil {
		log.Printf("Warning: failed to list sessions for recovery: %v", err)
	} else {
		for _, session := range sessions {
			fullData, err := s.registry.Get(session.Project, session.SessionName)
			if err != nil {
				log.Printf("Warning: failed to get full data for session %s/%s: %v", session.Project, session.SessionName, err)
				fullData = &session
			}

			persistedState := &manager.PersistedState{
				LastAgent:      fullData.LastAgent,
				IsAgentLocked:  fullData.IsAgentLocked,
				State:          fullData.State,
				LatestResponse: fullData.LatestResponse,
				Questions:      fullData.Questions,
				LastActivity:   fullData.LastActivity,
			}

			sm := manager.NewSessionManager(session.ID, session.WorkingDir, persistedState)
			sm.Start()
			s.sessions[session.ID] = sm
			log.Printf("Recovered session: %s %s (ID: %s, Dir: %s, State: %s)", session.Project, session.SessionName, session.ID, session.WorkingDir, fullData.State)
		}
		log.Printf("Recovered %d session(s) from registry", len(sessions))
	}

	addr := net.JoinHostPort(config.DaemonHost, strconv.Itoa(s.port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", addr, err)
	}
	s.listener = ln
	log.Printf("Daemon listening on %s", addr)

	// Handle signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		s.Stop()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Check if closed
			select {
			case <-c:
				return nil
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}

}

func (s *Server) Stop() {
	log.Println("Stopping daemon...")
	if s.listener != nil {
		s.listener.Close()
	}

	// Stop all managers
	for _, sm := range s.sessions {
		sm.Stop()
	}

	if err := os.Remove(config.PidFile); err != nil {
		log.Printf("Failed to remove PID file: %v", err)
	}
	os.Exit(0)
}

func (s *Server) handleConnection(conn net.Conn) {
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
		s.sendError(conn, "Invalid JSON")
		return
	}

	response := map[string]interface{}{"status": "error", "message": "Unknown action"}

	switch req.Action {
	case "PING":
		response = map[string]interface{}{"status": "ok", "message": "PONG"}

	case "START_SESSION":
		workingDir, _ := req.Payload["working_dir"].(string)
		if workingDir == "" {
			workingDir = config.ProjectRoot
		}

		if sm, exists := s.sessions[req.SessionID]; exists {
			sm.UpdateWorkingDir(workingDir)
			log.Printf("Updated working dir for session %s to %s", req.SessionID, workingDir)
		} else {
			sm := manager.NewSessionManager(req.SessionID, workingDir, nil)
			sm.Start()
			s.sessions[req.SessionID] = sm
			log.Printf("Started manager for session %s with dir %s", req.SessionID, workingDir)
		}
		response = map[string]interface{}{"status": "ok", "message": "Session managed"}

	case "GET_STATUS":
		if sm, ok := s.sessions[req.SessionID]; ok {
			response = map[string]interface{}{"status": "ok", "data": sm.GetSnapshot()}
		} else {
			response = map[string]interface{}{"status": "error", "message": "Session not found"}
		}

	case "INIT_SESSION":
		project, _ := req.Payload["project"].(string)
		sessionName, _ := req.Payload["session_name"].(string)
		workingDir, _ := req.Payload["working_dir"].(string)

		if project == "" || sessionName == "" {
			response = map[string]interface{}{"status": "error", "message": "project and session_name are required"}
			break
		}

		if workingDir == "" {
			workingDir = config.ProjectRoot
		}

		if existing, err := s.registry.Get(project, sessionName); err == nil {
			log.Printf("Session %s/%s exists, aborting old session %s", project, sessionName, existing.ID)
			if err := api.NewClient(existing.WorkingDir).AbortSession(existing.ID); err != nil {
				log.Printf("Failed to abort old session: %v", err)
			}
			if err := s.registry.Delete(project, sessionName); err != nil {
				log.Printf("Failed to delete old session: %v", err)
			}
		}

		client := api.NewClient(workingDir)
		sessionID, err := client.CreateSession(sessionName)
		if err != nil {
			response = map[string]interface{}{"status": "error", "message": "Failed to create session: " + err.Error()}
			break
		}

		if err := s.registry.Create(project, sessionName, sessionID, workingDir); err != nil {
			log.Printf("Failed to save session to registry: %v", err)
			response = map[string]interface{}{"status": "error", "message": "Failed to save session: " + err.Error()}
			break
		}

		log.Printf("Initialized session %s/%s with ID %s", project, sessionName, sessionID)
		response = map[string]interface{}{"status": "ok", "session_id": sessionID}

	case "ABORT_SESSION":
		project, _ := req.Payload["project"].(string)
		sessionName, _ := req.Payload["session_name"].(string)

		if project == "" || sessionName == "" {
			response = map[string]interface{}{"status": "error", "message": "project and session_name are required"}
			break
		}

		session, err := s.registry.Get(project, sessionName)
		if err != nil {
			response = map[string]interface{}{"status": "error", "message": "Session not found"}
			break
		}

		// Call remote abort
		abortErr := api.NewClient(session.WorkingDir).AbortSession(session.ID)
		if abortErr != nil {
			log.Printf("Warning: Failed to abort remote session: %v", abortErr)
		} else {
			// Wait for remote abort to propagate
			time.Sleep(3 * time.Second)
		}

		// Reset local manager state
		if sm, exists := s.sessions[session.ID]; exists {
			sm.AbortTask()
		}

		log.Printf("Aborted tasks for session %s/%s", project, sessionName)
		if abortErr != nil {
			response = map[string]interface{}{"status": "ok", "message": "Local tasks aborted, but remote abort failed: " + abortErr.Error()}
		} else {
			response = map[string]interface{}{"status": "ok", "message": "Session aborted and ready for new input"}
		}

	case "LIST_SESSIONS":
		sessions, err := s.registry.List()
		if err != nil {
			response = map[string]interface{}{"status": "error", "message": "Failed to list sessions: " + err.Error()}
			break
		}
		response = map[string]interface{}{"status": "ok", "sessions": sessions}

	case "GET_SESSION":
		project, _ := req.Payload["project"].(string)
		sessionName, _ := req.Payload["session_name"].(string)

		if project == "" || sessionName == "" {
			response = map[string]interface{}{"status": "error", "message": "project and session_name are required"}
			break
		}

		session, err := s.registry.Get(project, sessionName)
		if err != nil {
			response = map[string]interface{}{"status": "error", "message": "not found"}
			break
		}
		response = map[string]interface{}{"status": "ok", "session": session}

	case "PROMPT", "COMMAND", "ANSWER", "FIX":
		if sm, ok := s.sessions[req.SessionID]; ok {
			// Extract text content for special handling regarding busy state and agent locking
			targetText := ""
			if req.Action == "PROMPT" {
				if parts, ok := req.Payload["parts"].([]interface{}); ok && len(parts) > 0 {
					if partMap, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := partMap["text"].(string); ok {
							targetText = text
						}
					}
				}
			} else if req.Action == "COMMAND" {
				if cmd, ok := req.Payload["command"].(string); ok {
					targetText = cmd
				}
			}

			// Normalize text (trim slash if present locally to handle both /cmd and cmd styles)
			normalizedText := strings.TrimPrefix(targetText, "/")

			// Handle /start-work: lock agent to atlas
			if normalizedText == "start-work" {
				if sessionData, err := s.registry.FindByID(req.SessionID); err == nil {
					if err := s.registry.UpdateAgentState(sessionData.Project, sessionData.SessionName, "atlas", true); err != nil {
						log.Printf("Failed to lock agent for session %s: %v", req.SessionID, err)
					} else {
						log.Printf("Locked agent to 'atlas' for session %s", req.SessionID)
					}
				}
			}

			// Verify BUSY state for PROMPT
			if req.Action == "PROMPT" {
				snapshot := sm.GetSnapshot()
				state, _ := snapshot["state"].(manager.State)

				// Check if special prompt
				isSpecial := false
				if normalizedText == "start-work" || normalizedText == "continue" || normalizedText == "abort" || normalizedText == "retry" {
					isSpecial = true
				}

				if state == manager.StateBusy && !isSpecial {
					response = map[string]interface{}{"status": "error", "message": "Session is busy. Please patience wait for the previous message result before send new message."}
					break // break switch, send response
				}
			}

			if req.Action == "PROMPT" || req.Action == "COMMAND" {
				if sessionData, err := s.registry.FindByID(req.SessionID); err == nil {
					if sessionData.IsAgentLocked && sessionData.LastAgent != "" {
						req.Payload["agent"] = sessionData.LastAgent
						log.Printf("Using locked agent '%s' for session %s", sessionData.LastAgent, req.SessionID)
					}
				}
			}

			// Convert payload to specific types
			var internalPayload interface{}
			payloadBytes, _ := json.Marshal(req.Payload) // Re-marshal to unmarshal into struct

			if req.Action == "PROMPT" {
				var p types.PromptRequest
				json.Unmarshal(payloadBytes, &p)
				internalPayload = p
			} else if req.Action == "COMMAND" {
				var p types.CommandRequest
				json.Unmarshal(payloadBytes, &p)
				internalPayload = p
			} else if req.Action == "ANSWER" {
				var p types.AnswerRequest
				json.Unmarshal(payloadBytes, &p)
				internalPayload = p
			}

			sm.SubmitRequest(manager.Request{Type: req.Action, Payload: internalPayload})
			response = map[string]interface{}{"status": "ok", "message": "Request submitted"}

		} else {
			response = map[string]interface{}{"status": "error", "message": "Session not found"}
		}
	}

	s.sendResponse(conn, response)
}

func (s *Server) sendResponse(conn net.Conn, resp map[string]interface{}) {
	bytes, _ := json.Marshal(resp)
	conn.Write(bytes)
}

func (s *Server) sendError(conn net.Conn, msg string) {
	s.sendResponse(conn, map[string]interface{}{"status": "error", "message": msg})
}

func (s *Server) writePID() error {
	pid := os.Getpid()
	return os.WriteFile(config.PidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func isDaemonRunning() bool {
	addr := net.JoinHostPort(config.DaemonHost, strconv.Itoa(config.DaemonPort))
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		conn.Close()
		return true
	}

	pidBytes, err := os.ReadFile(config.PidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func cleanupStalePID() {
	pidBytes, err := os.ReadFile(config.PidFile)
	if err != nil {
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		os.Remove(config.PidFile)
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(config.PidFile)
		return
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		os.Remove(config.PidFile)
	}
}
