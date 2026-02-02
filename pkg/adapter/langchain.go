package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LangChainAdapter defines the interface for LangChain integration.
type LangChainAdapter interface {
	// ExecuteChain executes a LangChain chain and returns the result.
	ExecuteChain(
		ctx context.Context,
		chainType string,
		prompt string,
		variables map[string]interface{},
	) (*ChainResult, error)
	// Decompose decomposes a task into subtasks.
	Decompose(
		ctx context.Context,
		task string,
		maxSteps int,
	) (*DecomposeResult, error)
	// Health checks if the service is available.
	Health(ctx context.Context) error
}

// ChainResult represents a chain execution result.
type ChainResult struct {
	// Result is the chain output.
	Result string `json:"result"`
	// Steps are the execution steps.
	Steps []ChainStep `json:"steps"`
}

// ChainStep represents a step in chain execution.
type ChainStep struct {
	// Step is the step name.
	Step string `json:"step"`
	// Input is the step input.
	Input string `json:"input,omitempty"`
	// Output is the step output.
	Output string `json:"output,omitempty"`
}

// DecomposeResult represents a task decomposition result.
type DecomposeResult struct {
	// Subtasks are the decomposed subtasks.
	Subtasks []Subtask `json:"subtasks"`
	// Reasoning explains the decomposition.
	Reasoning string `json:"reasoning"`
}

// Subtask represents a decomposed subtask.
type Subtask struct {
	// ID is the subtask ID.
	ID int `json:"id"`
	// Description describes the subtask.
	Description string `json:"description"`
	// Dependencies are IDs of prerequisite subtasks.
	Dependencies []int `json:"dependencies"`
	// Complexity indicates the subtask complexity.
	Complexity string `json:"complexity"`
}

// LangChainConfig holds configuration for the LangChain adapter.
type LangChainConfig struct {
	// BaseURL is the LangChain service URL.
	BaseURL string `json:"base_url"`
	// Timeout is the request timeout.
	Timeout time.Duration `json:"timeout"`
}

// DefaultLangChainConfig returns a default configuration.
func DefaultLangChainConfig() *LangChainConfig {
	return &LangChainConfig{
		BaseURL: "http://localhost:8011",
		Timeout: 120 * time.Second,
	}
}

// LangChainHTTPAdapter implements LangChainAdapter using HTTP.
type LangChainHTTPAdapter struct {
	config     *LangChainConfig
	httpClient *http.Client
}

// NewLangChainHTTPAdapter creates a new LangChain HTTP adapter.
func NewLangChainHTTPAdapter(config *LangChainConfig) *LangChainHTTPAdapter {
	if config == nil {
		config = DefaultLangChainConfig()
	}
	return &LangChainHTTPAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// ExecuteChain executes a LangChain chain via HTTP.
func (a *LangChainHTTPAdapter) ExecuteChain(
	ctx context.Context,
	chainType string,
	prompt string,
	variables map[string]interface{},
) (*ChainResult, error) {
	type chainReq struct {
		ChainType   string                 `json:"chain_type"`
		Prompt      string                 `json:"prompt"`
		Variables   map[string]interface{} `json:"variables,omitempty"`
		Temperature float64                `json:"temperature"`
	}

	resp, err := a.doRequest(ctx, "POST", "/chain", &chainReq{
		ChainType:   chainType,
		Prompt:      prompt,
		Variables:   variables,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result ChainResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// Decompose decomposes a task via the LangChain service.
func (a *LangChainHTTPAdapter) Decompose(
	ctx context.Context,
	task string,
	maxSteps int,
) (*DecomposeResult, error) {
	if maxSteps <= 0 {
		maxSteps = 5
	}

	type decomposeReq struct {
		Task     string `json:"task"`
		MaxSteps int    `json:"max_steps"`
	}

	resp, err := a.doRequest(ctx, "POST", "/decompose", &decomposeReq{
		Task:     task,
		MaxSteps: maxSteps,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result DecomposeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// Health checks if the LangChain service is available.
func (a *LangChainHTTPAdapter) Health(ctx context.Context) error {
	resp, err := a.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (a *LangChainHTTPAdapter) doRequest(
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
		ctx, method, a.config.BaseURL+path, bodyReader,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.httpClient.Do(req)
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
