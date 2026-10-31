package engine_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

type envValueCase struct {
	value any
	want  argex.TypeTag
}

func TestEnvEmpty(t *testing.T) {
	ns := argex.NewEnv()
	assert.True(t, argex.IsMissing(envEval(t, ns, "length")))
	expr, err := argex.Parse("length([])")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"length"}, ns.Names(expr))
	_, err = ns.Compile(expr)
	assert.ErrorIs(t, err, argex.ErrFunction)
}

func TestEnvBindings(t *testing.T) {
	root := argex.DefaultEnv()
	child, err := root.Bind("limit", 10)
	if !assert.NoError(t, err) {
		return
	}
	sibling, err := root.Bind("limit", 20)
	if !assert.NoError(t, err) {
		return
	}
	grandchild, err := child.Bind("length", 7)
	if !assert.NoError(t, err) {
		return
	}
	rebound, err := child.Bind("limit", 30)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 30, envEval(t, rebound, "limit"))
	assert.Equal(t, 10, envEval(t, child, "limit"))
	assert.Equal(t, 20, envEval(t, sibling, "limit"))
	assert.Equal(t, 10, envEval(t, grandchild, "limit"))
	assert.Equal(t, 7, envEval(t, grandchild, "length"))
	assert.Equal(t, float64(2), envEval(t, root, "length([1,2])"))
	assert.Equal(t, float64(2), envEval(t, child, "length([1,2])"))
	assert.True(t, argex.IsMissing(envEval(t, root, "limit")))
	assert.Equal(t, float64(3), envEval(t, child, "let limit=3:limit"))
}

func TestEnvValueTypes(t *testing.T) {
	cases := map[string]envValueCase{
		"null":    {value: nil, want: argex.TypeNull},
		"missing": {value: argex.Missing{}, want: argex.TypeMissing},
		"boolean": {value: true, want: argex.TypeBool},
		"int":     {value: 1, want: argex.TypeNumber},
		"number":  {value: json.Number("1"), want: argex.TypeNumber},
		"string":  {value: "a", want: argex.TypeString},
		"array":   {value: []any{1}, want: argex.TypeArray},
		"object":  {value: map[string]any{"a": 1}, want: argex.TypeObject},
		"function": {
			value: &argex.Function{Call: echo},
			want:  argex.TypeFunction,
		},
		"unknown": {value: int64(1), want: argex.TypeAny},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ns, err := argex.NewEnv().Bind("bound", tc.value)
			if !assert.NoError(t, err) {
				return
			}
			expr, err := argex.Parse("bound")
			if !assert.NoError(t, err) {
				return
			}
			program, err := ns.Compile(expr)
			if !assert.NoError(t, err) {
				return
			}
			assert.True(t, program.StaticType().Is(tc.want))
			value, err := program.Eval(nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.value, value)
		})
	}
}

func TestEnvInvalidBindings(t *testing.T) {
	ns := argex.NewEnv()
	for _, fn := range []*argex.Function{
		nil, {}, {Call: echo, Arity: -1},
	} {
		child, err := ns.Bind("f", fn)
		assert.Nil(t, child)
		assert.ErrorIs(t, err, argex.ErrFunction)
	}
	var absent *argex.Env
	_, err := absent.Bind("f", 1)
	assert.ErrorIs(t, err, argex.ErrFunction)
	for _, name := range []string{"in", "λ", "_", "λ2", "a١", "_x"} {
		child, err := ns.Bind(name, 1)
		assert.NoError(t, err)
		assert.Equal(t, 1, envEval(t, child, name))
	}
}

func TestEnvBindingNames(t *testing.T) {
	for _, name := range []string{"", "true", "a.b"} {
		t.Run(name, func(t *testing.T) {
			child, err := argex.NewEnv().Bind(name, false)
			assert.NoError(t, err)
			assert.NotNil(t, child)
		})
	}
}

func TestEnvCompileResolution(t *testing.T) {
	expr, err := argex.Parse("custom(1)")
	if !assert.NoError(t, err) {
		return
	}
	root := argex.NewEnv()
	_, err = root.Compile(expr)
	assert.ErrorIs(t, err, argex.ErrFunction)
	child, err := root.Bind("custom", &argex.Function{Call: echo, Arity: 1})
	if !assert.NoError(t, err) {
		return
	}
	program, err := child.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	value, err := program.Eval(map[string]any{"custom": false})
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(1)}, value)
	_, err = root.Compile(expr)
	assert.ErrorIs(t, err, argex.ErrFunction)
}

func TestEnvResolutionIsolation(t *testing.T) {
	root := argex.NewEnv()
	first, err := root.Bind("answer", 1)
	if !assert.NoError(t, err) {
		return
	}
	second, err := root.Bind("answer", "two")
	if !assert.NoError(t, err) {
		return
	}
	expr, err := argex.Parse("answer")
	if !assert.NoError(t, err) {
		return
	}
	var workers sync.WaitGroup
	for _, ns := range []*argex.Env{first, second, root} {
		workers.Go(func() {
			for range reuseIterations {
				program, err := ns.Compile(expr)
				if !assert.NoError(t, err) {
					return
				}
				value, err := program.Eval(map[string]any{"answer": false})
				assert.NoError(t, err)
				resultType := program.StaticType()
				switch ns {
				case first:
					assert.Equal(t, 1, value)
					assert.True(t, resultType.Is(argex.TypeNumber))
					assert.Empty(t, ns.Names(expr))

				case second:
					assert.Equal(t, "two", value)
					assert.True(t, resultType.Is(argex.TypeString))
					assert.Empty(t, ns.Names(expr))

				default:
					assert.Equal(t, false, value)
					assert.True(t, resultType.Is(argex.TypeAny))
					assert.Equal(t, []string{"answer"}, ns.Names(expr))
				}
			}
		})
	}
	workers.Wait()
	var absent *argex.Env
	assert.Empty(t, absent.Names(expr))
	_, err = absent.Compile(expr)
	assert.ErrorIs(t, err, argex.ErrFunction)
}

func TestEnvShadowsInput(t *testing.T) {
	ns, err := argex.DefaultEnv().Bind("limit", 10)
	if !assert.NoError(t, err) {
		return
	}
	expr, err := argex.Parse("[limit, this.limit, length([1])]")
	if !assert.NoError(t, err) {
		return
	}
	assert.Empty(t, ns.Names(expr))
	program, err := ns.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	value, err := program.Eval(map[string]any{"limit": 20, "length": 0})
	assert.NoError(t, err)
	assert.Equal(t, []any{10, 20, float64(1)}, value)
}

func TestEnvMissingAccess(t *testing.T) {
	ns, err := argex.NewEnv().Bind("bound", argex.Missing{})
	if !assert.NoError(t, err) {
		return
	}
	for _, source := range []string{
		"bound.field", "bound[0]", "let x=bound:x.field",
		"(case { flag: bound, null }).field",
	} {
		t.Run(source, func(t *testing.T) {
			expr, err := argex.Parse(source)
			if !assert.NoError(t, err) {
				return
			}
			_, err = ns.Compile(expr)
			assert.ErrorIs(t, err, argex.ErrMissing)
		})
	}
	for _, source := range []string{"bound?.field", "bound?.[1/0]"} {
		t.Run(source, func(t *testing.T) {
			assert.True(t, argex.IsMissing(envEval(t, ns, source)))
		})
	}
}

func TestEnvNonCallableGlobals(t *testing.T) {
	cases := map[string]any{
		"integer":  int64(1),
		"callback": echo,
		"function": argex.Function{Call: echo},
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			ns, err := argex.NewEnv().Bind("f", value)
			if !assert.NoError(t, err) {
				return
			}
			for _, source := range []string{
				"f()", "f(...[])", "(f)()", "false && f()",
			} {
				expr, err := argex.Parse(source)
				if !assert.NoError(t, err) {
					return
				}
				_, err = ns.Compile(expr)
				assert.ErrorIs(t, err, argex.ErrType)
			}
			expr, err := argex.Parse("[f][0]()")
			if !assert.NoError(t, err) {
				return
			}
			program, err := ns.Compile(expr)
			if !assert.NoError(t, err) {
				return
			}
			_, err = program.Eval(nil)
			assert.ErrorIs(t, err, argex.ErrType)
		})
	}
}

func envEval(t *testing.T, ns *argex.Env, source string) any {
	t.Helper()
	expr, err := argex.Parse(source)
	if !assert.NoError(t, err) {
		return nil
	}
	program, err := ns.Compile(expr)
	if !assert.NoError(t, err) {
		return nil
	}
	value, err := program.Eval(nil)
	assert.NoError(t, err)
	return value
}

func echo(args ...any) (any, error) {
	return append([]any{}, args...), nil
}
