package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opencode_skill/internal/config"
	"opencode_skill/internal/types"
	"time"
)

type Client struct {
	BaseURL    string
	WorkingDir string
	httpClient *http.Client
}

func NewClient(workingDir string) *Client {
	return &Client{
		BaseURL:    config.OpenCodeURL,
		WorkingDir: workingDir,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *Client) CreateSession(title string) (string, error) {
	u := fmt.Sprintf("%s/session", c.BaseURL)
	payload := map[string]string{"title": title}

	bodyBytes, err := c.doRequest("POST", u, payload)
	if err != nil {
		return "", err
	}

	var sessionResp SessionResponse
	if err := json.Unmarshal(bodyBytes, &sessionResp); err != nil {
		return "", err
	}
	return sessionResp.ID, nil
}

func (c *Client) doRequest(method, url string, payload interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "opencode-wrapper-go/1.0")
	req.Header.Set("x-opencode-directory", c.WorkingDir)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API Error %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) postAndParse(u string, payload interface{}) (interface{}, error) {
	body, err := c.doRequest("POST", u, payload)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) SendPrompt(sessionID string, req types.PromptRequest) (interface{}, error) {
	u := fmt.Sprintf("%s/session/%s/message", c.BaseURL, sessionID)
	return c.postAndParse(u, req)
}

func (c *Client) SendCommand(sessionID string, req types.CommandRequest) (interface{}, error) {
	u := fmt.Sprintf("%s/session/%s/command", c.BaseURL, sessionID)
	return c.postAndParse(u, req)
}

func (c *Client) GetQuestions() ([]Question, error) {
	u := fmt.Sprintf("%s/question", c.BaseURL)
	resp, err := c.doRequest("GET", u, nil)
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

func (c *Client) AnswerQuestion(req types.AnswerRequest) error {
	u := fmt.Sprintf("%s/question/%s/reply", c.BaseURL, req.RequestID)
	payload := map[string]interface{}{
		"answers": req.Answers,
	}
	_, err := c.doRequest("POST", u, payload)
	return err
}

func (c *Client) AbortSession(sessionID string) error {
	u := fmt.Sprintf("%s/session/%s/abort", c.BaseURL, sessionID)
	_, err := c.doRequest("POST", u, map[string]interface{}{})
	return err
}
