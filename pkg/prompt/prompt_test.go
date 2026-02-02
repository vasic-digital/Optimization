package prompt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressor_Optimize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		config *Config
		want   string
	}{
		{
			name:   "empty prompt returns empty",
			input:  "",
			config: nil,
			want:   "",
		},
		{
			name:   "normalizes whitespace",
			input:  "  hello   world  ",
			config: nil,
			want:   "hello world",
		},
		{
			name:   "removes redundant phrases",
			input:  "Please note that Go is fast. Basically it compiles.",
			config: &Config{RemoveRedundancy: true, MaxTokens: 4096},
			want:   "Go is fast. it compiles.",
		},
		{
			name:   "preserves content when no redundancy",
			input:  "Go is a compiled language.",
			config: &Config{RemoveRedundancy: true, MaxTokens: 4096},
			want:   "Go is a compiled language.",
		},
		{
			name:   "truncates to max tokens",
			input:  "one two three four five six",
			config: &Config{MaxTokens: 3, RemoveRedundancy: false},
			want:   "one two three",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor := NewCompressor(tt.config)
			result, err := compressor.Optimize(context.Background(), tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCompressor_ImplementsOptimizer(t *testing.T) {
	var _ Optimizer = (*Compressor)(nil)
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "empty string", input: "", want: 0},
		{name: "single word", input: "hello", want: 1},
		{name: "multiple words", input: "hello world foo bar", want: 4},
		{name: "extra whitespace", input: "  hello   world  ", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EstimateTokens(tt.input))
		})
	}
}

func TestTemplate_Render(t *testing.T) {
	tests := []struct {
		name    string
		content string
		vars    map[string]string
		want    string
		wantErr bool
	}{
		{
			name:    "simple substitution",
			content: "Hello {{name}}, welcome to {{place}}!",
			vars:    map[string]string{"name": "Alice", "place": "Go"},
			want:    "Hello Alice, welcome to Go!",
		},
		{
			name:    "no variables needed",
			content: "Hello world!",
			vars:    map[string]string{},
			want:    "Hello world!",
		},
		{
			name:    "unresolved variable returns error",
			content: "Hello {{name}}, you are {{age}} years old.",
			vars:    map[string]string{"name": "Bob"},
			wantErr: true,
		},
		{
			name:    "repeated variable",
			content: "{{x}} and {{x}} again",
			vars:    map[string]string{"x": "value"},
			want:    "value and value again",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := &Template{
				Name:    "test",
				Content: tt.content,
			}
			result, err := tmpl.Render(tt.vars)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestTemplateRegistry_Register_And_Get(t *testing.T) {
	registry := NewTemplateRegistry()

	tmpl := &Template{
		Name:    "greeting",
		Content: "Hello {{name}}!",
	}
	require.NoError(t, registry.Register(tmpl))
	assert.Equal(t, 1, registry.Size())

	got, err := registry.Get("greeting")
	require.NoError(t, err)
	assert.Equal(t, "greeting", got.Name)

	_, err = registry.Get("nonexistent")
	assert.Error(t, err)
}

func TestTemplateRegistry_Register_NilTemplate(t *testing.T) {
	registry := NewTemplateRegistry()
	err := registry.Register(nil)
	assert.Error(t, err)
}

func TestTemplateRegistry_Register_EmptyName(t *testing.T) {
	registry := NewTemplateRegistry()
	err := registry.Register(&Template{Name: "", Content: "test"})
	assert.Error(t, err)
}

func TestTemplateRegistry_Remove(t *testing.T) {
	registry := NewTemplateRegistry()
	require.NoError(t, registry.Register(&Template{
		Name:    "tmpl",
		Content: "content",
	}))
	assert.Equal(t, 1, registry.Size())

	registry.Remove("tmpl")
	assert.Equal(t, 0, registry.Size())
}

func TestTemplateRegistry_List(t *testing.T) {
	registry := NewTemplateRegistry()
	require.NoError(t, registry.Register(&Template{
		Name: "a", Content: "aaa",
	}))
	require.NoError(t, registry.Register(&Template{
		Name: "b", Content: "bbb",
	}))

	names := registry.List()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "a")
	assert.Contains(t, names, "b")
}

func TestTemplateRegistry_RenderTemplate(t *testing.T) {
	registry := NewTemplateRegistry()
	require.NoError(t, registry.Register(&Template{
		Name:    "test",
		Content: "Hello {{name}}!",
	}))

	result, err := registry.RenderTemplate(
		"test",
		map[string]string{"name": "World"},
	)
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestTemplateRegistry_RenderTemplate_NotFound(t *testing.T) {
	registry := NewTemplateRegistry()
	_, err := registry.RenderTemplate("missing", nil)
	assert.Error(t, err)
}
