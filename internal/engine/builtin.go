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
		params   []TypeSet
		result   TypeTag
	}

	function func(...any) (any, error)
)

const (
	fnLength     = "length"
	fnAbs        = "abs"
	fnMin        = "min"
	fnMax        = "max"
	fnSum        = "sum"
	fnRound      = "round"
	fnFloor      = "floor"
	fnCeil       = "ceil"
	fnNumber     = "number"
	fnString     = "string"
	fnStartsWith = "startsWith"
	fnEndsWith   = "endsWith"
	fnIncludes   = "includes"
	fnLower      = "lower"
	fnUpper      = "upper"
	fnTrim       = "trim"
	fnSplit      = "split"
	fnJoin       = "join"
	fnKeys       = "keys"
	fnValues     = "values"
)

var (
	numberType     = TypeSet(TypeNumber)
	stringType     = TypeSet(TypeString)
	objectType     = TypeSet(TypeObject)
	arrayType      = TypeSet(TypeArray)
	lengthableType = stringType | arrayType | objectType
	textableType   = stringType | numberType |
		TypeSet(TypeBool) | TypeSet(TypeNull)
)

var builtins = map[string]builtin{
	fnLength: {
		call:   length,
		arity:  1,
		params: []TypeSet{lengthableType},
		result: TypeNumber,
	},
	fnAbs: {
		call:   absolute,
		arity:  1,
		params: []TypeSet{numberType},
		result: TypeNumber,
	},
	fnMin: {
		call:     minimum,
		arity:    1,
		variadic: true,
		params:   []TypeSet{numberType},
		result:   TypeNumber,
	},
	fnMax: {
		call:     maximum,
		arity:    1,
		variadic: true,
		params:   []TypeSet{numberType},
		result:   TypeNumber,
	},
	fnSum: {
		call:   sum,
		arity:  1,
		params: []TypeSet{arrayType},
		result: TypeNumber,
	},
	fnRound: {
		call:   round,
		arity:  1,
		params: []TypeSet{numberType},
		result: TypeNumber,
	},
	fnFloor: {
		call:   floor,
		arity:  1,
		params: []TypeSet{numberType},
		result: TypeNumber,
	},
	fnCeil: {
		call:   ceil,
		arity:  1,
		params: []TypeSet{numberType},
		result: TypeNumber,
	},
	fnNumber: {
		call:   toNumber,
		arity:  1,
		params: []TypeSet{stringType | numberType},
		result: TypeNumber,
	},
	fnString: {
		call:   toString,
		arity:  1,
		params: []TypeSet{textableType},
		result: TypeString,
	},
	fnStartsWith: {
		call:   startsWith,
		arity:  2,
		params: []TypeSet{stringType, stringType},
		result: TypeBool,
	},
	fnEndsWith: {
		call:   endsWith,
		arity:  2,
		params: []TypeSet{stringType, stringType},
		result: TypeBool,
	},
	fnIncludes: {
		call:   includes,
		arity:  2,
		params: []TypeSet{stringType, stringType},
		result: TypeBool,
	},
	fnLower: {
		call:   lower,
		arity:  1,
		params: []TypeSet{stringType},
		result: TypeString,
	},
	fnUpper: {
		call:   upper,
		arity:  1,
		params: []TypeSet{stringType},
		result: TypeString,
	},
	fnTrim: {
		call:   trim,
		arity:  1,
		params: []TypeSet{stringType},
		result: TypeString,
	},
	fnSplit: {
		call:   split,
		arity:  2,
		params: []TypeSet{stringType, stringType},
		result: TypeArray,
	},
	fnJoin: {
		call:   join,
		arity:  2,
		params: []TypeSet{arrayType, stringType},
		result: TypeString,
	},
	fnKeys: {
		call:   keys,
		arity:  1,
		params: []TypeSet{objectType},
		result: TypeArray,
	},
	fnValues: {
		call:   values,
		arity:  1,
		params: []TypeSet{objectType},
		result: TypeArray,
	},
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
		return nil, operandError(fnLength)
	}
}

func absolute(args ...any) (any, error) {
	n := number(args[0])
	if n-n != 0 {
		return nil, operandError(fnAbs)
	}
	return math.Abs(n), nil
}

func minimum(args ...any) (any, error) {
	result := number(args[0])
	if result-result != 0 {
		return nil, operandError(fnMin)
	}
	for _, arg := range args[1:] {
		n := number(arg)
		if n-n != 0 {
			return nil, operandError(fnMin)
		}
		result = min(result, n)
	}
	return result, nil
}

func maximum(args ...any) (any, error) {
	result := number(args[0])
	if result-result != 0 {
		return nil, operandError(fnMax)
	}
	for _, arg := range args[1:] {
		n := number(arg)
		if n-n != 0 {
			return nil, operandError(fnMax)
		}
		result = max(result, n)
	}
	return result, nil
}

func sum(args ...any) (any, error) {
	items, ok := args[0].([]any)
	if !ok {
		return nil, operandError(fnSum)
	}
	result := 0.0
	for _, item := range items {
		n := number(item)
		if n-n != 0 {
			return nil, operandError(fnSum)
		}
		result += n
		if math.IsNaN(result) || math.IsInf(result, 0) {
			return nil, fmt.Errorf("%w: %s", ErrArithmetic, fnSum)
		}
	}
	return result, nil
}

func round(args ...any) (any, error) {
	n := number(args[0])
	if n-n != 0 {
		return nil, operandError(fnRound)
	}
	return math.Round(n), nil
}

func floor(args ...any) (any, error) {
	n := number(args[0])
	if n-n != 0 {
		return nil, operandError(fnFloor)
	}
	return math.Floor(n), nil
}

func ceil(args ...any) (any, error) {
	n := number(args[0])
	if n-n != 0 {
		return nil, operandError(fnCeil)
	}
	return math.Ceil(n), nil
}

func toNumber(args ...any) (any, error) {
	value := args[0]
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if len(text) == 0 || !json.Valid([]byte(text)) {
			return nil, operandError(fnNumber)
		}
		n, err := strconv.ParseFloat(text, 64)
		if err != nil || n-n != 0 {
			return nil, operandError(fnNumber)
		}
		return n, nil
	}
	n := number(value)
	if n-n != 0 {
		return nil, operandError(fnNumber)
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
			return nil, operandError(fnString)
		}
		return strconv.FormatFloat(n, 'g', -1, 64), nil
	}
}

func startsWith(args ...any) (any, error) {
	text, ok := args[0].(string)
	if !ok {
		return nil, operandError(fnStartsWith)
	}
	prefix, ok := args[1].(string)
	if !ok {
		return nil, operandError(fnStartsWith)
	}
	return strings.HasPrefix(text, prefix), nil
}

func endsWith(args ...any) (any, error) {
	text, ok := args[0].(string)
	if !ok {
		return nil, operandError(fnEndsWith)
	}
	suffix, ok := args[1].(string)
	if !ok {
		return nil, operandError(fnEndsWith)
	}
	return strings.HasSuffix(text, suffix), nil
}

func includes(args ...any) (any, error) {
	text, ok := args[0].(string)
	if !ok {
		return nil, operandError(fnIncludes)
	}
	part, ok := args[1].(string)
	if !ok {
		return nil, operandError(fnIncludes)
	}
	return strings.Contains(text, part), nil
}

func lower(args ...any) (any, error) {
	if text, ok := args[0].(string); ok {
		return strings.ToLower(text), nil
	}
	return nil, operandError(fnLower)
}

func upper(args ...any) (any, error) {
	if text, ok := args[0].(string); ok {
		return strings.ToUpper(text), nil
	}
	return nil, operandError(fnUpper)
}

func trim(args ...any) (any, error) {
	if text, ok := args[0].(string); ok {
		return strings.TrimSpace(text), nil
	}
	return nil, operandError(fnTrim)
}

func split(args ...any) (any, error) {
	text, ok := args[0].(string)
	if !ok {
		return nil, operandError(fnSplit)
	}
	separator, ok := args[1].(string)
	if !ok {
		return nil, operandError(fnSplit)
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
		return nil, operandError(fnJoin)
	}
	separator, ok := args[1].(string)
	if !ok {
		return nil, operandError(fnJoin)
	}
	var result strings.Builder
	for i, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, operandError(fnJoin)
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
		return nil, operandError(fnKeys)
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
		return nil, operandError(fnValues)
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
