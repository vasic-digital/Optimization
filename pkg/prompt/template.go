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
//
// Security: unresolved-variable detection scans the ORIGINAL template
// content (not the post-substitution result), so user-supplied variable
// values containing {{...}} sequences are treated as literal text and
// cannot be mistaken for additional template placeholders. This prevents
// CONST-035 injection bluffs (security test TestTemplate_InjectionResistance).
func (t *Template) Render(vars map[string]string) (string, error) {
	// Step 1: detect unresolved placeholders in the ORIGINAL content
	// before any substitution. A placeholder is "unresolved" iff its
	// variable name has no entry in the supplied vars map.
	content := t.Content
	cursor := 0
	for {
		idx := strings.Index(content[cursor:], "{{")
		if idx == -1 {
			break
		}
		absIdx := cursor + idx
		end := strings.Index(content[absIdx:], "}}")
		if end == -1 {
			break // dangling "{{" with no closing — leave it as literal
		}
		varName := content[absIdx+2 : absIdx+end]
		if _, ok := vars[varName]; !ok {
			return "", fmt.Errorf(
				"unresolved variable: %s", varName,
			)
		}
		cursor = absIdx + end + 2
	}

	// Step 2: substitute every supplied variable. Values containing
	// {{...}} sequences are inert because step 1 already validated
	// the original content; the post-substitution string is never
	// re-scanned for placeholders.
	result := content
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
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
