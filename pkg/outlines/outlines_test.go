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
			name:    "partial match passes MatchString",
			pattern: `\d{3}-\d{4}`,
			output:  "call 123-4567 now",
			want:    "call 123-4567 now",
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
