package engine

import (
	"encoding/json"
	"fmt"
	"math"
)

// number uses NaN for invalid inputs so finiteness checks stay inline
func number(value any) float64 {
	switch value := value.(type) {
	case float64:
		return value

	case int:
		return float64(value)
	}
	return jsonNumber(value)
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
	return fmt.Errorf("%w: %s", TypeError, operation)
}
