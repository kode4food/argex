package engine

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Function is a callable value. Arity is exact, or a minimum when Variadic.
// Params gives static types, repeating the last. A zero Result is unknown.
// Call must validate runtime types and must not change or retain its args
type Function struct {
	Call     func(...any) (any, error)
	Arity    int
	Variadic bool
	Params   []TypeSet
	Result   TypeTag
}

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

var builtins = map[string]any{
	fnLength: &Function{
		Call:   length,
		Arity:  1,
		Params: []TypeSet{lengthableType},
		Result: TypeNumber,
	},
	fnAbs: &Function{
		Call:   absolute,
		Arity:  1,
		Params: []TypeSet{numberType},
		Result: TypeNumber,
	},
	fnMin: &Function{
		Call:     minimum,
		Arity:    1,
		Variadic: true,
		Params:   []TypeSet{numberType},
		Result:   TypeNumber,
	},
	fnMax: &Function{
		Call:     maximum,
		Arity:    1,
		Variadic: true,
		Params:   []TypeSet{numberType},
		Result:   TypeNumber,
	},
	fnSum: &Function{
		Call:   sum,
		Arity:  1,
		Params: []TypeSet{arrayType},
		Result: TypeNumber,
	},
	fnRound: &Function{
		Call:   round,
		Arity:  1,
		Params: []TypeSet{numberType},
		Result: TypeNumber,
	},
	fnFloor: &Function{
		Call:   floor,
		Arity:  1,
		Params: []TypeSet{numberType},
		Result: TypeNumber,
	},
	fnCeil: &Function{
		Call:   ceil,
		Arity:  1,
		Params: []TypeSet{numberType},
		Result: TypeNumber,
	},
	fnNumber: &Function{
		Call:   toNumber,
		Arity:  1,
		Params: []TypeSet{stringType | numberType},
		Result: TypeNumber,
	},
	fnString: &Function{
		Call:   toString,
		Arity:  1,
		Params: []TypeSet{textableType},
		Result: TypeString,
	},
	fnStartsWith: &Function{
		Call:   startsWith,
		Arity:  2,
		Params: []TypeSet{stringType, stringType},
		Result: TypeBool,
	},
	fnEndsWith: &Function{
		Call:   endsWith,
		Arity:  2,
		Params: []TypeSet{stringType, stringType},
		Result: TypeBool,
	},
	fnIncludes: &Function{
		Call:   includes,
		Arity:  2,
		Params: []TypeSet{stringType, stringType},
		Result: TypeBool,
	},
	fnLower: &Function{
		Call:   lower,
		Arity:  1,
		Params: []TypeSet{stringType},
		Result: TypeString,
	},
	fnUpper: &Function{
		Call:   upper,
		Arity:  1,
		Params: []TypeSet{stringType},
		Result: TypeString,
	},
	fnTrim: &Function{
		Call:   trim,
		Arity:  1,
		Params: []TypeSet{stringType},
		Result: TypeString,
	},
	fnSplit: &Function{
		Call:   split,
		Arity:  2,
		Params: []TypeSet{stringType, stringType},
		Result: TypeArray,
	},
	fnJoin: &Function{
		Call:   join,
		Arity:  2,
		Params: []TypeSet{arrayType, stringType},
		Result: TypeString,
	},
	fnKeys: &Function{
		Call:   keys,
		Arity:  1,
		Params: []TypeSet{objectType},
		Result: TypeArray,
	},
	fnValues: &Function{
		Call:   values,
		Arity:  1,
		Params: []TypeSet{objectType},
		Result: TypeArray,
	},
}

var defaultEnv = &Env{entries: maps.Clone(builtins)}

// DefaultEnv returns the shared environment containing the builtin functions
func DefaultEnv() *Env {
	return defaultEnv
}

// MarshalJSON rejects functions, which JSON can't represent
func (*Function) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("%w: function", ErrType)
}

func (f *Function) accepts(arity int) bool {
	return arity >= f.Arity && (f.Variadic || arity == f.Arity)
}

func (f *Function) valid() bool {
	return f != nil && f.Call != nil && f.Arity >= 0
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
