package outlines

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Constrainer defines the interface for constraining LLM output.
type Constrainer interface {
	// Constrain validates and optionally fixes output against a schema.
	// Returns the constrained output and any error.
	Constrain(output string, schema *Schema) (string, error)
}

// ValidationError represents a validation error.
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// ValidationResult contains the result of schema validation.
type ValidationResult struct {
	Valid  bool               `json:"valid"`
	Errors []*ValidationError `json:"errors,omitempty"`
	Data   interface{}        `json:"data,omitempty"`
}

// AddError adds a validation error.
func (r *ValidationResult) AddError(path, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, &ValidationError{
		Path:    path,
		Message: message,
	})
}

// ErrorMessages returns all error messages as strings.
func (r *ValidationResult) ErrorMessages() []string {
	messages := make([]string, len(r.Errors))
	for i, err := range r.Errors {
		messages[i] = err.Error()
	}
	return messages
}

// JSONConstrainer validates and fixes JSON output against a schema.
type JSONConstrainer struct{}

// NewJSONConstrainer creates a new JSON constrainer.
func NewJSONConstrainer() *JSONConstrainer {
	return &JSONConstrainer{}
}

// Constrain validates JSON output against the schema.
func (c *JSONConstrainer) Constrain(
	output string,
	schema *Schema,
) (string, error) {
	// Try to extract JSON from the output.
	jsonStr := extractJSON(output)
	if jsonStr == "" {
		return "", fmt.Errorf("no valid JSON found in output")
	}

	// Parse and validate.
	result := Validate(jsonStr, schema)
	if !result.Valid {
		return jsonStr, fmt.Errorf(
			"validation failed: %s",
			strings.Join(result.ErrorMessages(), "; "),
		)
	}

	return jsonStr, nil
}

// RegexConstrainer validates output against a regex pattern.
type RegexConstrainer struct {
	pattern *regexp.Regexp
}

// NewRegexConstrainer creates a new regex constrainer.
func NewRegexConstrainer(pattern string) (*RegexConstrainer, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}
	return &RegexConstrainer{pattern: re}, nil
}

// Constrain validates output against the regex pattern.
// The schema parameter is ignored for regex constraining.
func (c *RegexConstrainer) Constrain(
	output string,
	_ *Schema,
) (string, error) {
	output = strings.TrimSpace(output)
	if c.pattern.MatchString(output) {
		return output, nil
	}

	// Try to find a matching substring.
	match := c.pattern.FindString(output)
	if match != "" {
		return match, nil
	}

	return output, fmt.Errorf(
		"output does not match pattern %q", c.pattern.String(),
	)
}

// Validate validates a JSON string against a schema.
func Validate(jsonStr string, schema *Schema) *ValidationResult {
	result := &ValidationResult{Valid: true}

	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		result.AddError("", fmt.Sprintf("invalid JSON: %v", err))
		return result
	}

	validateValue(data, schema, "", result)
	result.Data = data
	return result
}

func validateValue(
	data interface{},
	schema *Schema,
	path string,
	result *ValidationResult,
) {
	if schema == nil {
		return
	}

	if len(schema.Enum) > 0 {
		validateEnum(data, schema.Enum, path, result)
		return
	}

	switch schema.Type {
	case "object":
		validateObject(data, schema, path, result)
	case "array":
		validateArray(data, schema, path, result)
	case "string":
		validateString(data, schema, path, result)
	case "number":
		validateNumber(data, schema, path, result)
	case "integer":
		validateInteger(data, schema, path, result)
	case "boolean":
		validateBoolean(data, path, result)
	case "":
		// No type specified, any type is valid.
	}
}

func validateObject(
	data interface{},
	schema *Schema,
	path string,
	result *ValidationResult,
) {
	obj, ok := data.(map[string]interface{})
	if !ok {
		result.AddError(path, "expected object")
		return
	}

	for _, req := range schema.Required {
		if _, exists := obj[req]; !exists {
			result.AddError(
				joinPath(path, req),
				"required property missing",
			)
		}
	}

	for propName, propSchema := range schema.Properties {
		if propValue, exists := obj[propName]; exists {
			validateValue(
				propValue,
				propSchema,
				joinPath(path, propName),
				result,
			)
		}
	}

	if schema.AdditionalProperties != nil && !*schema.AdditionalProperties {
		for propName := range obj {
			if _, defined := schema.Properties[propName]; !defined {
				result.AddError(
					joinPath(path, propName),
					"additional property not allowed",
				)
			}
		}
	}
}

func validateArray(
	data interface{},
	schema *Schema,
	path string,
	result *ValidationResult,
) {
	arr, ok := data.([]interface{})
	if !ok {
		result.AddError(path, "expected array")
		return
	}

	if schema.MinItems != nil && len(arr) < *schema.MinItems {
		result.AddError(path, fmt.Sprintf(
			"array must have at least %d items", *schema.MinItems,
		))
	}
	if schema.MaxItems != nil && len(arr) > *schema.MaxItems {
		result.AddError(path, fmt.Sprintf(
			"array must have at most %d items", *schema.MaxItems,
		))
	}

	if schema.Items != nil {
		for i, item := range arr {
			validateValue(
				item,
				schema.Items,
				fmt.Sprintf("%s[%d]", path, i),
				result,
			)
		}
	}
}

func validateString(
	data interface{},
	schema *Schema,
	path string,
	result *ValidationResult,
) {
	str, ok := data.(string)
	if !ok {
		result.AddError(path, "expected string")
		return
	}

	if schema.MinLength != nil && len(str) < *schema.MinLength {
		result.AddError(path, fmt.Sprintf(
			"string must be at least %d characters", *schema.MinLength,
		))
	}
	if schema.MaxLength != nil && len(str) > *schema.MaxLength {
		result.AddError(path, fmt.Sprintf(
			"string must be at most %d characters", *schema.MaxLength,
		))
	}

	if schema.Pattern != "" {
		re, err := regexp.Compile(schema.Pattern)
		if err == nil && !re.MatchString(str) {
			result.AddError(path, fmt.Sprintf(
				"string must match pattern %q", schema.Pattern,
			))
		}
	}
}

func validateNumber(
	data interface{},
	schema *Schema,
	path string,
	result *ValidationResult,
) {
	var num float64
	switch n := data.(type) {
	case float64:
		num = n
	case int:
		num = float64(n)
	default:
		result.AddError(path, "expected number")
		return
	}

	validateNumericConstraints(num, schema, path, result)
}

func validateInteger(
	data interface{},
	schema *Schema,
	path string,
	result *ValidationResult,
) {
	var num float64
	switch n := data.(type) {
	case float64:
		if n != float64(int64(n)) {
			result.AddError(path, "expected integer")
			return
		}
		num = n
	case int:
		num = float64(n)
	default:
		result.AddError(path, "expected integer")
		return
	}

	validateNumericConstraints(num, schema, path, result)
}

func validateNumericConstraints(
	num float64,
	schema *Schema,
	path string,
	result *ValidationResult,
) {
	if schema.Minimum != nil && num < *schema.Minimum {
		result.AddError(path, fmt.Sprintf(
			"value must be >= %v", *schema.Minimum,
		))
	}
	if schema.Maximum != nil && num > *schema.Maximum {
		result.AddError(path, fmt.Sprintf(
			"value must be <= %v", *schema.Maximum,
		))
	}
}

func validateBoolean(
	data interface{},
	path string,
	result *ValidationResult,
) {
	if _, ok := data.(bool); !ok {
		result.AddError(path, "expected boolean")
	}
}

func validateEnum(
	data interface{},
	enum []interface{},
	path string,
	result *ValidationResult,
) {
	for _, allowed := range enum {
		if reflect.DeepEqual(data, allowed) {
			return
		}
	}
	result.AddError(path, fmt.Sprintf(
		"value must be one of %v", enum,
	))
}

func joinPath(base, property string) string {
	if base == "" {
		return property
	}
	return base + "." + property
}

// extractJSON extracts JSON content from a response that may contain text.
func extractJSON(response string) string {
	response = strings.TrimSpace(response)

	// Try to find JSON object.
	if start := strings.Index(response, "{"); start != -1 {
		if end := findMatchingBrace(
			response[start:], '{', '}',
		); end != -1 {
			return response[start : start+end+1]
		}
	}

	// Try to find JSON array.
	if start := strings.Index(response, "["); start != -1 {
		if end := findMatchingBrace(
			response[start:], '[', ']',
		); end != -1 {
			return response[start : start+end+1]
		}
	}

	// Check if entire response is valid JSON.
	var js interface{}
	if err := json.Unmarshal([]byte(response), &js); err == nil {
		return response
	}

	return ""
}

func findMatchingBrace(s string, open, close byte) int {
	if len(s) == 0 || s[0] != open {
		return -1
	}

	count := 0
	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == open {
			count++
		} else if c == close {
			count--
			if count == 0 {
				return i
			}
		}
	}

	return -1
}
