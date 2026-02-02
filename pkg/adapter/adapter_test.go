package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLlamaIndexHTTPAdapter_Query(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		topK       int
		serverResp QueryResult
		statusCode int
		wantErr    bool
	}{
		{
			name:  "successful query",
			query: "What is Go?",
			topK:  3,
			serverResp: QueryResult{
				Answer: "Go is a programming language.",
				Sources: []Source{
					{Content: "Go docs", Score: 0.95},
				},
				Confidence: 0.9,
			},
			statusCode: http.StatusOK,
		},
		{
			name:       "server error",
			query:      "test",
			topK:       1,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:  "default topK applied",
			query: "test",
			topK:  0, // Should default to 5.
			serverResp: QueryResult{
				Answer: "answer",
			},
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/query", r.URL.Path)
					w.WriteHeader(tt.statusCode)
					if tt.statusCode == http.StatusOK {
						_ = json.NewEncoder(w).Encode(tt.serverResp)
					}
				}),
			)
			defer server.Close()

			adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
				BaseURL: server.URL,
				Timeout: 5 * time.Second,
			})

			result, err := adapter.Query(
				context.Background(), tt.query, tt.topK,
			)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.serverResp.Answer, result.Answer)
			}
		})
	}
}

func TestLlamaIndexHTTPAdapter_Rerank(t *testing.T) {
	serverResp := struct {
		RankedDocuments []RankedDocument `json:"ranked_documents"`
	}{
		RankedDocuments: []RankedDocument{
			{Content: "doc1", Score: 0.9, Rank: 1},
			{Content: "doc2", Score: 0.7, Rank: 2},
		},
	}

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rerank", r.URL.Path)
			_ = json.NewEncoder(w).Encode(serverResp)
		}),
	)
	defer server.Close()

	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	docs, err := adapter.Rerank(
		context.Background(),
		"query",
		[]string{"doc1", "doc2", "doc3"},
		2,
	)
	require.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Equal(t, "doc1", docs[0].Content)
}

func TestLlamaIndexHTTPAdapter_Health(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/health", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	err := adapter.Health(context.Background())
	assert.NoError(t, err)
}

func TestLangChainHTTPAdapter_ExecuteChain(t *testing.T) {
	tests := []struct {
		name       string
		chainType  string
		prompt     string
		serverResp ChainResult
		statusCode int
		wantErr    bool
	}{
		{
			name:      "successful chain execution",
			chainType: "summary",
			prompt:    "Summarize this text.",
			serverResp: ChainResult{
				Result: "Summary here.",
				Steps: []ChainStep{
					{Step: "analyze", Output: "analyzed"},
				},
			},
			statusCode: http.StatusOK,
		},
		{
			name:       "server error",
			chainType:  "test",
			prompt:     "test",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/chain", r.URL.Path)
					w.WriteHeader(tt.statusCode)
					if tt.statusCode == http.StatusOK {
						_ = json.NewEncoder(w).Encode(tt.serverResp)
					}
				}),
			)
			defer server.Close()

			adapter := NewLangChainHTTPAdapter(&LangChainConfig{
				BaseURL: server.URL,
				Timeout: 5 * time.Second,
			})

			result, err := adapter.ExecuteChain(
				context.Background(),
				tt.chainType,
				tt.prompt,
				nil,
			)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.serverResp.Result, result.Result)
			}
		})
	}
}

func TestLangChainHTTPAdapter_Decompose(t *testing.T) {
	serverResp := DecomposeResult{
		Subtasks: []Subtask{
			{ID: 1, Description: "step 1", Complexity: "low"},
			{ID: 2, Description: "step 2", Dependencies: []int{1}},
		},
		Reasoning: "decomposed logically",
	}

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/decompose", r.URL.Path)
			_ = json.NewEncoder(w).Encode(serverResp)
		}),
	)
	defer server.Close()

	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	result, err := adapter.Decompose(context.Background(), "build app", 5)
	require.NoError(t, err)
	assert.Len(t, result.Subtasks, 2)
	assert.Equal(t, "decomposed logically", result.Reasoning)
}

func TestLangChainHTTPAdapter_Health(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/health", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	err := adapter.Health(context.Background())
	assert.NoError(t, err)
}

func TestDefaultConfigs(t *testing.T) {
	llamaCfg := DefaultLlamaIndexConfig()
	assert.Equal(t, "http://localhost:8012", llamaCfg.BaseURL)
	assert.Equal(t, 120*time.Second, llamaCfg.Timeout)

	langCfg := DefaultLangChainConfig()
	assert.Equal(t, "http://localhost:8011", langCfg.BaseURL)
	assert.Equal(t, 120*time.Second, langCfg.Timeout)
}

func TestLlamaIndexHTTPAdapter_ImplementsInterface(t *testing.T) {
	var _ LlamaIndexAdapter = (*LlamaIndexHTTPAdapter)(nil)
}

func TestLangChainHTTPAdapter_ImplementsInterface(t *testing.T) {
	var _ LangChainAdapter = (*LangChainHTTPAdapter)(nil)
}
