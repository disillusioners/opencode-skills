package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"opencode_wrapper/internal/config"
	"strconv"
	"strings"
)

type Client struct {
	BaseURL string
}

func NewClient() *Client {
	return &Client{
		BaseURL: config.OpenCodeURL,
	}
}

func (c *Client) CreateSession(title string) (string, error) {
	u := fmt.Sprintf("%s/session", c.BaseURL)
	payload := map[string]string{"title": title}

	respData, err := c.sendRequest("POST", u, payload)
	if err != nil {
		return "", err
	}

	var sessionResp SessionResponse
	if err := json.Unmarshal(respData, &sessionResp); err != nil {
		return "", err
	}
	return sessionResp.ID, nil
}

func (c *Client) SendPrompt(sessionID string, req PromptRequest) (interface{}, error) {
	u := fmt.Sprintf("%s/session/%s/message", c.BaseURL, sessionID)
	return c.postAndParse(u, req)
}

func (c *Client) SendCommand(sessionID string, req CommandRequest) (interface{}, error) {
	u := fmt.Sprintf("%s/session/%s/command", c.BaseURL, sessionID)
	return c.postAndParse(u, req)
}

func (c *Client) GetQuestions() ([]Question, error) {
	u := fmt.Sprintf("%s/question", c.BaseURL)
	resp, err := c.sendRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var questions []Question
	if err := json.Unmarshal(resp, &questions); err == nil {
		return questions, nil
	}

	var wrapper struct {
		Data []Question `json:"data"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil {
		return wrapper.Data, nil
	}

	return nil, fmt.Errorf("failed to parse questions response")
}

func (c *Client) AnswerQuestion(req AnswerRequest) error {
	u := fmt.Sprintf("%s/question/%s/reply", c.BaseURL, req.RequestID)
	payload := map[string]interface{}{
		"answers": req.Answers,
	}
	_, err := c.sendRequest("POST", u, payload)
	return err
}

func (c *Client) AbortSession(sessionID string) error {
	u := fmt.Sprintf("%s/session/%s/abort", c.BaseURL, sessionID)
	_, err := c.sendRequest("POST", u, map[string]interface{}{})
	return err
}

// Helpers

func (c *Client) postAndParse(u string, payload interface{}) (interface{}, error) {
	body, err := c.sendRequest("POST", u, payload)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		// Empty body implies success/null result
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Custom sendRequest using net.Dial to handle broken server responses (missing Content-Length)
func (c *Client) sendRequest(method string, u string, payload interface{}) ([]byte, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	host := parsedURL.Host
	if !strings.Contains(host, ":") {
		host = host + ":80"
	}

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Prepare Body
	var bodyBytes []byte
	if payload != nil {
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	// Prepare Request
	reqLine := fmt.Sprintf("%s %s HTTP/1.1\r\n", method, parsedURL.Path)
	headers := []string{
		fmt.Sprintf("Host: %s", host),
		"User-Agent: opencode-wrapper-go/1.0",
		"Accept: */*",
		"Content-Type: application/json",
		fmt.Sprintf("x-opencode-directory: %s", config.ProjectRoot),
		fmt.Sprintf("Content-Length: %d", len(bodyBytes)),
		"Connection: close",
		"\r\n",
	}

	reqStr := reqLine + strings.Join(headers, "\r\n")

	// Send
	log.Printf("Custom API %s %s", method, u)
	if _, err := conn.Write([]byte(reqStr)); err != nil {
		return nil, err
	}
	if len(bodyBytes) > 0 {
		if _, err := conn.Write(bodyBytes); err != nil {
			return nil, err
		}
	}

	// Read Response
	reader := bufio.NewReader(conn)

	// Read Status Line
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) >= 2 {
		statusCode, _ := strconv.Atoi(parts[1])
		log.Printf("API Response Status: %d", statusCode)
		if statusCode >= 400 {
			// Try to read body for error message?
			// Assuming short error message
			// Just return error
			return nil, fmt.Errorf("API Error %d", statusCode)
		}
	}

	// Read Headers
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break // End of headers
		}

		// Parse headers we care about
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				contentLength, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
		}
	}

	// Read Body
	if contentLength > 0 {
		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			return nil, err
		}
		return body, nil
	} else if contentLength == 0 {
		return []byte{}, nil
	} else {
		// Content-Length missing.
		// Server keeps connection open so we can't read until EOF.
		// Assume empty body for this specific server/API which violates spec.
		log.Println("Warning: Content-Length missing, assuming empty body.")
		return []byte{}, nil
	}
}
