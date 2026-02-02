// Package sglang provides an integration interface for SGLang,
// an efficient LLM serving framework with RadixAttention for prefix caching.
package sglang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client defines the interface for SGLang generation.
type Client interface {
	// Generate executes a program and returns the generated text.
	Generate(ctx context.Context, program *Program) (string, error)
	// Health checks if the SGLang service is available.
	Health(ctx context.Context) error
}

// Program represents an SGLang program with instructions.
type Program struct {
	// SystemPrompt is the system prompt for the generation.
	SystemPrompt string `json:"system_prompt,omitempty"`
	// UserPrompt is the user prompt.
	UserPrompt string `json:"user_prompt"`
	// Temperature controls randomness (0-2).
	Temperature float64 `json:"temperature,omitempty"`
	// MaxTokens is the maximum tokens to generate.
	MaxTokens int `json:"max_tokens,omitempty"`
	// TopP is nucleus sampling probability.
	TopP float64 `json:"top_p,omitempty"`
	// Stop is a list of stop sequences.
	Stop []string `json:"stop,omitempty"`
}

// Config holds configuration for the SGLang client.
type Config struct {
	// Endpoint is the SGLang server URL.
	Endpoint string `json:"endpoint"`
	// Model is the model to use.
	Model string `json:"model,omitempty"`
	// Timeout is the request timeout.
	Timeout time.Duration `json:"timeout"`
}

// DefaultConfig returns a default SGLang configuration.
func DefaultConfig() *Config {
	return &Config{
		Endpoint: "http://localhost:30000",
		Timeout:  120 * time.Second,
	}
}

// message represents a chat message.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// completionRequest is the request to the SGLang API.
type completionRequest struct {
	Model       string    `json:"model,omitempty"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// completionChoice is a response choice.
type completionChoice struct {
	Message message `json:"message"`
}

// completionResponse is the response from the SGLang API.
type completionResponse struct {
	Choices []completionChoice `json:"choices"`
}

// HTTPClient implements Client using HTTP requests to an SGLang server.
type HTTPClient struct {
	config     *Config
	httpClient *http.Client
}

// NewHTTPClient creates a new SGLang HTTP client.
func NewHTTPClient(config *Config) *HTTPClient {
	if config == nil {
		config = DefaultConfig()
	}
	return &HTTPClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Generate executes a program via the SGLang API.
func (c *HTTPClient) Generate(
	ctx context.Context,
	program *Program,
) (string, error) {
	if program == nil {
		return "", fmt.Errorf("program must not be nil")
	}

	messages := make([]message, 0, 2)
	if program.SystemPrompt != "" {
		messages = append(messages, message{
			Role:    "system",
			Content: program.SystemPrompt,
		})
	}
	messages = append(messages, message{
		Role:    "user",
		Content: program.UserPrompt,
	})

	temp := program.Temperature
	if temp == 0 {
		temp = 0.7
	}
	maxTokens := program.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	req := &completionRequest{
		Model:       c.config.Model,
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
		TopP:        program.TopP,
		Stop:        program.Stop,
	}

	resp, err := c.doRequest(ctx, "POST", "/v1/chat/completions", req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result completionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", nil
	}

	return result.Choices[0].Message.Content, nil
}

// Health checks if the SGLang service is available.
func (c *HTTPClient) Health(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (c *HTTPClient) doRequest(
	ctx context.Context,
	method, path string,
	body interface{},
) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(
		ctx, method, c.config.Endpoint+path, bodyReader,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer func() { _ = resp.Body.Close() }()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"request failed with status %d: %s",
			resp.StatusCode, string(bodyBytes),
		)
	}

	return resp, nil
}
