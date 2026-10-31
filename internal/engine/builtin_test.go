package engine_test

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

func TestFunctionNumericInputs(t *testing.T) {
	for _, value := range []any{2, float64(2), json.Number("2")} {
		variables := map[string]any{"value": value}
		for _, source := range []string{
			"number(value)", "sum([value])", "round(value)",
			"floor(value)", "ceil(value)",
		} {
			result, err := argex.Eval(source, variables)
			assert.NoError(t, err)
			assert.Equal(t, float64(2), result)
		}
		result, err := argex.Eval("string(value)", variables)
		assert.NoError(t, err)
		assert.Equal(t, "2", result)
	}
	for _, value := range []any{
		math.NaN(), math.Inf(1), math.Inf(-1), json.Number("bad"),
	} {
		for _, source := range []string{
			"number(value)", "string(value)", "sum([value])",
			"round(value)", "floor(value)", "ceil(value)",
		} {
			_, err := argex.Eval(source, map[string]any{"value": value})
			assert.ErrorIs(t, err, argex.ErrType)
		}
	}
}

func TestCustomFunctions(t *testing.T) {
	fn := &argex.Function{
		Call:     echo,
		Arity:    1,
		Variadic: true,
		Params:   []argex.TypeSet{argex.TypeSet(argex.TypeNumber)},
		Result:   argex.TypeArray,
	}
	ns, err := argex.NewEnv().Bind("echo", fn)
	if !assert.NoError(t, err) {
		return
	}
	for _, source := range []string{
		"echo(1,2)", "echo(...[1,2])", "let f=echo:f(1,2)",
		"let f=echo:f(...[1,2])", "[echo][0](1,2)",
		"[echo][0](...[1,2])", "{f:echo}.f(1,2)",
	} {
		assert.Equal(t,
			[]any{float64(1), float64(2)}, envEval(t, ns, source),
		)
	}
	expr, err := argex.Parse("echo(1,2)")
	if !assert.NoError(t, err) {
		return
	}
	program, err := ns.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, program.StaticType().Is(argex.TypeArray))
	for _, source := range []string{"echo(true)", "echo(1,true)"} {
		expr, err := argex.Parse(source)
		if !assert.NoError(t, err) {
			return
		}
		_, err = ns.Compile(expr)
		assert.ErrorIs(t, err, argex.ErrType)
	}
}

func TestDynamicFunctionErrors(t *testing.T) {
	fn := &argex.Function{Call: func(...any) (any, error) {
		return nil, fmt.Errorf("%w: callback", argex.ErrArithmetic)
	}}
	ns, err := argex.NewEnv().Bind("fail", fn)
	if !assert.NoError(t, err) {
		return
	}
	for _, source := range []string{
		"fail()", "fail(...[])", "let f=fail:f()", "let f=fail:f(...[])",
	} {
		expr, err := argex.Parse(source)
		if !assert.NoError(t, err) {
			return
		}
		program, err := ns.Compile(expr)
		if !assert.NoError(t, err) {
			return
		}
		_, err = program.Eval(nil)
		assert.ErrorIs(t, err, argex.ErrArithmetic)
	}
	for _, source := range []string{"this.f()", "this.f(...[])"} {
		expr, err := argex.Parse(source)
		if !assert.NoError(t, err) {
			return
		}
		program, err := ns.Compile(expr)
		if !assert.NoError(t, err) {
			return
		}
		for _, invalid := range []*argex.Function{
			nil, {}, {Call: echo, Arity: -1},
		} {
			_, err = program.Eval(map[string]any{"f": invalid})
			assert.ErrorIs(t, err, argex.ErrFunction)
		}
	}
}

func TestFunctionEvaluationOrder(t *testing.T) {
	var calls []any
	record := &argex.Function{
		Call: func(args ...any) (any, error) {
			calls = append(calls, args[0])
			return args[0], nil
		},
		Arity: 1,
	}
	ns, err := argex.NewEnv().Bind("record", record)
	if !assert.NoError(t, err) {
		return
	}
	factory := &argex.Function{Call: func(...any) (any, error) {
		calls = append(calls, "callee")
		return &argex.Function{Call: echo, Variadic: true}, nil
	}}
	ns, err = ns.Bind("factory", factory)
	if !assert.NoError(t, err) {
		return
	}
	for _, source := range []string{
		"factory()(record(1), record(2))",
		"factory()(record(1), ...[record(2)])",
	} {
		calls = nil
		value := envEval(t, ns, source)
		assert.Equal(t, []any{float64(1), float64(2)}, value)
		assert.Equal(t, []any{"callee", float64(1), float64(2)}, calls)
	}
	for _, source := range []string{
		"false && record(true)", "true || record(false)",
		"1 ?? record(2)", "case {true: 1, record(2)}",
	} {
		calls = nil
		envEval(t, ns, source)
		assert.Empty(t, calls)
	}
}

func TestFunctionValues(t *testing.T) {
	ns := argex.DefaultEnv()
	assert.Same(t, ns, argex.DefaultEnv())
	value := envEval(t, ns, "length")
	_, ok := value.(*argex.Function)
	assert.True(t, ok)
	_, err := json.Marshal(value)
	assert.ErrorIs(t, err, argex.ErrType)
	assert.Equal(t, true, envEval(t, ns, "length == length"))
	assert.Equal(t, false, envEval(t, ns, "length == abs"))
	value = envEval(t, ns, "([length,sum] | map f: f([]) ?? 0)[0] == 0")
	assert.Equal(t, true, value)
}

func TestDynamicCallStack(t *testing.T) {
	args := strings.Repeat("1,", expressionDepth/4)
	source := "let f=max: [f(" + args + "), f(...[" + args + "])]"
	value, err := argex.Eval(source, nil)
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(1), float64(1)}, value)
	value, err = argex.Eval("let f=length: [f([]),f([1]),f([1,2])]", nil)
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(0), float64(1), float64(2)}, value)
}
