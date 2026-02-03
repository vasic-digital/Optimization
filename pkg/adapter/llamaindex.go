// Package adapter provides LLM framework adapters for integrating
// with LlamaIndex, LangChain, and other frameworks via HTTP clients.
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

// LlamaIndexAdapter defines the interface for LlamaIndex integration.
type LlamaIndexAdapter interface {
	// Query queries documents and returns an answer with sources.
	Query(ctx context.Context, query string, topK int) (*QueryResult, error)
	// Rerank reranks documents by query relevance.
	Rerank(
		ctx context.Context,
		query string,
		documents []string,
		topK int,
	) ([]RankedDocument, error)
	// Health checks if the service is available.
	Health(ctx context.Context) error
}

// QueryResult represents a document query result.
type QueryResult struct {
	// Answer is the generated answer.
	Answer string `json:"answer"`
	// Sources are the source documents used.
	Sources []Source `json:"sources"`
	// Confidence is the confidence score.
	Confidence float64 `json:"confidence"`
}

// Source represents a document source.
type Source struct {
	// Content is the source content.
	Content string `json:"content"`
	// Score is the relevance score.
	Score float64 `json:"score"`
	// Metadata contains additional metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RankedDocument represents a reranked document.
type RankedDocument struct {
	// Content is the document content.
	Content string `json:"content"`
	// Score is the relevance score.
	Score float64 `json:"score"`
	// Rank is the position in the ranking.
	Rank int `json:"rank"`
}

// LlamaIndexConfig holds configuration for the LlamaIndex adapter.
type LlamaIndexConfig struct {
	// BaseURL is the LlamaIndex service URL.
	BaseURL string `json:"base_url"`
	// Timeout is the request timeout.
	Timeout time.Duration `json:"timeout"`
}

// DefaultLlamaIndexConfig returns a default configuration.
func DefaultLlamaIndexConfig() *LlamaIndexConfig {
	return &LlamaIndexConfig{
		BaseURL: "http://localhost:8012",
		Timeout: 120 * time.Second,
	}
}

// LlamaIndexHTTPAdapter implements LlamaIndexAdapter using HTTP.
type LlamaIndexHTTPAdapter struct {
	config      *LlamaIndexConfig
	httpClient  *http.Client
	marshalJSON jsonMarshaler
}

// NewLlamaIndexHTTPAdapter creates a new LlamaIndex HTTP adapter.
func NewLlamaIndexHTTPAdapter(config *LlamaIndexConfig) *LlamaIndexHTTPAdapter {
	if config == nil {
		config = DefaultLlamaIndexConfig()
	}
	return &LlamaIndexHTTPAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		marshalJSON: json.Marshal,
	}
}

// Query queries documents via the LlamaIndex service.
func (a *LlamaIndexHTTPAdapter) Query(
	ctx context.Context,
	query string,
	topK int,
) (*QueryResult, error) {
	if topK <= 0 {
		topK = 5
	}

	type queryReq struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}

	resp, err := a.doRequest(ctx, "POST", "/query", &queryReq{
		Query: query,
		TopK:  topK,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// Rerank reranks documents via the LlamaIndex service.
func (a *LlamaIndexHTTPAdapter) Rerank(
	ctx context.Context,
	query string,
	documents []string,
	topK int,
) ([]RankedDocument, error) {
	if topK <= 0 {
		topK = 5
	}

	type rerankReq struct {
		Query     string   `json:"query"`
		Documents []string `json:"documents"`
		TopK      int      `json:"top_k"`
	}

	resp, err := a.doRequest(ctx, "POST", "/rerank", &rerankReq{
		Query:     query,
		Documents: documents,
		TopK:      topK,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		RankedDocuments []RankedDocument `json:"ranked_documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result.RankedDocuments, nil
}

// Health checks if the LlamaIndex service is available.
func (a *LlamaIndexHTTPAdapter) Health(ctx context.Context) error {
	resp, err := a.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (a *LlamaIndexHTTPAdapter) doRequest(
	ctx context.Context,
	method, path string,
	body interface{},
) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		marshal := a.marshalJSON
		if marshal == nil {
			marshal = json.Marshal
		}
		data, err := marshal(body)
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
