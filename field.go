package rmime

import "strings"

// Field is a field in a message or message-part header.
type Field struct {
	N string   `json:"name"`  // Name of the field.
	V []string `json:"value"` // Values of the field.
}

// Name returns the name of a field in canonical form.
func (f Field) Name() string {
	return strings.Title(strings.TrimSpace(f.N))
}

// Value returns the values of the field combined with canonical
// spacing.
func (f Field) Value() string {
	result := strings.TrimSpace(f.V[0])
	for i := 1; i < len(f.V); i++ {
		result += " " + strings.TrimSpace(f.V[i])
	}
	return result
}
