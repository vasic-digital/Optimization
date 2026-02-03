package sglang

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

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "http://localhost:30000", cfg.Endpoint)
	assert.Equal(t, 120*time.Second, cfg.Timeout)
}

func TestHTTPClient_Generate(t *testing.T) {
	tests := []struct {
		name       string
		program    *Program
		serverResp completionResponse
		statusCode int
		want       string
		wantErr    bool
	}{
		{
			name: "successful generation",
			program: &Program{
				UserPrompt:  "Hello",
				Temperature: 0.5,
				MaxTokens:   100,
			},
			serverResp: completionResponse{
				Choices: []completionChoice{
					{Message: message{Role: "assistant", Content: "Hi there!"}},
				},
			},
			statusCode: http.StatusOK,
			want:       "Hi there!",
		},
		{
			name: "with system prompt",
			program: &Program{
				SystemPrompt: "You are helpful.",
				UserPrompt:   "Hello",
			},
			serverResp: completionResponse{
				Choices: []completionChoice{
					{Message: message{
						Role:    "assistant",
						Content: "How can I help?",
					}},
				},
			},
			statusCode: http.StatusOK,
			want:       "How can I help?",
		},
		{
			name:       "empty choices returns empty string",
			program:    &Program{UserPrompt: "Hello"},
			serverResp: completionResponse{Choices: []completionChoice{}},
			statusCode: http.StatusOK,
			want:       "",
		},
		{
			name:       "server error",
			program:    &Program{UserPrompt: "Hello"},
			serverResp: completionResponse{},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:    "nil program returns error",
			program: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.program == nil {
				client := NewHTTPClient(nil)
				_, err := client.Generate(context.Background(), nil)
				assert.Error(t, err)
				return
			}

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					if tt.statusCode == http.StatusOK {
						_ = json.NewEncoder(w).Encode(tt.serverResp)
					} else {
						_, _ = w.Write([]byte("error"))
					}
				}),
			)
			defer server.Close()

			client := NewHTTPClient(&Config{
				Endpoint: server.URL,
				Timeout:  5 * time.Second,
			})

			result, err := client.Generate(context.Background(), tt.program)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestHTTPClient_Health(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "healthy service",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unhealthy service",
			statusCode: http.StatusServiceUnavailable,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
				}),
			)
			defer server.Close()

			client := NewHTTPClient(&Config{
				Endpoint: server.URL,
				Timeout:  5 * time.Second,
			})

			err := client.Health(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPClient_ImplementsClient(t *testing.T) {
	var _ Client = (*HTTPClient)(nil)
}

func TestProgram_Defaults(t *testing.T) {
	// Verify that defaults are applied during generation.
	var receivedReq completionRequest

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedReq)
			resp := completionResponse{
				Choices: []completionChoice{
					{Message: message{Content: "ok"}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer server.Close()

	client := NewHTTPClient(&Config{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	})

	_, err := client.Generate(context.Background(), &Program{
		UserPrompt: "test",
	})
	require.NoError(t, err)

	// Check defaults were applied.
	assert.Equal(t, 0.7, receivedReq.Temperature)
	assert.Equal(t, 500, receivedReq.MaxTokens)
}

// Tests for NewHTTPClient with nil config.

func TestNewHTTPClient_NilConfig(t *testing.T) {
	client := NewHTTPClient(nil)
	require.NotNil(t, client)
	assert.Equal(t, "http://localhost:30000", client.config.Endpoint)
	assert.Equal(t, 120*time.Second, client.config.Timeout)
}

// Tests for Generate with decode error.

func TestHTTPClient_Generate_DecodeError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json response"))
		}),
	)
	defer server.Close()

	client := NewHTTPClient(&Config{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	})

	_, err := client.Generate(context.Background(), &Program{
		UserPrompt: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

// Tests for Generate with all program options.

func TestHTTPClient_Generate_AllOptions(t *testing.T) {
	var receivedReq completionRequest

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/chat/completions", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			_ = json.NewDecoder(r.Body).Decode(&receivedReq)
			resp := completionResponse{
				Choices: []completionChoice{
					{Message: message{Content: "response"}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer server.Close()

	client := NewHTTPClient(&Config{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})

	result, err := client.Generate(context.Background(), &Program{
		SystemPrompt: "You are a helpful assistant",
		UserPrompt:   "Hello",
		Temperature:  0.5,
		MaxTokens:    100,
		TopP:         0.9,
		Stop:         []string{"END", "STOP"},
	})
	require.NoError(t, err)
	assert.Equal(t, "response", result)

	// Verify request was constructed correctly.
	assert.Equal(t, "test-model", receivedReq.Model)
	assert.Equal(t, 0.5, receivedReq.Temperature)
	assert.Equal(t, 100, receivedReq.MaxTokens)
	assert.Equal(t, 0.9, receivedReq.TopP)
	assert.Equal(t, []string{"END", "STOP"}, receivedReq.Stop)
	assert.Len(t, receivedReq.Messages, 2)
	assert.Equal(t, "system", receivedReq.Messages[0].Role)
	assert.Equal(t, "user", receivedReq.Messages[1].Role)
}

// Tests for doRequest with connection failure.

func TestHTTPClient_ConnectionFailure(t *testing.T) {
	client := NewHTTPClient(&Config{
		Endpoint: "http://localhost:59997", // Non-existent server.
		Timeout:  100 * time.Millisecond,
	})

	// Test Generate connection failure.
	_, err := client.Generate(context.Background(), &Program{
		UserPrompt: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")

	// Test Health connection failure.
	err = client.Health(context.Background())
	assert.Error(t, err)
}

// Tests for doRequest with invalid URL.

func TestHTTPClient_InvalidURL(t *testing.T) {
	client := NewHTTPClient(&Config{
		Endpoint: "://invalid-url",
		Timeout:  5 * time.Second,
	})

	// Test Generate with invalid URL.
	_, err := client.Generate(context.Background(), &Program{
		UserPrompt: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")

	// Test Health with invalid URL.
	err = client.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

// Tests for doRequest error body reading on HTTP error status.

func TestHTTPClient_ErrorBodyReading(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("detailed error message"))
		}),
	)
	defer server.Close()

	client := NewHTTPClient(&Config{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	})

	_, err := client.Generate(context.Background(), &Program{
		UserPrompt: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "detailed error message")
	assert.Contains(t, err.Error(), "400")
}

// Tests for context cancellation.

func TestHTTPClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	client := NewHTTPClient(&Config{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := client.Generate(ctx, &Program{
		UserPrompt: "test",
	})
	assert.Error(t, err)
}

// Tests for doRequest marshal error path using dependency injection.

func TestHTTPClient_Generate_MarshalError_DI(t *testing.T) {
	client := NewHTTPClient(&Config{
		Endpoint: "http://localhost:30000",
		Timeout:  5 * time.Second,
	})

	// Inject a failing marshaler
	client.marshalJSON = func(v interface{}) ([]byte, error) {
		return nil, fmt.Errorf("simulated marshal error")
	}

	_, err := client.Generate(context.Background(), &Program{
		UserPrompt: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal request")
	assert.Contains(t, err.Error(), "simulated marshal error")
}

// Test that nil marshaler defaults to json.Marshal.

func TestHTTPClient_NilMarshalerDefaultsToJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := completionResponse{
				Choices: []completionChoice{
					{Message: message{Content: "response"}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer server.Close()

	client := &HTTPClient{
		config: &Config{
			Endpoint: server.URL,
			Timeout:  5 * time.Second,
		},
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		marshalJSON: nil, // Explicitly nil
	}

	result, err := client.Generate(context.Background(), &Program{
		UserPrompt: "test",
	})
	require.NoError(t, err)
	assert.Equal(t, "response", result)
}
