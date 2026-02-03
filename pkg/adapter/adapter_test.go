package adapter

import (
	"context"
	"encoding/json"
	"fmt"
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

// Tests for nil config handling in adapter constructors.

func TestNewLlamaIndexHTTPAdapter_NilConfig(t *testing.T) {
	adapter := NewLlamaIndexHTTPAdapter(nil)
	require.NotNil(t, adapter)
	assert.Equal(t, "http://localhost:8012", adapter.config.BaseURL)
	assert.Equal(t, 120*time.Second, adapter.config.Timeout)
}

func TestNewLangChainHTTPAdapter_NilConfig(t *testing.T) {
	adapter := NewLangChainHTTPAdapter(nil)
	require.NotNil(t, adapter)
	assert.Equal(t, "http://localhost:8011", adapter.config.BaseURL)
	assert.Equal(t, 120*time.Second, adapter.config.Timeout)
}

// Tests for HTTP request/decode errors.

func TestLlamaIndexHTTPAdapter_Query_DecodeError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}),
	)
	defer server.Close()

	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	result, err := adapter.Query(context.Background(), "test", 5)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestLlamaIndexHTTPAdapter_Rerank_DecodeError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not valid json"))
		}),
	)
	defer server.Close()

	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	docs, err := adapter.Rerank(context.Background(), "query", []string{"doc1"}, 2)
	assert.Nil(t, docs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestLlamaIndexHTTPAdapter_Rerank_DefaultTopK(t *testing.T) {
	var receivedTopK int
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				TopK int `json:"top_k"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			receivedTopK = req.TopK
			resp := struct {
				RankedDocuments []RankedDocument `json:"ranked_documents"`
			}{
				RankedDocuments: []RankedDocument{{Content: "doc", Score: 0.9, Rank: 1}},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer server.Close()

	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	// Test with topK = 0, should default to 5.
	_, err := adapter.Rerank(context.Background(), "query", []string{"doc"}, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, receivedTopK)

	// Test with topK = -1, should also default to 5.
	_, err = adapter.Rerank(context.Background(), "query", []string{"doc"}, -1)
	require.NoError(t, err)
	assert.Equal(t, 5, receivedTopK)
}

func TestLangChainHTTPAdapter_ExecuteChain_DecodeError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{malformed json"))
		}),
	)
	defer server.Close()

	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	result, err := adapter.ExecuteChain(context.Background(), "chain", "prompt", nil)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestLangChainHTTPAdapter_Decompose_DecodeError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<<<not json>>>"))
		}),
	)
	defer server.Close()

	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	result, err := adapter.Decompose(context.Background(), "task", 3)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestLangChainHTTPAdapter_Decompose_DefaultMaxSteps(t *testing.T) {
	tests := []struct {
		name            string
		maxSteps        int
		expectedMaxStep int
	}{
		{
			name:            "zero maxSteps defaults to 5",
			maxSteps:        0,
			expectedMaxStep: 5,
		},
		{
			name:            "negative maxSteps defaults to 5",
			maxSteps:        -10,
			expectedMaxStep: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedMaxSteps int
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req struct {
						MaxSteps int `json:"max_steps"`
					}
					_ = json.NewDecoder(r.Body).Decode(&req)
					receivedMaxSteps = req.MaxSteps
					resp := DecomposeResult{
						Subtasks:  []Subtask{{ID: 1, Description: "step"}},
						Reasoning: "done",
					}
					_ = json.NewEncoder(w).Encode(resp)
				}),
			)
			defer server.Close()

			adapter := NewLangChainHTTPAdapter(&LangChainConfig{
				BaseURL: server.URL,
				Timeout: 5 * time.Second,
			})

			_, err := adapter.Decompose(context.Background(), "task", tt.maxSteps)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMaxStep, receivedMaxSteps)
		})
	}
}

// Tests for health check failures.

func TestLlamaIndexHTTPAdapter_Health_Failure(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("service down"))
		}),
	)
	defer server.Close()

	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	err := adapter.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

func TestLangChainHTTPAdapter_Health_Failure(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		}),
	)
	defer server.Close()

	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	err := adapter.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

// Tests for connection failures.

func TestLlamaIndexHTTPAdapter_ConnectionFailure(t *testing.T) {
	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: "http://localhost:59999", // Non-existent server.
		Timeout: 100 * time.Millisecond,
	})

	// Test Query connection failure.
	result, err := adapter.Query(context.Background(), "test", 5)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")

	// Test Rerank connection failure.
	docs, err := adapter.Rerank(context.Background(), "test", []string{"doc"}, 2)
	assert.Nil(t, docs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")

	// Test Health connection failure.
	err = adapter.Health(context.Background())
	assert.Error(t, err)
}

func TestLangChainHTTPAdapter_ConnectionFailure(t *testing.T) {
	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: "http://localhost:59998", // Non-existent server.
		Timeout: 100 * time.Millisecond,
	})

	// Test ExecuteChain connection failure.
	result, err := adapter.ExecuteChain(context.Background(), "chain", "prompt", nil)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")

	// Test Decompose connection failure.
	decompResult, err := adapter.Decompose(context.Background(), "task", 5)
	assert.Nil(t, decompResult)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")

	// Test Health connection failure.
	err = adapter.Health(context.Background())
	assert.Error(t, err)
}

// Tests for context cancellation.

func TestLlamaIndexHTTPAdapter_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := adapter.Query(ctx, "test", 5)
	assert.Error(t, err)
}

func TestLangChainHTTPAdapter_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := adapter.ExecuteChain(ctx, "chain", "prompt", nil)
	assert.Error(t, err)
}

// Tests for invalid URL in config (triggers request creation error).

func TestLlamaIndexHTTPAdapter_InvalidURL(t *testing.T) {
	// Using an invalid URL that will cause http.NewRequestWithContext to fail.
	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: "://invalid-url",
		Timeout: 5 * time.Second,
	})

	// Query with invalid URL.
	result, err := adapter.Query(context.Background(), "test", 5)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")

	// Rerank with invalid URL.
	docs, err := adapter.Rerank(context.Background(), "test", []string{"doc"}, 2)
	assert.Nil(t, docs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")

	// Health with invalid URL.
	err = adapter.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestLangChainHTTPAdapter_InvalidURL(t *testing.T) {
	// Using an invalid URL that will cause http.NewRequestWithContext to fail.
	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: "://bad-url",
		Timeout: 5 * time.Second,
	})

	// ExecuteChain with invalid URL.
	result, err := adapter.ExecuteChain(context.Background(), "chain", "prompt", nil)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")

	// Decompose with invalid URL.
	decompResult, err := adapter.Decompose(context.Background(), "task", 5)
	assert.Nil(t, decompResult)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")

	// Health with invalid URL.
	err = adapter.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

// Tests for doRequest marshal error (json.Marshal fails on unmarshalable types).

type unmarshalableType struct {
	Ch chan int
}

func TestLlamaIndexHTTPAdapter_DoRequest_MarshalError(t *testing.T) {
	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: "http://localhost:8012",
		Timeout: 5 * time.Second,
	})

	// Accessing doRequest indirectly through Query with a type that causes marshal error
	// is not possible since Query uses typed requests. We need to verify the marshal
	// error path is covered through the doRequest function.
	// Since doRequest is private, we test via a custom type test if possible.
	// However, all public methods use typed request structs that always marshal.
	// The marshal error line is defensive and may not be easily testable with
	// standard usage.

	// For completeness, verify that the adapter handles server errors correctly
	// which exercises the error body reading path.
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request error body"))
		}),
	)
	defer server.Close()

	adapter = NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := adapter.Query(context.Background(), "test", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad request error body")
}

func TestLangChainHTTPAdapter_DoRequest_MarshalError(t *testing.T) {
	// Similar to above, verify error body reading in doRequest for LangChain.
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("chain error details"))
		}),
	)
	defer server.Close()

	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := adapter.ExecuteChain(context.Background(), "chain", "prompt", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chain error details")
}

// Tests for doRequest marshal error paths using dependency injection.

func TestLlamaIndexHTTPAdapter_DoRequest_MarshalError_DI(t *testing.T) {
	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: "http://localhost:8012",
		Timeout: 5 * time.Second,
	})

	// Inject a failing marshaler
	adapter.marshalJSON = func(v interface{}) ([]byte, error) {
		return nil, fmt.Errorf("simulated marshal error")
	}

	_, err := adapter.Query(context.Background(), "test", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal request")
	assert.Contains(t, err.Error(), "simulated marshal error")
}

func TestLangChainHTTPAdapter_DoRequest_MarshalError_DI(t *testing.T) {
	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: "http://localhost:8011",
		Timeout: 5 * time.Second,
	})

	// Inject a failing marshaler
	adapter.marshalJSON = func(v interface{}) ([]byte, error) {
		return nil, fmt.Errorf("simulated marshal error")
	}

	_, err := adapter.ExecuteChain(context.Background(), "chain", "prompt", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal request")
	assert.Contains(t, err.Error(), "simulated marshal error")
}

func TestLlamaIndexHTTPAdapter_Rerank_MarshalError_DI(t *testing.T) {
	adapter := NewLlamaIndexHTTPAdapter(&LlamaIndexConfig{
		BaseURL: "http://localhost:8012",
		Timeout: 5 * time.Second,
	})

	// Inject a failing marshaler
	adapter.marshalJSON = func(v interface{}) ([]byte, error) {
		return nil, fmt.Errorf("rerank marshal error")
	}

	_, err := adapter.Rerank(context.Background(), "query", []string{"doc"}, 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal request")
}

func TestLangChainHTTPAdapter_Decompose_MarshalError_DI(t *testing.T) {
	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: "http://localhost:8011",
		Timeout: 5 * time.Second,
	})

	// Inject a failing marshaler
	adapter.marshalJSON = func(v interface{}) ([]byte, error) {
		return nil, fmt.Errorf("decompose marshal error")
	}

	_, err := adapter.Decompose(context.Background(), "task", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal request")
}

// Test that nil marshaler defaults to json.Marshal.

func TestLlamaIndexHTTPAdapter_NilMarshalerDefaultsToJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := QueryResult{Answer: "ok"}
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer server.Close()

	adapter := &LlamaIndexHTTPAdapter{
		config: &LlamaIndexConfig{
			BaseURL: server.URL,
			Timeout: 5 * time.Second,
		},
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		marshalJSON: nil, // Explicitly nil
	}

	result, err := adapter.Query(context.Background(), "test", 5)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Answer)
}

func TestLangChainHTTPAdapter_NilMarshalerDefaultsToJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChainResult{Result: "ok"}
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer server.Close()

	adapter := &LangChainHTTPAdapter{
		config: &LangChainConfig{
			BaseURL: server.URL,
			Timeout: 5 * time.Second,
		},
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		marshalJSON: nil, // Explicitly nil
	}

	result, err := adapter.ExecuteChain(context.Background(), "chain", "prompt", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Result)
}

// Test for ExecuteChain with variables map.

func TestLangChainHTTPAdapter_ExecuteChain_WithVariables(t *testing.T) {
	var receivedVars map[string]interface{}

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Variables map[string]interface{} `json:"variables"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			receivedVars = req.Variables
			resp := ChainResult{Result: "ok"}
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer server.Close()

	adapter := NewLangChainHTTPAdapter(&LangChainConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	vars := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	_, err := adapter.ExecuteChain(context.Background(), "chain", "prompt", vars)
	require.NoError(t, err)

	assert.Equal(t, "value1", receivedVars["key1"])
	assert.Equal(t, float64(42), receivedVars["key2"]) // JSON numbers are float64.
	assert.Equal(t, true, receivedVars["key3"])
}
