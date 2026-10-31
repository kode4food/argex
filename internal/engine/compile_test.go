package engine_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

const (
	reuseWorkers    = 10
	reuseIterations = 10
	expressionDepth = 256
)

func TestSkippedCompilation(t *testing.T) {
	for _, source := range []string{
		"case { true: 1, unknown() }",
		"false && unknown()", "1 ?? unknown()", "null?.[unknown()]",
		"[] | map x: unknown()", "[] | filter x: unknown()",
		"[unknown()] | map x: x | filter y: y",
		"[] | map x: x | filter y: unknown()",
	} {
		_, err := argex.Eval(source, nil)
		assert.ErrorIs(t, err, argex.ErrFunction)
	}
}

func TestLimits(t *testing.T) {
	_, err := argex.Parse(
		strings.Repeat("(", expressionDepth) +
			"1" + strings.Repeat(")", expressionDepth),
	)
	assert.ErrorIs(t, err, argex.ErrLimit)
	_, err = argex.Parse(strings.Repeat("let x = 1: ", expressionDepth+1) + "1")
	assert.ErrorIs(t, err, argex.ErrLimit)
	expr, err := argex.Parse("1" + strings.Repeat("+1", expressionDepth+1))
	if !assert.NoError(t, err) {
		return
	}
	_, err = argex.DefaultEnv().Compile(expr)
	assert.ErrorIs(t, err, argex.ErrLimit)
}

func TestProgramEval(t *testing.T) {
	expr, err := argex.Parse("value + 1")
	if !assert.NoError(t, err) {
		return
	}
	program, err := argex.DefaultEnv().Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	value, err := program.Eval(map[string]any{"value": 2})
	assert.NoError(t, err)
	assert.Equal(t, float64(3), value)
}

func TestPreparationReuse(t *testing.T) {
	source := `case selector {
		1: {["result"]: case selector { 1: 10, 30 }},
		{["result"]: 20},
	}`
	expr, err := argex.Parse(source)
	if !assert.NoError(t, err) {
		return
	}
	base := argex.DefaultEnv()
	for _, tc := range []struct {
		selector float64
		want     float64
	}{
		{selector: 1, want: 10},
		{selector: 2, want: 20},
		{selector: 1, want: 10},
	} {
		env, err := base.Bind("selector", tc.selector)
		if !assert.NoError(t, err) {
			return
		}
		program, err := env.Compile(expr)
		if !assert.NoError(t, err) {
			return
		}
		assert.True(t, program.StaticType().Is(argex.TypeObject))
		value, err := program.Eval(nil)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"result": tc.want}, value)
		assert.Equal(t, []string{"selector"}, base.Names(expr))
		assert.Empty(t, env.Names(expr))
	}
}

func TestReuse(t *testing.T) {
	expr, err := argex.Parse("(items | filter x: x > 1) | map x: x + n")
	if !assert.NoError(t, err) {
		return
	}
	program, err := argex.DefaultEnv().Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	variables := map[string]any{"items": []any{1, 2}, "n": 3}
	var workers sync.WaitGroup
	for range reuseWorkers {
		workers.Go(func() {
			for range reuseIterations {
				result, err := program.Eval(variables)
				if !assert.NoError(t, err) {
					return
				}
				assert.Equal(t, []any{float64(5)}, result)
				result.([]any)[0] = "changed"
			}
		})
	}
	workers.Wait()
	assert.Equal(t, []any{1, 2}, variables["items"])
	second, err := argex.DefaultEnv().Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	value, err := second.Eval(map[string]any{"items": []any{5}, "n": 1})
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(6)}, value)
}
