package engine

// Error is a comparable error value shared with the public API
type Error string

const (
	SyntaxError     Error = "invalid expression"
	TypeError       Error = "invalid operand type"
	MissingError    Error = "missing value"
	ArithmeticError Error = "invalid arithmetic"
	FunctionError   Error = "unknown function or invalid arguments"
	LimitError      Error = "expression limit exceeded"
)

// Error returns the error message
func (e Error) Error() string {
	return string(e)
}
