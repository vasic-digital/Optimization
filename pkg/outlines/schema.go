// Package outlines provides structured output constraints for LLM responses
// using JSON Schema validation and regex pattern matching.
package outlines

import (
	"encoding/json"
	"fmt"
)

// Schema defines an output structure using JSON Schema.
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	MinItems             *int               `json:"minItems,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty"`
	UniqueItems          bool               `json:"uniqueItems,omitempty"`
	AdditionalProperties *bool              `json:"additionalProperties,omitempty"`
	Description          string             `json:"description,omitempty"`
	Default              interface{}        `json:"default,omitempty"`
	Format               string             `json:"format,omitempty"`
	OneOf                []*Schema          `json:"oneOf,omitempty"`
	AnyOf                []*Schema          `json:"anyOf,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
}

// ParseSchema parses a JSON schema from bytes.
func ParseSchema(data []byte) (*Schema, error) {
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}
	return &schema, nil
}

// String returns the JSON representation of the schema.
func (s *Schema) String() string {
	bytes, _ := json.MarshalIndent(s, "", "  ")
	return string(bytes)
}

// IsRequired checks if a property is required.
func (s *Schema) IsRequired(property string) bool {
	for _, req := range s.Required {
		if req == property {
			return true
		}
	}
	return false
}

// SchemaBuilder provides a fluent API for building schemas.
type SchemaBuilder struct {
	schema *Schema
}

// NewSchemaBuilder creates a new schema builder.
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{schema: &Schema{}}
}

// Object sets the type to object.
func (b *SchemaBuilder) Object() *SchemaBuilder {
	b.schema.Type = "object"
	if b.schema.Properties == nil {
		b.schema.Properties = make(map[string]*Schema)
	}
	return b
}

// Array sets the type to array.
func (b *SchemaBuilder) Array() *SchemaBuilder {
	b.schema.Type = "array"
	return b
}

// StringType sets the type to string.
func (b *SchemaBuilder) StringType() *SchemaBuilder {
	b.schema.Type = "string"
	return b
}

// NumberType sets the type to number.
func (b *SchemaBuilder) NumberType() *SchemaBuilder {
	b.schema.Type = "number"
	return b
}

// IntegerType sets the type to integer.
func (b *SchemaBuilder) IntegerType() *SchemaBuilder {
	b.schema.Type = "integer"
	return b
}

// BooleanType sets the type to boolean.
func (b *SchemaBuilder) BooleanType() *SchemaBuilder {
	b.schema.Type = "boolean"
	return b
}

// Property adds a property to an object schema.
func (b *SchemaBuilder) Property(
	name string,
	schema *Schema,
) *SchemaBuilder {
	if b.schema.Properties == nil {
		b.schema.Properties = make(map[string]*Schema)
	}
	b.schema.Properties[name] = schema
	return b
}

// RequiredProps marks properties as required.
func (b *SchemaBuilder) RequiredProps(
	properties ...string,
) *SchemaBuilder {
	b.schema.Required = append(b.schema.Required, properties...)
	return b
}

// Items sets the items schema for an array.
func (b *SchemaBuilder) Items(schema *Schema) *SchemaBuilder {
	b.schema.Items = schema
	return b
}

// EnumValues sets allowed values.
func (b *SchemaBuilder) EnumValues(
	values ...interface{},
) *SchemaBuilder {
	b.schema.Enum = values
	return b
}

// SetPattern sets a regex pattern for strings.
func (b *SchemaBuilder) SetPattern(pattern string) *SchemaBuilder {
	b.schema.Pattern = pattern
	return b
}

// SetDescription sets the schema description.
func (b *SchemaBuilder) SetDescription(desc string) *SchemaBuilder {
	b.schema.Description = desc
	return b
}

// Build returns the constructed schema.
func (b *SchemaBuilder) Build() *Schema {
	return b.schema
}

// Helper constructors for common schemas.

// StringSchema creates a string schema.
func StringSchema() *Schema {
	return &Schema{Type: "string"}
}

// IntegerSchema creates an integer schema.
func IntegerSchema() *Schema {
	return &Schema{Type: "integer"}
}

// NumberSchema creates a number schema.
func NumberSchema() *Schema {
	return &Schema{Type: "number"}
}

// BooleanSchema creates a boolean schema.
func BooleanSchema() *Schema {
	return &Schema{Type: "boolean"}
}

// ArraySchema creates an array schema with the given item type.
func ArraySchema(items *Schema) *Schema {
	return &Schema{Type: "array", Items: items}
}

// ObjectSchema creates an object schema with properties.
func ObjectSchema(
	properties map[string]*Schema,
	required ...string,
) *Schema {
	return &Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}
