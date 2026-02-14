package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"opencode_wrapper/internal/api"
	"opencode_wrapper/internal/config"
	"opencode_wrapper/internal/manager"
)

type Server struct {
	sessions map[string]*manager.SessionManager
	listener net.Listener
	registry *Registry
}

func NewServer(registry *Registry) *Server {
	return &Server{
		sessions: make(map[string]*manager.SessionManager),
		registry: registry,
	}
}

func (s *Server) Start() {
	if err := s.writePID(); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}

	// Auto-recover sessions from registry
	sessions, err := s.registry.List()
	if err != nil {
		log.Printf("Warning: failed to list sessions for recovery: %v", err)
	} else {
		for _, session := range sessions {
			fullID := fmt.Sprintf("%s:%s", session.Project, session.SessionName)
			sm := manager.NewSessionManager(session.ID, session.WorkingDir)
			sm.Start()
			s.sessions[session.ID] = sm
			log.Printf("Recovered session: %s (ID: %s, Dir: %s)", fullID, session.ID, session.WorkingDir)
		}
		log.Printf("Recovered %d session(s) from registry", len(sessions))
	}

	addr := fmt.Sprintf("%s:%d", config.DaemonHost, config.DaemonPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
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
				return
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
			sm := manager.NewSessionManager(req.SessionID, workingDir)
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

		if err := api.NewClient(session.WorkingDir).AbortSession(session.ID); err != nil {
			log.Printf("Failed to abort session: %v", err)
			response = map[string]interface{}{"status": "error", "message": "Failed to abort session: " + err.Error()}
			break
		}

		if err := s.registry.Delete(project, sessionName); err != nil {
			log.Printf("Failed to delete session from registry: %v", err)
		}

		log.Printf("Aborted session %s/%s", project, sessionName)
		response = map[string]interface{}{"status": "ok"}

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
			// Verify BUSY state for PROMPT
			if req.Action == "PROMPT" {
				snapshot := sm.GetSnapshot()
				state, _ := snapshot["state"].(manager.State)

				// Check if special prompt
				isSpecial := false
				if parts, ok := req.Payload["parts"].([]interface{}); ok && len(parts) > 0 {
					if partMap, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := partMap["text"].(string); ok {
							if text == "start-work" || text == "continue" || text == "abort" || text == "retry" {
								isSpecial = true
							}
						}
					}
				}

				if state == manager.StateBusy && !isSpecial {
					response = map[string]interface{}{"status": "error", "message": "Session is busy"}
					break // break switch, send response
				}
			}

			// Convert payload to specific types
			var internalPayload interface{}
			payloadBytes, _ := json.Marshal(req.Payload) // Re-marshal to unmarshal into struct

			if req.Action == "PROMPT" {
				var p api.PromptRequest
				json.Unmarshal(payloadBytes, &p)
				internalPayload = p
			} else if req.Action == "COMMAND" {
				var p api.CommandRequest
				json.Unmarshal(payloadBytes, &p)
				internalPayload = p
			} else if req.Action == "ANSWER" {
				var p api.AnswerRequest
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
