package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

type (
	builtin struct {
		call     function
		arity    int
		variadic bool
	}

	function func(...any) (any, error)
)

var builtins = map[string]builtin{
	"length":     {call: length, arity: 1},
	"abs":        {call: absolute, arity: 1},
	"min":        {call: minimum, arity: 1, variadic: true},
	"max":        {call: maximum, arity: 1, variadic: true},
	"sum":        {call: sum, arity: 1},
	"round":      {call: round, arity: 1},
	"floor":      {call: floor, arity: 1},
	"ceil":       {call: ceil, arity: 1},
	"number":     {call: toNumber, arity: 1},
	"string":     {call: toString, arity: 1},
	"startsWith": {call: startsWith, arity: 2},
	"endsWith":   {call: endsWith, arity: 2},
	"includes":   {call: includes, arity: 2},
	"lower":      {call: lower, arity: 1},
	"upper":      {call: upper, arity: 1},
	"trim":       {call: trim, arity: 1},
	"split":      {call: split, arity: 2},
	"join":       {call: join, arity: 2},
	"keys":       {call: keys, arity: 1},
	"values":     {call: values, arity: 1},
}

func length(args ...any) (any, error) {
	switch value := args[0].(type) {
	case string:
		return float64(utf8.RuneCountInString(value)), nil

	case []any:
		return float64(len(value)), nil

	case map[string]any:
		return float64(len(value)), nil

	default:
		return nil, operandError("length")
	}
}

func absolute(args ...any) (any, error) {
	n := number(args[0])
	if n-n != 0 {
		return nil, operandError("abs")
	}
	return math.Abs(n), nil
}

func minimum(args ...any) (any, error) {
	result := number(args[0])
	if result-result != 0 {
		return nil, operandError("min")
	}
	for _, arg := range args[1:] {
		n := number(arg)
		if n-n != 0 {
			return nil, operandError("min")
		}
		result = min(result, n)
	}
	return result, nil
}

func maximum(args ...any) (any, error) {
	result := number(args[0])
	if result-result != 0 {
		return nil, operandError("max")
	}
	for _, arg := range args[1:] {
		n := number(arg)
		if n-n != 0 {
			return nil, operandError("max")
		}
		result = max(result, n)
	}
	return result, nil
}

func sum(args ...any) (any, error) {
	items, ok := args[0].([]any)
	if !ok {
		return nil, operandError("sum")
	}
	result := 0.0
	for _, item := range items {
		n := number(item)
		if n-n != 0 {
			return nil, operandError("sum")
		}
		result += n
		if math.IsNaN(result) || math.IsInf(result, 0) {
			return nil, fmt.Errorf("%w: sum", ErrArithmetic)
		}
	}
	return result, nil
}

func round(args ...any) (any, error) {
	n := number(args[0])
	if n-n != 0 {
		return nil, operandError("round")
	}
	return math.Round(n), nil
}

func floor(args ...any) (any, error) {
	n := number(args[0])
	if n-n != 0 {
		return nil, operandError("floor")
	}
	return math.Floor(n), nil
}

func ceil(args ...any) (any, error) {
	n := number(args[0])
	if n-n != 0 {
		return nil, operandError("ceil")
	}
	return math.Ceil(n), nil
}

func toNumber(args ...any) (any, error) {
	value := args[0]
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if len(text) == 0 || !json.Valid([]byte(text)) {
			return nil, operandError("number")
		}
		n, err := strconv.ParseFloat(text, 64)
		if err != nil || n-n != 0 {
			return nil, operandError("number")
		}
		return n, nil
	}
	n := number(value)
	if n-n != 0 {
		return nil, operandError("number")
	}
	return n, nil
}

func toString(args ...any) (any, error) {
	switch value := args[0].(type) {
	case string:
		return args[0], nil

	case nil:
		return "null", nil

	case bool:
		return strconv.FormatBool(value), nil

	default:
		n := number(value)
		if n-n != 0 {
			return nil, operandError("string")
		}
		return strconv.FormatFloat(n, 'g', -1, 64), nil
	}
}

func startsWith(args ...any) (any, error) {
	text, ok := args[0].(string)
	if !ok {
		return nil, operandError("startsWith")
	}
	prefix, ok := args[1].(string)
	if !ok {
		return nil, operandError("startsWith")
	}
	return strings.HasPrefix(text, prefix), nil
}

func endsWith(args ...any) (any, error) {
	text, ok := args[0].(string)
	if !ok {
		return nil, operandError("endsWith")
	}
	suffix, ok := args[1].(string)
	if !ok {
		return nil, operandError("endsWith")
	}
	return strings.HasSuffix(text, suffix), nil
}

func includes(args ...any) (any, error) {
	text, ok := args[0].(string)
	if !ok {
		return nil, operandError("includes")
	}
	part, ok := args[1].(string)
	if !ok {
		return nil, operandError("includes")
	}
	return strings.Contains(text, part), nil
}

func lower(args ...any) (any, error) {
	if text, ok := args[0].(string); ok {
		return strings.ToLower(text), nil
	}
	return nil, operandError("lower")
}

func upper(args ...any) (any, error) {
	if text, ok := args[0].(string); ok {
		return strings.ToUpper(text), nil
	}
	return nil, operandError("upper")
}

func trim(args ...any) (any, error) {
	if text, ok := args[0].(string); ok {
		return strings.TrimSpace(text), nil
	}
	return nil, operandError("trim")
}

func split(args ...any) (any, error) {
	text, ok := args[0].(string)
	if !ok {
		return nil, operandError("split")
	}
	separator, ok := args[1].(string)
	if !ok {
		return nil, operandError("split")
	}
	parts := strings.Split(text, separator)
	result := make([]any, len(parts))
	for i, part := range parts {
		result[i] = part
	}
	return result, nil
}

func join(args ...any) (any, error) {
	items, ok := args[0].([]any)
	if !ok {
		return nil, operandError("join")
	}
	separator, ok := args[1].(string)
	if !ok {
		return nil, operandError("join")
	}
	var result strings.Builder
	for i, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, operandError("join")
		}
		if i > 0 {
			result.WriteString(separator)
		}
		result.WriteString(text)
	}
	return result.String(), nil
}

func keys(args ...any) (any, error) {
	object, ok := args[0].(map[string]any)
	if !ok {
		return nil, operandError("keys")
	}
	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	slices.Sort(names)
	result := make([]any, len(names))
	for i, name := range names {
		result[i] = name
	}
	return result, nil
}

func values(args ...any) (any, error) {
	object, ok := args[0].(map[string]any)
	if !ok {
		return nil, operandError("values")
	}
	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	slices.Sort(names)
	result := make([]any, len(names))
	for i, name := range names {
		result[i] = object[name]
	}
	return result, nil
}
