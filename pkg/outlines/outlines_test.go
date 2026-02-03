package outlines

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSchema(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid object schema",
			input:   `{"type":"object","properties":{"name":{"type":"string"}}}`,
			wantErr: false,
		},
		{
			name:    "valid string schema",
			input:   `{"type":"string"}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := ParseSchema([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, schema)
			}
		})
	}
}

func TestSchema_IsRequired(t *testing.T) {
	schema := &Schema{
		Type:     "object",
		Required: []string{"name", "age"},
	}

	assert.True(t, schema.IsRequired("name"))
	assert.True(t, schema.IsRequired("age"))
	assert.False(t, schema.IsRequired("email"))
}

func TestSchemaBuilder(t *testing.T) {
	schema := NewSchemaBuilder().
		Object().
		Property("name", StringSchema()).
		Property("age", IntegerSchema()).
		RequiredProps("name").
		SetDescription("A person").
		Build()

	assert.Equal(t, "object", schema.Type)
	assert.Len(t, schema.Properties, 2)
	assert.Equal(t, "string", schema.Properties["name"].Type)
	assert.Equal(t, "integer", schema.Properties["age"].Type)
	assert.Contains(t, schema.Required, "name")
	assert.Equal(t, "A person", schema.Description)
}

func TestSchemaBuilder_Array(t *testing.T) {
	schema := NewSchemaBuilder().
		Array().
		Items(StringSchema()).
		Build()

	assert.Equal(t, "array", schema.Type)
	assert.NotNil(t, schema.Items)
	assert.Equal(t, "string", schema.Items.Type)
}

func TestValidate_ValidObject(t *testing.T) {
	schema := ObjectSchema(map[string]*Schema{
		"name": StringSchema(),
		"age":  IntegerSchema(),
	}, "name")

	result := Validate(`{"name":"Alice","age":30}`, schema)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidate_MissingRequired(t *testing.T) {
	schema := ObjectSchema(map[string]*Schema{
		"name": StringSchema(),
	}, "name")

	result := Validate(`{}`, schema)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestValidate_InvalidJSON(t *testing.T) {
	schema := StringSchema()
	result := Validate(`{invalid`, schema)
	assert.False(t, result.Valid)
}

func TestValidate_TypeMismatch(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		input  string
	}{
		{
			name:   "string expected got number",
			schema: StringSchema(),
			input:  `42`,
		},
		{
			name:   "number expected got string",
			schema: NumberSchema(),
			input:  `"hello"`,
		},
		{
			name:   "boolean expected got string",
			schema: BooleanSchema(),
			input:  `"true"`,
		},
		{
			name:   "array expected got string",
			schema: ArraySchema(StringSchema()),
			input:  `"hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.input, tt.schema)
			assert.False(t, result.Valid)
		})
	}
}

func TestValidate_StringConstraints(t *testing.T) {
	min := 3
	max := 10
	schema := &Schema{
		Type:      "string",
		MinLength: &min,
		MaxLength: &max,
	}

	result := Validate(`"hello"`, schema)
	assert.True(t, result.Valid)

	result = Validate(`"hi"`, schema)
	assert.False(t, result.Valid)

	result = Validate(`"this is too long"`, schema)
	assert.False(t, result.Valid)
}

func TestValidate_NumericConstraints(t *testing.T) {
	min := 0.0
	max := 100.0
	schema := &Schema{
		Type:    "number",
		Minimum: &min,
		Maximum: &max,
	}

	result := Validate(`50`, schema)
	assert.True(t, result.Valid)

	result = Validate(`-1`, schema)
	assert.False(t, result.Valid)

	result = Validate(`101`, schema)
	assert.False(t, result.Valid)
}

func TestValidate_ArrayConstraints(t *testing.T) {
	min := 2
	max := 4
	schema := &Schema{
		Type:     "array",
		Items:    StringSchema(),
		MinItems: &min,
		MaxItems: &max,
	}

	result := Validate(`["a","b","c"]`, schema)
	assert.True(t, result.Valid)

	result = Validate(`["a"]`, schema)
	assert.False(t, result.Valid)

	result = Validate(`["a","b","c","d","e"]`, schema)
	assert.False(t, result.Valid)
}

func TestValidate_Enum(t *testing.T) {
	schema := &Schema{
		Enum: []interface{}{"red", "green", "blue"},
	}

	result := Validate(`"red"`, schema)
	assert.True(t, result.Valid)

	result = Validate(`"yellow"`, schema)
	assert.False(t, result.Valid)
}

func TestValidate_Pattern(t *testing.T) {
	schema := &Schema{
		Type:    "string",
		Pattern: `^\d{3}-\d{4}$`,
	}

	result := Validate(`"123-4567"`, schema)
	assert.True(t, result.Valid)

	result = Validate(`"abc"`, schema)
	assert.False(t, result.Valid)
}

func TestJSONConstrainer_Constrain(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		schema  *Schema
		want    string
		wantErr bool
	}{
		{
			name:   "valid JSON passes through",
			output: `{"name":"Alice"}`,
			schema: ObjectSchema(map[string]*Schema{
				"name": StringSchema(),
			}),
			want: `{"name":"Alice"}`,
		},
		{
			name:   "extracts JSON from text",
			output: `Here is the result: {"name":"Bob"} hope this helps`,
			schema: ObjectSchema(map[string]*Schema{
				"name": StringSchema(),
			}),
			want: `{"name":"Bob"}`,
		},
		{
			name:    "no JSON found",
			output:  "just plain text",
			schema:  StringSchema(),
			wantErr: true,
		},
		{
			name:   "invalid against schema",
			output: `{"age":30}`,
			schema: ObjectSchema(map[string]*Schema{
				"name": StringSchema(),
			}, "name"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constrainer := NewJSONConstrainer()
			result, err := constrainer.Constrain(tt.output, tt.schema)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestRegexConstrainer_Constrain(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:    "full match",
			pattern: `^\d{3}-\d{4}$`,
			output:  "123-4567",
			want:    "123-4567",
		},
		{
			name:    "partial match extracts pattern",
			pattern: `\d{3}-\d{4}`,
			output:  "call 123-4567 now",
			want:    "123-4567", // Now extracts just the match
		},
		{
			name:    "no match",
			pattern: `^\d{3}-\d{4}$`,
			output:  "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constrainer, err := NewRegexConstrainer(tt.pattern)
			require.NoError(t, err)

			result, err := constrainer.Constrain(tt.output, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestRegexConstrainer_InvalidPattern(t *testing.T) {
	_, err := NewRegexConstrainer("[invalid")
	assert.Error(t, err)
}

func TestJSONConstrainer_ImplementsConstrainer(t *testing.T) {
	var _ Constrainer = (*JSONConstrainer)(nil)
}

func TestRegexConstrainer_ImplementsConstrainer(t *testing.T) {
	var _ Constrainer = (*RegexConstrainer)(nil)
}

func TestHelperSchemas(t *testing.T) {
	assert.Equal(t, "string", StringSchema().Type)
	assert.Equal(t, "integer", IntegerSchema().Type)
	assert.Equal(t, "number", NumberSchema().Type)
	assert.Equal(t, "boolean", BooleanSchema().Type)

	arr := ArraySchema(StringSchema())
	assert.Equal(t, "array", arr.Type)
	assert.Equal(t, "string", arr.Items.Type)

	obj := ObjectSchema(map[string]*Schema{
		"x": StringSchema(),
	}, "x")
	assert.Equal(t, "object", obj.Type)
	assert.Contains(t, obj.Required, "x")
}

func TestSchema_String(t *testing.T) {
	tests := []struct {
		name     string
		schema   *Schema
		contains []string
	}{
		{
			name:     "simple string schema",
			schema:   StringSchema(),
			contains: []string{`"type": "string"`},
		},
		{
			name: "object with properties",
			schema: ObjectSchema(map[string]*Schema{
				"name": StringSchema(),
			}, "name"),
			contains: []string{
				`"type": "object"`,
				`"properties"`,
				`"name"`,
				`"required"`,
			},
		},
		{
			name: "schema with pattern",
			schema: &Schema{
				Type:    "string",
				Pattern: `^\d+$`,
			},
			contains: []string{`"type": "string"`, `"pattern"`},
		},
		{
			name: "schema with constraints",
			schema: func() *Schema {
				min := 1
				max := 10
				return &Schema{
					Type:      "string",
					MinLength: &min,
					MaxLength: &max,
				}
			}(),
			contains: []string{`"minLength": 1`, `"maxLength": 10`},
		},
		{
			name: "empty schema",
			schema: &Schema{},
			contains: []string{`{`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.schema.String()
			assert.NotEmpty(t, result)
			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestSchemaBuilder_TypeMethods(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *SchemaBuilder
		expected string
	}{
		{
			name: "StringType sets type to string",
			builder: func() *SchemaBuilder {
				return NewSchemaBuilder().StringType()
			},
			expected: "string",
		},
		{
			name: "NumberType sets type to number",
			builder: func() *SchemaBuilder {
				return NewSchemaBuilder().NumberType()
			},
			expected: "number",
		},
		{
			name: "IntegerType sets type to integer",
			builder: func() *SchemaBuilder {
				return NewSchemaBuilder().IntegerType()
			},
			expected: "integer",
		},
		{
			name: "BooleanType sets type to boolean",
			builder: func() *SchemaBuilder {
				return NewSchemaBuilder().BooleanType()
			},
			expected: "boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := tt.builder().Build()
			assert.Equal(t, tt.expected, schema.Type)
		})
	}
}

func TestSchemaBuilder_EnumValues(t *testing.T) {
	tests := []struct {
		name   string
		values []interface{}
	}{
		{
			name:   "string enum values",
			values: []interface{}{"red", "green", "blue"},
		},
		{
			name:   "integer enum values",
			values: []interface{}{1, 2, 3},
		},
		{
			name:   "mixed enum values",
			values: []interface{}{"active", 1, true},
		},
		{
			name:   "single enum value",
			values: []interface{}{"only"},
		},
		{
			name:   "empty enum values",
			values: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewSchemaBuilder().
				StringType().
				EnumValues(tt.values...).
				Build()
			assert.Equal(t, tt.values, schema.Enum)
		})
	}
}

func TestSchemaBuilder_SetPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{
			name:    "digit pattern",
			pattern: `^\d+$`,
		},
		{
			name:    "email-like pattern",
			pattern: `^[a-z]+@[a-z]+\.[a-z]+$`,
		},
		{
			name:    "phone pattern",
			pattern: `^\d{3}-\d{3}-\d{4}$`,
		},
		{
			name:    "empty pattern",
			pattern: "",
		},
		{
			name:    "uuid pattern",
			pattern: `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewSchemaBuilder().
				StringType().
				SetPattern(tt.pattern).
				Build()
			assert.Equal(t, tt.pattern, schema.Pattern)
		})
	}
}

func TestSchemaBuilder_Chaining(t *testing.T) {
	schema := NewSchemaBuilder().
		StringType().
		EnumValues("a", "b", "c").
		SetPattern(`^[a-c]$`).
		SetDescription("Letter choice").
		Build()

	assert.Equal(t, "string", schema.Type)
	assert.Equal(t, []interface{}{"a", "b", "c"}, schema.Enum)
	assert.Equal(t, `^[a-c]$`, schema.Pattern)
	assert.Equal(t, "Letter choice", schema.Description)
}

func TestValidateObject_AdditionalProperties(t *testing.T) {
	tests := []struct {
		name                 string
		additionalProperties *bool
		input                string
		wantValid            bool
		wantErrorContains    string
	}{
		{
			name:                 "additional properties allowed (nil)",
			additionalProperties: nil,
			input:                `{"name":"Alice","extra":"field"}`,
			wantValid:            true,
		},
		{
			name: "additional properties allowed (true)",
			additionalProperties: func() *bool {
				b := true
				return &b
			}(),
			input:     `{"name":"Alice","extra":"field"}`,
			wantValid: true,
		},
		{
			name: "additional properties forbidden",
			additionalProperties: func() *bool {
				b := false
				return &b
			}(),
			input:             `{"name":"Alice","extra":"field"}`,
			wantValid:         false,
			wantErrorContains: "additional property not allowed",
		},
		{
			name: "no additional properties present when forbidden",
			additionalProperties: func() *bool {
				b := false
				return &b
			}(),
			input:     `{"name":"Alice"}`,
			wantValid: true,
		},
		{
			name: "multiple additional properties forbidden",
			additionalProperties: func() *bool {
				b := false
				return &b
			}(),
			input:             `{"name":"Alice","extra1":"a","extra2":"b"}`,
			wantValid:         false,
			wantErrorContains: "additional property not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"name": StringSchema(),
				},
				AdditionalProperties: tt.additionalProperties,
			}

			result := Validate(tt.input, schema)
			assert.Equal(t, tt.wantValid, result.Valid)
			if !tt.wantValid && tt.wantErrorContains != "" {
				messages := result.ErrorMessages()
				found := false
				for _, msg := range messages {
					if assert.ObjectsAreEqual(true, contains(msg, tt.wantErrorContains)) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected error containing %q, got %v",
					tt.wantErrorContains, messages)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestExtractJSON_Array(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		schema   *Schema
	}{
		{
			name:     "simple array",
			input:    `[1, 2, 3]`,
			expected: `[1, 2, 3]`,
			schema:   &Schema{Type: "array", Items: NumberSchema()},
		},
		{
			name:     "array with text before",
			input:    `Here is the result: [1, 2, 3]`,
			expected: `[1, 2, 3]`,
			schema:   &Schema{Type: "array", Items: NumberSchema()},
		},
		{
			name:     "array with text after",
			input:    `[1, 2, 3] is the result`,
			expected: `[1, 2, 3]`,
			schema:   &Schema{Type: "array", Items: NumberSchema()},
		},
		{
			name:     "array with text around",
			input:    `Result: ["a", "b", "c"] done`,
			expected: `["a", "b", "c"]`,
			schema:   &Schema{Type: "array", Items: StringSchema()},
		},
		{
			name:     "nested array",
			input:    `[[1, 2], [3, 4]]`,
			expected: `[[1, 2], [3, 4]]`,
			schema:   &Schema{Type: "array", Items: &Schema{Type: "array", Items: NumberSchema()}},
		},
		{
			name:     "array of numbers looks like objects",
			input:    `[100, 200, 300]`,
			expected: `[100, 200, 300]`,
			schema:   &Schema{Type: "array", Items: NumberSchema()},
		},
		{
			name:     "array with brackets in strings",
			input:    `["a[b]c", "d[e]f"]`,
			expected: `["a[b]c", "d[e]f"]`,
			schema:   &Schema{Type: "array", Items: StringSchema()},
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: `[]`,
			schema:   &Schema{Type: "array", Items: StringSchema()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constrainer := NewJSONConstrainer()
			result, err := constrainer.Constrain(tt.input, tt.schema)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractJSON_ObjectPreferredOverArrayWhenFirst(t *testing.T) {
	// extractJSON prefers object over array because it checks for { first
	input := `{"a": 1} [1, 2]`
	constrainer := NewJSONConstrainer()
	schema := ObjectSchema(map[string]*Schema{
		"a": IntegerSchema(),
	})
	result, err := constrainer.Constrain(input, schema)
	require.NoError(t, err)
	assert.Equal(t, `{"a": 1}`, result)
}


func TestExtractJSON_NoJSON(t *testing.T) {
	constrainer := NewJSONConstrainer()
	schema := StringSchema()
	_, err := constrainer.Constrain("no json here", schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON found")
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name:     "error with path",
			err:      &ValidationError{Path: "user.name", Message: "required"},
			expected: "user.name: required",
		},
		{
			name:     "error without path",
			err:      &ValidationError{Path: "", Message: "invalid JSON"},
			expected: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestValidationResult_AddError(t *testing.T) {
	result := &ValidationResult{Valid: true}
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	result.AddError("field", "error message")
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "field", result.Errors[0].Path)
	assert.Equal(t, "error message", result.Errors[0].Message)
}

func TestValidationResult_ErrorMessages(t *testing.T) {
	result := &ValidationResult{Valid: false}
	result.Errors = []*ValidationError{
		{Path: "a", Message: "error1"},
		{Path: "", Message: "error2"},
		{Path: "b.c", Message: "error3"},
	}

	messages := result.ErrorMessages()
	assert.Len(t, messages, 3)
	assert.Equal(t, "a: error1", messages[0])
	assert.Equal(t, "error2", messages[1])
	assert.Equal(t, "b.c: error3", messages[2])
}

func TestValidate_NilSchema(t *testing.T) {
	result := Validate(`{"any": "value"}`, nil)
	assert.True(t, result.Valid)
}

func TestValidate_EmptyTypeSchema(t *testing.T) {
	schema := &Schema{}
	result := Validate(`"any string"`, schema)
	assert.True(t, result.Valid)

	result = Validate(`123`, schema)
	assert.True(t, result.Valid)

	result = Validate(`{"obj": true}`, schema)
	assert.True(t, result.Valid)
}

func TestValidate_IntegerNotFloat(t *testing.T) {
	schema := IntegerSchema()

	result := Validate(`42`, schema)
	assert.True(t, result.Valid)

	result = Validate(`42.5`, schema)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessages()[0], "expected integer")
}

func TestValidate_NestedObjects(t *testing.T) {
	schema := ObjectSchema(map[string]*Schema{
		"user": ObjectSchema(map[string]*Schema{
			"profile": ObjectSchema(map[string]*Schema{
				"name": StringSchema(),
			}, "name"),
		}, "profile"),
	}, "user")

	validJSON := `{"user":{"profile":{"name":"Alice"}}}`
	result := Validate(validJSON, schema)
	assert.True(t, result.Valid)

	invalidJSON := `{"user":{"profile":{}}}`
	result = Validate(invalidJSON, schema)
	assert.False(t, result.Valid)
}

func TestSchemaBuilder_PropertyWithoutObject(t *testing.T) {
	schema := NewSchemaBuilder().
		Property("name", StringSchema()).
		Build()

	assert.NotNil(t, schema.Properties)
	assert.Equal(t, "string", schema.Properties["name"].Type)
}

func TestRegexConstrainer_ExtractsMatch(t *testing.T) {
	// RegexConstrainer checks if MatchString passes on trimmed output.
	// If it does, it extracts the matched substring via FindString.
	constrainer, err := NewRegexConstrainer(`\d{3}-\d{4}`)
	require.NoError(t, err)

	// This matches and extracts just the pattern match
	result, err := constrainer.Constrain("The number is 123-4567 in the text", nil)
	require.NoError(t, err)
	assert.Equal(t, "123-4567", result) // Now returns just the match
}

func TestRegexConstrainer_FullMatchReturnsMatch(t *testing.T) {
	// When the input exactly matches the pattern, returns the full input
	constrainer, err := NewRegexConstrainer(`^\d{3}-\d{4}$`)
	require.NoError(t, err)

	result, err := constrainer.Constrain("123-4567", nil)
	require.NoError(t, err)
	assert.Equal(t, "123-4567", result)
}

func TestRegexConstrainer_ExtractsWhenNoFullMatch(t *testing.T) {
	// Use anchored pattern that won't match the full string
	constrainer, err := NewRegexConstrainer(`^\d{3}-\d{4}$`)
	require.NoError(t, err)

	// MatchString fails because the full string doesn't match ^...$
	// But FindString should find nothing because the pattern requires start/end anchors
	result, err := constrainer.Constrain("The number is 123-4567 in the text", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match pattern")
	assert.Equal(t, "The number is 123-4567 in the text", result)
}

func TestRegexConstrainer_FindsSubmatch(t *testing.T) {
	// Use a non-anchored pattern that won't match full string via MatchString
	// but can be extracted via FindString
	constrainer, err := NewRegexConstrainer(`ID:\d+`)
	require.NoError(t, err)

	// MatchString returns true because "ID:123" is found in the string
	// FindString extracts just the matched portion
	result, err := constrainer.Constrain("User ID:123 data", nil)
	require.NoError(t, err)
	assert.Equal(t, "ID:123", result) // Now returns just the match
}

func TestValidate_ObjectExpectedGotOther(t *testing.T) {
	schema := ObjectSchema(map[string]*Schema{
		"name": StringSchema(),
	})

	result := Validate(`"not an object"`, schema)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessages()[0], "expected object")

	result = Validate(`[1, 2, 3]`, schema)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessages()[0], "expected object")
}

// Tests for validateNumber with int type conversion.

func TestValidate_NumberWithInt(t *testing.T) {
	// JSON numbers can be parsed as float64 or int depending on the JSON library.
	// In Go's encoding/json, numbers are always float64 when unmarshaling to interface{}.
	// This test ensures the number validator handles edge cases.
	schema := NumberSchema()

	// Standard float
	result := Validate(`3.14`, schema)
	assert.True(t, result.Valid)

	// Integer value (still float64 in Go)
	result = Validate(`42`, schema)
	assert.True(t, result.Valid)

	// Large number
	result = Validate(`1e10`, schema)
	assert.True(t, result.Valid)

	// Negative number
	result = Validate(`-99.5`, schema)
	assert.True(t, result.Valid)
}

// Tests for validateInteger edge cases.

func TestValidate_Integer_EdgeCases(t *testing.T) {
	schema := IntegerSchema()

	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{"positive integer", `100`, true},
		{"zero", `0`, true},
		{"negative integer", `-50`, true},
		{"float with zero decimal", `42.0`, true},
		{"float with non-zero decimal", `42.5`, false},
		{"very small decimal", `42.00001`, false},
		{"large integer", `9999999999`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.input, schema)
			assert.Equal(t, tt.wantValid, result.Valid)
		})
	}
}

// Tests for extractJSON edge cases.

func TestExtractJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "only whitespace",
			input:   "   \n\t   ",
			wantErr: true,
		},
		{
			name:    "plain text no JSON",
			input:   "This is just plain text with no JSON at all",
			wantErr: true,
		},
		{
			name:    "unclosed brace",
			input:   `{"key": "value"`,
			wantErr: true,
		},
		{
			name:    "unclosed bracket",
			input:   `[1, 2, 3`,
			wantErr: true,
		},
		{
			name:    "JSON primitive (valid)",
			input:   `"just a string"`,
			wantErr: false,
		},
		{
			name:    "JSON number primitive",
			input:   `42`,
			wantErr: false,
		},
		{
			name:    "JSON boolean primitive",
			input:   `true`,
			wantErr: false,
		},
		{
			name:    "JSON null primitive",
			input:   `null`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constrainer := NewJSONConstrainer()
			// Use an empty schema that accepts any type
			schema := &Schema{}
			_, err := constrainer.Constrain(tt.input, schema)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Tests for findMatchingBrace edge cases.

func TestExtractJSON_NestedBraces(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "deeply nested object",
			input:    `{"a":{"b":{"c":{"d":"e"}}}}`,
			expected: `{"a":{"b":{"c":{"d":"e"}}}}`,
		},
		{
			name:     "object with escaped quotes",
			input:    `{"key":"value with \"quotes\""}`,
			expected: `{"key":"value with \"quotes\""}`,
		},
		{
			name:     "object with escaped backslash",
			input:    `{"path":"C:\\Users\\test"}`,
			expected: `{"path":"C:\\Users\\test"}`,
		},
		{
			name:     "array with brackets in strings",
			input:    `["text [in] brackets", "more [brackets]"]`,
			expected: `["text [in] brackets", "more [brackets]"]`,
		},
		{
			name:     "object with array containing objects",
			input:    `{"items":[{"id":1},{"id":2}]}`,
			expected: `{"items":[{"id":1},{"id":2}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constrainer := NewJSONConstrainer()
			schema := &Schema{}
			result, err := constrainer.Constrain(tt.input, schema)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for RegexConstrainer when MatchString fails but FindString succeeds.

func TestRegexConstrainer_FindStringFallback(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		input       string
		wantResult  string
		wantErr     bool
		errContains string
	}{
		{
			// Pattern matches anywhere, extracts just the match
			name:       "partial pattern extracts match",
			pattern:    `\d+`,
			input:      "abc 123 def",
			wantResult: "123", // Extracts just the match now
			wantErr:    false,
		},
		{
			// Anchored pattern that doesn't match full string
			// MatchString fails, returns error
			name:        "anchored pattern no match",
			pattern:     `^\d+$`,
			input:       "abc 123 def",
			wantResult:  "abc 123 def",
			wantErr:     true,
			errContains: "does not match pattern",
		},
		{
			// Leading/trailing whitespace is trimmed, then exact match
			name:       "whitespace trimming exact match",
			pattern:    `^hello$`,
			input:      "  hello  ",
			wantResult: "hello",
			wantErr:    false,
		},
		{
			// Partial pattern inside trimmed text
			name:       "whitespace trimming partial match",
			pattern:    `hello`,
			input:      "  say hello world  ",
			wantResult: "hello", // Extracts just hello
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constrainer, err := NewRegexConstrainer(tt.pattern)
			require.NoError(t, err)

			result, err := constrainer.Constrain(tt.input, nil)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

// Test for validateNumber when given non-numeric type.

func TestValidate_NumberExpectedGotOther(t *testing.T) {
	schema := NumberSchema()

	result := Validate(`"not a number"`, schema)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessages()[0], "expected number")

	result = Validate(`true`, schema)
	assert.False(t, result.Valid)

	result = Validate(`{"a":1}`, schema)
	assert.False(t, result.Valid)

	result = Validate(`[1,2,3]`, schema)
	assert.False(t, result.Valid)
}

// Test for validateInteger when given non-numeric type.

func TestValidate_IntegerExpectedGotOther(t *testing.T) {
	schema := IntegerSchema()

	result := Validate(`"not an integer"`, schema)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessages()[0], "expected integer")

	result = Validate(`true`, schema)
	assert.False(t, result.Valid)

	result = Validate(`{"a":1}`, schema)
	assert.False(t, result.Valid)
}

// Test for RegexConstrainer FindString extraction.

func TestRegexConstrainer_FindStringExtraction(t *testing.T) {
	// The refactored code now extracts matches when MatchString succeeds.
	// When MatchString returns true, FindString is called to extract the match.
	constrainer, err := NewRegexConstrainer(`\d{3}`)
	require.NoError(t, err)

	// MatchString finds the pattern, FindString extracts it
	result, err := constrainer.Constrain("abc 123 def", nil)
	require.NoError(t, err)
	assert.Equal(t, "123", result) // Returns just the extracted match
}

// Test for validateNumber with int type value.
// JSON unmarshals integers to float64, but we test the int case.

func TestValidate_NumberConstraints_Boundaries(t *testing.T) {
	min := 10.0
	max := 20.0
	schema := &Schema{
		Type:    "number",
		Minimum: &min,
		Maximum: &max,
	}

	// Exactly at minimum.
	result := Validate(`10`, schema)
	assert.True(t, result.Valid)

	// Exactly at maximum.
	result = Validate(`20`, schema)
	assert.True(t, result.Valid)

	// Just below minimum.
	result = Validate(`9.9`, schema)
	assert.False(t, result.Valid)

	// Just above maximum.
	result = Validate(`20.1`, schema)
	assert.False(t, result.Valid)
}

// Test for validateInteger constraints.

func TestValidate_IntegerConstraints_Boundaries(t *testing.T) {
	min := 0.0
	max := 100.0
	schema := &Schema{
		Type:    "integer",
		Minimum: &min,
		Maximum: &max,
	}

	// Exactly at minimum.
	result := Validate(`0`, schema)
	assert.True(t, result.Valid)

	// Exactly at maximum.
	result = Validate(`100`, schema)
	assert.True(t, result.Valid)

	// Below minimum.
	result = Validate(`-1`, schema)
	assert.False(t, result.Valid)

	// Above maximum.
	result = Validate(`101`, schema)
	assert.False(t, result.Valid)
}

// Test for findMatchingBrace with empty string.

func TestExtractJSON_EmptyString(t *testing.T) {
	constrainer := NewJSONConstrainer()
	_, err := constrainer.Constrain("", &Schema{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON found")
}

// Test for findMatchingBrace edge case: only open brace.

func TestExtractJSON_OnlyOpenBrace(t *testing.T) {
	constrainer := NewJSONConstrainer()
	_, err := constrainer.Constrain("{", &Schema{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON found")
}

// Test for findMatchingBrace with mismatched braces.

func TestExtractJSON_MismatchedBraces(t *testing.T) {
	constrainer := NewJSONConstrainer()

	_, err := constrainer.Constrain("{ [ } ]", &Schema{})
	assert.Error(t, err)

	_, err = constrainer.Constrain("[ { ] }", &Schema{})
	assert.Error(t, err)
}

// Tests for validateNumber and validateInteger with int type (for direct calls).
// These cover branches that aren't reached via JSON unmarshaling (which always uses float64).

func TestValidate_DirectIntType(t *testing.T) {
	// When calling ValidateValue directly with an int value (not via JSON unmarshal),
	// the int case branch is executed. This tests the validateNumber int case.

	// Test validateNumber with int type
	result := ValidateValue(42, NumberSchema(), "")
	assert.True(t, result.Valid)

	// Test validateNumber with int and constraints
	min := 10.0
	max := 100.0
	schema := &Schema{
		Type:    "number",
		Minimum: &min,
		Maximum: &max,
	}
	result = ValidateValue(50, schema, "")
	assert.True(t, result.Valid)

	result = ValidateValue(5, schema, "")
	assert.False(t, result.Valid)

	result = ValidateValue(150, schema, "")
	assert.False(t, result.Valid)
}

func TestValidate_DirectIntType_Integer(t *testing.T) {
	// Test validateInteger with int type
	result := ValidateValue(42, IntegerSchema(), "")
	assert.True(t, result.Valid)

	// Test validateInteger with int and constraints
	min := 0.0
	max := 100.0
	schema := &Schema{
		Type:    "integer",
		Minimum: &min,
		Maximum: &max,
	}
	result = ValidateValue(50, schema, "")
	assert.True(t, result.Valid)

	result = ValidateValue(-5, schema, "")
	assert.False(t, result.Valid)

	result = ValidateValue(150, schema, "")
	assert.False(t, result.Valid)
}

func TestValidateValue_WithPath(t *testing.T) {
	// Test that path is properly set in errors
	result := ValidateValue("not a number", NumberSchema(), "$.field")
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessages()[0], "$.field")
}

// Test for the edge case where MatchString returns true but FindString returns empty.
// This is theoretically impossible with standard Go regex, but we have a defensive fallback.

func TestRegexConstrainer_MatchStringTrueFindStringEmpty(t *testing.T) {
	// The only way MatchString returns true but FindString returns empty is with
	// zero-width assertions (like (?=foo) or ^$ on empty string).
	// Let's try with the ^ pattern on empty string.
	constrainer, err := NewRegexConstrainer(`^`)
	require.NoError(t, err)

	// "^" matches at position 0 of any string but FindString returns ""
	// because there's no actual content to extract.
	result, err := constrainer.Constrain("hello", nil)
	require.NoError(t, err)
	// Since FindString returns empty for "^", we fall back to returning the full output
	assert.Equal(t, "hello", result)
}

func TestRegexConstrainer_EmptyMatchFallback(t *testing.T) {
	// Test with a pattern that matches but extracts nothing
	// Using lookahead-like behavior with ^
	constrainer, err := NewRegexConstrainer(`^()`)
	require.NoError(t, err)

	result, err := constrainer.Constrain("test", nil)
	require.NoError(t, err)
	// Empty match, falls back to full output
	assert.Equal(t, "test", result)
}

// Test for findMatchingBrace edge cases - called via extractJSON.

func TestExtractJSON_FindMatchingBrace_EmptyString(t *testing.T) {
	// extractJSON is called with empty string - returns ""
	constrainer := NewJSONConstrainer()
	_, err := constrainer.Constrain("", &Schema{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON found")
}

func TestExtractJSON_FindMatchingBrace_JustOpenBrace(t *testing.T) {
	// String starting with { but never closing
	constrainer := NewJSONConstrainer()
	_, err := constrainer.Constrain("{", &Schema{})
	assert.Error(t, err)
}

func TestExtractJSON_FindMatchingBrace_BraceInString(t *testing.T) {
	// Braces inside strings should be ignored
	constrainer := NewJSONConstrainer()
	result, err := constrainer.Constrain(`{"key": "value with { and }"}`, &Schema{})
	require.NoError(t, err)
	assert.Equal(t, `{"key": "value with { and }"}`, result)
}

func TestExtractJSON_FindMatchingBrace_EscapedQuote(t *testing.T) {
	// Escaped quotes inside strings
	constrainer := NewJSONConstrainer()
	result, err := constrainer.Constrain(`{"key": "value \"with\" quotes"}`, &Schema{})
	require.NoError(t, err)
	assert.Equal(t, `{"key": "value \"with\" quotes"}`, result)
}

func TestExtractJSON_FindMatchingBrace_NestedEscapes(t *testing.T) {
	// Multiple levels of escaping
	constrainer := NewJSONConstrainer()
	result, err := constrainer.Constrain(`{"key": "\\"}`, &Schema{})
	require.NoError(t, err)
	assert.Equal(t, `{"key": "\\"}`, result)
}

// --- Direct tests for findMatchingBrace internal function ---

func TestFindMatchingBrace_EmptyString(t *testing.T) {
	// Test with empty string - should return -1 (defensive check).
	result := findMatchingBrace("", '{', '}')
	assert.Equal(t, -1, result)
}

func TestFindMatchingBrace_WrongFirstChar(t *testing.T) {
	// Test with string that doesn't start with the open brace.
	result := findMatchingBrace("abc{}", '{', '}')
	assert.Equal(t, -1, result)
}

func TestFindMatchingBrace_ValidObject(t *testing.T) {
	// Test with valid JSON object.
	result := findMatchingBrace(`{"key": "value"}`, '{', '}')
	assert.Equal(t, 15, result)
}

func TestFindMatchingBrace_ValidArray(t *testing.T) {
	// Test with valid JSON array.
	result := findMatchingBrace(`[1, 2, 3]`, '[', ']')
	assert.Equal(t, 8, result)
}

func TestFindMatchingBrace_NestedObjects(t *testing.T) {
	// Test with nested objects.
	result := findMatchingBrace(`{"outer": {"inner": "value"}}`, '{', '}')
	assert.Equal(t, 28, result)
}

func TestFindMatchingBrace_BracesInString(t *testing.T) {
	// Test with braces inside strings (should be ignored).
	result := findMatchingBrace(`{"key": "value { and }"}`, '{', '}')
	assert.Equal(t, 23, result)
}

func TestFindMatchingBrace_EscapedQuotes(t *testing.T) {
	// Test with escaped quotes.
	result := findMatchingBrace(`{"key": "value \"with\" quotes"}`, '{', '}')
	assert.Equal(t, 31, result)
}

func TestFindMatchingBrace_Unbalanced(t *testing.T) {
	// Test with unbalanced braces.
	result := findMatchingBrace(`{"key": "value"`, '{', '}')
	assert.Equal(t, -1, result)
}

func TestFindMatchingBrace_OnlyOpen(t *testing.T) {
	// Test with only opening brace.
	result := findMatchingBrace(`{`, '{', '}')
	assert.Equal(t, -1, result)
}
