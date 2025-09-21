package jsonschema

// Schema is a minimal JSON Schema representation used for export.
// Keep this struct small for MVP and extend incrementally.
type Schema struct {
	// Core
	Type    string `json:"type,omitempty"`
	Format  string `json:"format,omitempty"`
	Default any    `json:"default,omitempty"`

	// Object
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	AdditionalProperties any                `json:"additionalProperties,omitempty"`

	// Array
	Items    *Schema `json:"items,omitempty"`
	MinItems *int    `json:"minItems,omitempty"`
	MaxItems *int    `json:"maxItems,omitempty"`

	// Union
	OneOf []*Schema `json:"oneOf,omitempty"`
}
