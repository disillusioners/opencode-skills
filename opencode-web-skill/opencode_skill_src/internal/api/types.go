package api

// Request Types

type PromptRequest struct {
	Agent string       `json:"agent"`
	Model ModelDetails `json:"model"`
	Parts []Part       `json:"parts"`
}

type CommandRequest struct {
	Agent     string       `json:"agent"`
	Model     ModelDetails `json:"model"`
	Command   string       `json:"command"`
	Arguments string       `json:"arguments"`
	Parts     []Part       `json:"parts"`
}

type AnswerRequest struct {
	RequestID string     `json:"requestID"`
	Answers   [][]string `json:"answers"`
}

type ModelDetails struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type Part struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Response Types

type APIResponse struct {
	Status  string      `json:"status,omitempty"` // Sometimes used
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type SessionResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Question struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Questions []struct {
		Question string   `json:"question"`
		Options  []Option `json:"options,omitempty"`
	} `json:"questions"`
}

type Option struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}
