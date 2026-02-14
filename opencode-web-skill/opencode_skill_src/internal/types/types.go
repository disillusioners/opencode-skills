package types

// Request Types - shared between CLI and daemon

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
