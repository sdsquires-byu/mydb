package mydb

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierPart = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Identifier is a table or column name that will be validated before use.
type Identifier struct {
	name string
}

// Ident marks a table or column name as an SQL identifier.
func Ident(name string) Identifier {
	return Identifier{name: name}
}

// Parameter is a value that should be bound as a query argument.
type Parameter struct {
	value any
}

// Value marks a value as a bind parameter.
func Value(value any) Parameter {
	return Parameter{value: value}
}

// Join assembles SQL fragments into a Statement.
//
// Strings are treated as trusted SQL fragments, Ident values are validated
// identifiers, and Value values become placeholders with matching arguments.
func Join(parts ...any) (Statement, error) {
	tokens := make([]string, 0, len(parts))
	args := make([]any, 0)

	for _, part := range parts {
		switch p := part.(type) {
		case string:
			if text := strings.TrimSpace(p); text != "" {
				tokens = append(tokens, text)
			}
		case Identifier:
			name, err := p.sql()
			if err != nil {
				return Statement{}, err
			}
			tokens = append(tokens, name)
		case Parameter:
			tokens = append(tokens, "?")
			args = append(args, p.value)
		case Statement:
			if text := strings.TrimSpace(p.Text); text != "" {
				tokens = append(tokens, text)
				args = append(args, p.Args...)
			}
		default:
			return Statement{}, fmt.Errorf("%w: %T", ErrUnsupportedSQLPart, part)
		}
	}

	return Stmt(strings.Join(tokens, " "), args...), nil
}

func (i Identifier) sql() (string, error) {
	parts := strings.Split(i.name, ".")
	if len(parts) == 0 {
		return "", fmt.Errorf("%w: %q", ErrInvalidIdentifier, i.name)
	}

	for _, part := range parts {
		if !identifierPart.MatchString(part) {
			return "", fmt.Errorf("%w: %q", ErrInvalidIdentifier, i.name)
		}
	}

	return strings.Join(parts, "."), nil
}
