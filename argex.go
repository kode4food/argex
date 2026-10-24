// Package argex evaluates pure expressions over JSON values
package argex

import "github.com/kode4food/argex/internal/engine"

type (
	// Program is a compiled function, safe for concurrent calls
	Program func(map[string]any) (any, error)

	// Expression is an immutable parsed expression
	Expression = engine.Expression

	// Missing distinguishes absent data from an explicit JSON null
	Missing = engine.Missing
)

var (
	ErrSyntax     error = engine.SyntaxError
	ErrType       error = engine.TypeError
	ErrMissing    error = engine.MissingError
	ErrArithmetic error = engine.ArithmeticError
	ErrFunction   error = engine.FunctionError
	ErrLimit      error = engine.LimitError
)

// Parse parses a complete expression; whitespace never determines grouping
func Parse(source string) (*Expression, error) {
	return engine.Parse(source)
}

// Eval evaluates source against named JSON values
func Eval(source string, variables map[string]any) (any, error) {
	expr, err := Parse(source)
	if err != nil {
		return nil, err
	}
	program, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return program(variables)
}

// Compile prepares an expression for repeated evaluation
func Compile(expr *Expression) (Program, error) {
	return engine.Compile(expr)
}

// IsMissing reports whether a value represents absent data
func IsMissing(value any) bool {
	return engine.IsMissing(value)
}
