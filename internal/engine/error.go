package engine

// Error is a comparable error value shared with the public API
type Error string

var (
	ErrSyntax     Error = "invalid expression"
	ErrType       Error = "invalid operand type"
	ErrMissing    Error = "missing value"
	ErrArithmetic Error = "invalid arithmetic"
	ErrFunction   Error = "unknown function or invalid arguments"
	ErrLimit      Error = "expression limit exceeded"
)

// Error returns the error message
func (e Error) Error() string {
	return string(e)
}
