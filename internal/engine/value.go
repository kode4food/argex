package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
)

type operands struct {
	left  any
	right any
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

func nullish(value any) bool {
	return value == nil || IsMissing(value)
}

func operandError(operation string) error {
	return fmt.Errorf("%w: %s", ErrType, operation)
}

func structuralEqual(left, right any) bool {
	var pairs [inlineValues]operands
	pairs[0] = operands{left: left, right: right}
	pending := pairs[:1]
	matched := true
	for len(pending) > 0 && matched {
		last := len(pending) - 1
		pair := pending[last]
		pending = pending[:last]
		switch left := pair.left.(type) {
		case nil:
			matched = pair.right == nil

		case Missing:
			matched = IsMissing(pair.right)

		case bool:
			right, ok := pair.right.(bool)
			matched = ok && left == right

		case string:
			right, ok := pair.right.(string)
			matched = ok && left == right

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
			a := number(pair.left)
			b := number(pair.right)
			if a-a == 0 {
				matched = b-b == 0 && a == b
				continue
			}
			matched = reflect.DeepEqual(pair.left, pair.right)
		}
	}
	return matched
}
