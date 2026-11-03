package engine

import (
	"encoding/json"
	"fmt"
	"math"
)

type (
	// Missing distinguishes absent data from an explicit JSON null
	Missing struct{}

	operands struct {
		left  any
		right any
	}
)

// MarshalJSON rejects absent values instead of encoding them as null
func (Missing) MarshalJSON() ([]byte, error) {
	return nil, ErrMissing
}

// IsMissing reports whether a value represents absent data
func IsMissing(value any) bool {
	_, ok := value.(Missing)
	return ok
}

// number uses NaN for invalid inputs so finiteness checks stay inline
func number(value any) float64 {
	switch value := value.(type) {
	case float64:
		return value

	case int:
		return float64(value)

	default:
		return jsonNumber(value)
	}
}

func jsonNumber(value any) float64 {
	n, ok := value.(json.Number)
	if !ok {
		return math.NaN()
	}
	result, err := n.Float64()
	if err != nil {
		return math.NaN()
	}
	return result
}

func operandError(operation string) error {
	return fmt.Errorf("%w: %s", ErrType, operation)
}

func structuralEqual(left, right any) bool {
	switch left.(type) {
	case []any, map[string]any:
		return containerEqual(left, right)

	default:
		return scalarEqual(left, right)
	}
}

// containerEqual walks arrays and objects with an explicit stack, so only
// container comparisons pay for setting one up
func containerEqual(left, right any) bool {
	var pairs [inlineValues]operands
	pairs[0] = operands{left: left, right: right}
	pending := pairs[:1]
	matched := true
	for len(pending) > 0 && matched {
		last := len(pending) - 1
		pair := pending[last]
		pending = pending[:last]
		switch left := pair.left.(type) {
		case []any:
			right, ok := pair.right.([]any)
			if !ok || len(left) != len(right) {
				matched = false
				continue
			}
			for i, item := range left {
				pending = append(pending, operands{
					left:  item,
					right: right[i],
				})
			}

		case map[string]any:
			right, ok := pair.right.(map[string]any)
			if !ok || len(left) != len(right) {
				matched = false
				continue
			}
			for key, item := range left {
				child, ok := right[key]
				if !ok {
					matched = false
					break
				}
				pending = append(pending, operands{
					left:  item,
					right: child,
				})
			}

		default:
			matched = scalarEqual(pair.left, pair.right)
		}
	}
	return matched
}

// scalarEqual compares values that are neither arrays nor objects
func scalarEqual(left, right any) bool {
	switch left := left.(type) {
	case nil:
		return right == nil

	case Missing:
		return IsMissing(right)

	case bool:
		right, ok := right.(bool)
		return ok && left == right

	case string:
		right, ok := right.(string)
		return ok && left == right

	case *Function:
		right, ok := right.(*Function)
		return ok && left == right

	case float64:
		return left == number(right)

	default:
		return number(left) == number(right)
	}
}
