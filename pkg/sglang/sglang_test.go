package sglang

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
