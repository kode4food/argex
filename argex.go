// Package argex evaluates Argyll Expressions over JSON values
package argex

import "github.com/kode4food/argex/internal/engine"

type (
	// Program is a compiled expression, safe for concurrent evaluation
	Program = engine.Program

	// Expression is an immutable parsed expression
	Expression = engine.Expression

	// Env holds immutable global bindings for compiling expressions
	Env = engine.Env

	// Function is a callable value in Argyll Expressions
	Function = engine.Function

	// Missing distinguishes absent data from an explicit JSON null
	Missing = engine.Missing

	// TypeTag is a single possible runtime type for an expression's result
	TypeTag = engine.TypeTag

	// TypeSet is the union of every TypeTag an expression might evaluate to
	TypeSet = engine.TypeSet
)

const (
	TypeBool     TypeTag = engine.TypeBool
	TypeNumber   TypeTag = engine.TypeNumber
	TypeString   TypeTag = engine.TypeString
	TypeArray    TypeTag = engine.TypeArray
	TypeObject   TypeTag = engine.TypeObject
	TypeNull     TypeTag = engine.TypeNull
	TypeMissing  TypeTag = engine.TypeMissing
	TypeFunction TypeTag = engine.TypeFunction
	TypeAny      TypeTag = engine.TypeAny
)

var (
	ErrSyntax     error = engine.ErrSyntax
	ErrType       error = engine.ErrType
	ErrMissing    error = engine.ErrMissing
	ErrArithmetic error = engine.ErrArithmetic
	ErrFunction   error = engine.ErrFunction
	ErrLimit      error = engine.ErrLimit
)

// NewEnv returns an empty environment
func NewEnv() *Env {
	return engine.NewEnv()
}

// DefaultEnv returns the shared environment containing the builtin functions
func DefaultEnv() *Env {
	return engine.DefaultEnv()
}

// Parse parses a complete expression. Whitespace never determines grouping
func Parse(source string) (*Expression, error) {
	return engine.Parse(source)
}

// Eval evaluates source against the builtins with this as its JSON input
func Eval(source string, this any) (any, error) {
	expr, err := Parse(source)
	if err != nil {
		return nil, err
	}
	program, err := DefaultEnv().Compile(expr)
	if err != nil {
		return nil, err
	}
	return program.Eval(this)
}

// IsMissing reports whether a value represents absent data
func IsMissing(value any) bool {
	return engine.IsMissing(value)
}
