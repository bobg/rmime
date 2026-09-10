package rmime

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/transform"
)

// Field is a field in a message or message-part header.
type Field struct {
	N string   `json:"name"`  // Name of the field.
	V []string `json:"value"` // Values of the field.
}

// Name returns the name of a field in canonical form.
func (f Field) Name() string {
	result := strings.TrimSpace(f.N)
	result = strings.ToLower(result)

	titler := cases.Title(language.AmericanEnglish)
	result, _, _ = transform.String(titler, result)

	return result
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
