package prompt

import (
	"fmt"
	"strings"
	"sync"
)

// Template represents a prompt template with variable substitution.
type Template struct {
	// Name is the template name.
	Name string `json:"name"`
	// Content is the template content with {{variable}} placeholders.
	Content string `json:"content"`
	// Description describes the template's purpose.
	Description string `json:"description,omitempty"`
	// Variables lists the expected variable names.
	Variables []string `json:"variables,omitempty"`
}

// Render renders the template with the given variables.
// Variables are substituted using {{key}} placeholders.
func (t *Template) Render(vars map[string]string) (string, error) {
	result := t.Content
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Check for unresolved placeholders.
	if idx := strings.Index(result, "{{"); idx != -1 {
		end := strings.Index(result[idx:], "}}")
		if end != -1 {
			varName := result[idx+2 : idx+end]
			return "", fmt.Errorf(
				"unresolved variable: %s", varName,
			)
		}
	}

	return result, nil
}

// TemplateRegistry manages prompt templates.
type TemplateRegistry struct {
	mu        sync.RWMutex
	templates map[string]*Template
}

// NewTemplateRegistry creates a new template registry.
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string]*Template),
	}
}

// Register registers a template.
func (r *TemplateRegistry) Register(template *Template) error {
	if template == nil {
		return fmt.Errorf("template must not be nil")
	}
	if template.Name == "" {
		return fmt.Errorf("template name must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.templates[template.Name] = template
	return nil
}

// Get retrieves a template by name.
func (r *TemplateRegistry) Get(name string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, ok := r.templates[name]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return tmpl, nil
}

// Remove removes a template by name.
func (r *TemplateRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.templates, name)
}

// List returns all registered template names.
func (r *TemplateRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// Size returns the number of registered templates.
func (r *TemplateRegistry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.templates)
}

// RenderTemplate retrieves a template by name and renders it with variables.
func (r *TemplateRegistry) RenderTemplate(
	name string,
	vars map[string]string,
) (string, error) {
	tmpl, err := r.Get(name)
	if err != nil {
		return "", err
	}
	return tmpl.Render(vars)
}
