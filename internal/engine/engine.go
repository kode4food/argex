package engine

import (
	"fmt"

	"github.com/kode4food/argex/internal/isa"
)

type (
	// Expression is an immutable parsed expression
	Expression struct {
		root *node
	}

	// Missing distinguishes absent data from an explicit JSON null
	Missing struct{}

	program struct {
		code       isa.Instructions
		constants  []any
		iterations []iterationSpec
		stackSize  int
		localCount int
		numeric    bool
	}

	node struct {
		kind     string
		text     string
		value    any
		children []*node
		keys     []string
		optional bool
	}
)

const maxDepth = 256

// MarshalJSON rejects absent values instead of encoding them as null
func (Missing) MarshalJSON() ([]byte, error) {
	return nil, MissingError
}

// Compile prepares an expression for repeated evaluation
func Compile(expr *Expression) (func(map[string]any) (any, error), error) {
	if expr == nil || expr.root == nil {
		return nil, fmt.Errorf("%w: empty expression", SyntaxError)
	}
	c := &compiler{program: &program{}}
	if err := c.compile(expr.root, 0); err != nil {
		return nil, err
	}
	return c.program.evaluate, nil
}

// IsMissing reports whether a value represents absent data
func IsMissing(value any) bool {
	_, ok := value.(Missing)
	return ok
}
