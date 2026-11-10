package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argex"
)

func TestBindingPatterns(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   any
	}{
		{
			name: "object",
			source: `let {name, age: years} = {name: "Ada", age: 36}:
				[name, years]`,
			want: []any{"Ada", float64(36)},
		},
		{
			name:   "array",
			source: "let [first, _, third,] = [1, 2, 3, 4]: first + third",
			want:   float64(4),
		},
		{
			name:   "elision",
			source: "let [, second,, fourth] = [1, 2, 3, 4]: second + fourth",
			want:   float64(6),
		},
		{
			name:   "discard missing",
			source: "let {absent: _} = {}, [_,] = []: 1",
			want:   float64(1),
		},
		{
			name: "nested",
			source: `let {user: {name}, points: [x, [y]]} =
				{user: {name: "Ada"}, points: [2, [3]]}: [name, x + y]`,
			want: []any{"Ada", float64(5)},
		},
		{
			name:   "sequential",
			source: "let [x] = [2], {y} = {y: x * 3}, x = y + 1: x",
			want:   float64(7),
		},
		{
			name:   "null member",
			source: "let {x} = {x: null}, [y] = [null]: [x, y]",
			want:   []any{nil, nil},
		},
		{
			name:   "fallback source",
			source: "let {x} = absent ?? {x: 7}: x",
			want:   float64(7),
		},
		{name: "empty", source: "let {} = {}, [] = []: 1", want: float64(1)},
		{
			name: "reserved key",
			source: `let {this: x, "strange-key": y,} =
				{this: 1, "strange-key": 2}: x + y`,
			want: float64(3),
		},
		{
			name:   "shadow builtin",
			source: "let {length} = {length: abs}: length(-3)",
			want:   float64(3),
		},
		{
			name:   "outer source",
			source: "let x = [3]: (let [x] = x: x) + x[0]",
			want:   float64(6),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := argex.Eval(tc.source, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestPatternDefaults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   any
	}{
		{name: "object", source: "let {x = 3} = {}: x", want: float64(3)},
		{name: "array", source: "let [x = 3] = []: x", want: float64(3)},
		{
			name:   "coalesce preserves null",
			source: "let {x = 3} = {x: null}: x ?? 7",
		},
		{
			name:   "rename",
			source: "let {x: renamed = 3} = {}: renamed",
			want:   float64(3),
		},
		{
			name: "preserve array values",
			source: `let [a = 1, b = 1, c = 1, d = 1] =
				[null, false, 0, ""]: [a, b, c, d]`,
			want: []any{nil, false, float64(0), ""},
		},
		{
			name:   "earlier binding",
			source: "let {x = 2, y = x + 1} = {}: y",
			want:   float64(3),
		},
		{
			name:   "outer binding",
			source: "let x = 7, {x = x + 1} = {}: x",
			want:   float64(8),
		},
		{
			name:   "future binding",
			source: "let y = 7, [x = y, y = 1] = []: [x, y]",
			want:   []any{float64(7), float64(1)},
		},
		{
			name:   "nested object",
			source: "let {user: {name = 2} = {}} = {}: name",
			want:   float64(2),
		},
		{
			name:   "nested array",
			source: "let [[x = 2] = []] = []: x",
			want:   float64(2),
		},
		{
			name:   "explicit missing",
			source: "let {x = 2} = {x: absent}: x",
			want:   float64(2),
		},
		{
			name:   "discard default",
			source: "let {x: _ = 2} = {}: 1",
			want:   float64(1),
		},
		{
			name:   "earlier function",
			source: "let {f = abs, x = f(-3)} = {}: x",
			want:   float64(3),
		},
		{
			name:   "case expression",
			source: "let [x = case { true: 2, 3 }] = []: x",
			want:   float64(2),
		},
		{
			name:   "pipeline expression",
			source: "let [x = ([1, 2] | map n: n * 2)] = []: x",
			want:   []any{float64(2), float64(4)},
		},
		{
			name:   "pipeline key",
			source: "[{}, {}] | map i, {x = i, y = x + 1}: [x, y]",
			want:   []any{[]any{float64(0), float64(1)}, []any{float64(1), float64(2)}},
		},
		{
			name:   "earlier stage",
			source: "[{x: 3}] | map {x}: {} | map {y = x}: y",
			want:   []any{float64(3)},
		},
		{
			name:   "pipeline inside default",
			source: `[{}, {}] | map i, {x = ([i] | map n: n + 1)}: x`,
			want:   []any{[]any{float64(1)}, []any{float64(2)}},
		},
		{
			name:   "stage shadows earlier",
			source: "[{x: 3}] | map {x}: {} | map {x = x + 1}: x",
			want:   []any{float64(4)},
		},
		{
			name:   "array pipeline",
			source: "[[], [2], [null]] | map [x = 3]: x",
			want:   []any{float64(3), float64(2), nil},
		},
		{
			name:   "filter keeps object",
			source: "[{}, {x: 2}] | filter {x = 0}: x > 1",
			want:   []any{map[string]any{"x": float64(2)}},
		},
		{
			name:   "any default",
			source: "[{}, {x: true}, null] | any {x = false}: x",
			want:   true,
		},
		{
			name:   "all default",
			source: "[{}, null] | all {x = false}: x",
			want:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := argex.Eval(tc.source, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestPatternDefaultEffects(t *testing.T) {
	calls := 0
	env, err := argex.DefaultEnv().Bind("next", &argex.Function{
		Call: func(...any) (any, error) {
			calls++
			return float64(calls), nil
		},
	})
	if !assert.NoError(t, err) {
		return
	}
	expr, err := argex.Parse("let {x = next(), y = next()} = this: [x, y]")
	if !assert.NoError(t, err) {
		return
	}
	program, err := env.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	assert.Zero(t, calls)
	for _, tc := range []struct {
		name  string
		input map[string]any
		want  []any
		calls int
	}{
		{
			name:  "null and false",
			input: map[string]any{"x": nil, "y": false},
			want:  []any{nil, false},
		},
		{
			name:  "zero and empty",
			input: map[string]any{"x": 0, "y": ""},
			want:  []any{0, ""},
		},
		{
			name:  "both missing",
			input: map[string]any{},
			want:  []any{float64(1), float64(2)},
			calls: 2,
		},
		{
			name:  "one missing",
			input: map[string]any{"x": 10},
			want:  []any{10, float64(3)},
			calls: 3,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := program.Eval(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
			assert.Equal(t, tc.calls, calls)
		})
	}
}

func TestPatternDefaultErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   error
	}{
		{
			name:   "required field",
			source: "let {x = 1, y} = {}: x",
			want:   argex.ErrMissing,
		},
		{
			name:   "required element",
			source: "let [x = 1, y] = []: x",
			want:   argex.ErrMissing,
		},
		{
			name:   "fallback error",
			source: "let {x = 1 / 0} = {}: x",
			want:   argex.ErrArithmetic,
		},
		{
			name:   "empty fallback",
			source: "let {x: [y] = []} = {}: y",
			want:   argex.ErrMissing,
		},
		{
			name:   "wrong fallback type",
			source: "let {x: [y] = 1} = {}: y",
			want:   argex.ErrType,
		},
		{
			name:   "null container",
			source: "let {x: [y] = [1]} = {x: null}: y",
			want:   argex.ErrMissing,
		},
		{
			name:   "null pipeline value",
			source: "[null] | map {x = 1}: x",
			want:   argex.ErrMissing,
		},
		{
			name:   "wrong pipeline type",
			source: "[1] | map {x = 1}: x",
			want:   argex.ErrType,
		},
		{
			name:   "required pipeline field",
			source: "[{}] | map {x = 1, y}: x",
			want:   argex.ErrMissing,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := argex.Eval(tc.source, nil)
			assert.ErrorIs(t, err, tc.want)
		})
	}
	value, err := argex.Eval("let {x = 1 / 0} = {x: null}: x", nil)
	assert.NoError(t, err)
	assert.Nil(t, value)
}

func TestPipelinePatterns(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   any
	}{
		{
			name:   "object",
			source: "[{x: 1}, {x: 2}] | map {x}: x * 2",
			want:   []any{float64(2), float64(4)},
		},
		{
			name:   "array",
			source: "[[1, 2], [3, 4]] | map i, [x, y]: i + x + y",
			want:   []any{float64(3), float64(8)},
		},
		{
			name:   "object keys",
			source: `{a: {x: 1}, b: {x: 2}} | map key, {x}: {key, x}`,
			want: []any{
				map[string]any{"key": "a", "x": float64(1)},
				map[string]any{"key": "b", "x": float64(2)},
			},
		},
		{
			name:   "preserve filter values",
			source: "[{x: 1}, {x: 2}] | filter {x}: x > 1",
			want:   []any{map[string]any{"x": float64(2)}},
		},
		{
			name:   "earlier stage",
			source: "[{x: 1}, {x: 2}] | map {x}: x * 2 | map x + 1",
			want:   []any{float64(2), float64(3)},
		},
		{
			name: "later indexes",
			source: `[{x: 0}, {x: 2}, {x: 3}] | filter {x}: x > 0
				| map i, {x}: x * 10 | map j, value: [i, j, x, value]`,
			want: []any{
				[]any{float64(0), float64(0), float64(2), float64(20)},
				[]any{float64(1), float64(1), float64(3), float64(30)},
			},
		},
		{
			name:   "shadow stage",
			source: "[[1], [2]] | map [x]: [x * 2] | map [x]: x",
			want:   []any{float64(2), float64(4)},
		},
		{
			name:   "nested pipeline",
			source: "[{x: 2}] | map {x}: ([[3]] | map [y]: x + y)",
			want:   []any{[]any{float64(5)}},
		},
		{
			name:   "any short circuit",
			source: "[{x: true}, {}] | any {x}: x",
			want:   true,
		},
		{
			name:   "all short circuit",
			source: "[[false], []] | all [x]: x",
			want:   false,
		},
		{name: "empty collection", source: "[] | map {x}: x", want: []any{}},
		{
			name: "filter before destructuring",
			source: `[null, {x: 2}] | filter row: row != null
				| map {x}: x`,
			want: []any{float64(2)},
		},
		{
			name:   "reduce element",
			source: "[{x: 1}, {x: 2}] | reduce t = 0, {x}: t + x",
			want:   float64(3),
		},
		{
			name: "reduce object accumulator",
			source: `[1, 2, 3] | reduce {sum, n} = {sum: 0, n: 0}, x:
				{sum: sum + x, n: n + 1}`,
			want: map[string]any{"sum": float64(6), "n": float64(3)},
		},
		{
			name:   "reduce array accumulator",
			source: "[1, 2] | reduce [a, b] = [0, 0], x: [b, a + x]",
			want:   []any{float64(1), float64(2)},
		},
		{
			name:   "reduce accumulator default",
			source: "[1, 2] | reduce {t = 10} = {}, x: {t: t + x}",
			want:   map[string]any{"t": float64(13)},
		},
		{
			name:   "reduce accumulator sees earlier stages",
			source: "[1, 2] | map x: x | reduce {t = x} = {}: {t: t + x}",
			want:   map[string]any{"t": float64(4)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := argex.Eval(tc.source, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestPatternErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   error
	}{
		{name: "field", source: "let {x} = {}: x", want: argex.ErrMissing},
		{name: "index", source: "let [x] = []: x", want: argex.ErrMissing},
		{
			name:   "missing element",
			source: "let [x] = [absent]: x",
			want:   argex.ErrMissing,
		},
		{
			name:   "pipeline missing element",
			source: "[[absent]] | map [x]: x",
			want:   argex.ErrMissing,
		},
		{
			name:   "unused binding",
			source: "let {x} = {}: 1",
			want:   argex.ErrMissing,
		},
		{
			name:   "body fallback",
			source: "let {x} = {}: x ?? 1",
			want:   argex.ErrMissing,
		},
		{
			name:   "null nested source",
			source: "let {x: {y}} = {x: null}: y",
			want:   argex.ErrMissing,
		},
		{
			name:   "wrong nested type",
			source: "let {x: [y]} = {x: {}}: y",
			want:   argex.ErrType,
		},
		{
			name:   "runtime object type",
			source: "let {} = this: 1",
			want:   argex.ErrType,
		},
		{
			name:   "runtime array type",
			source: "let [] = this: 1",
			want:   argex.ErrType,
		},
		{
			name:   "runtime null source",
			source: "let {} = absent: 1",
			want:   argex.ErrMissing,
		},
		{
			name:   "pipeline field",
			source: "[{}] | map {x}: x",
			want:   argex.ErrMissing,
		},
		{
			name:   "pipeline null",
			source: "[null] | map []: 1",
			want:   argex.ErrMissing,
		},
		{
			name:   "pipeline shape",
			source: "[{}] | map [x]: x",
			want:   argex.ErrType,
		},
		{
			name:   "filter predicate",
			source: "[[1]] | filter [x]: x",
			want:   argex.ErrType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := argex.Parse(tc.source)
			if !assert.NoError(t, err) {
				return
			}
			program, err := argex.DefaultEnv().Compile(expr)
			if !assert.NoError(t, err) {
				return
			}
			_, err = program.Eval(1)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPatternSourceOnce(t *testing.T) {
	calls := 0
	env, err := argex.DefaultEnv().Bind("source", &argex.Function{
		Call: func(...any) (any, error) {
			calls++
			return map[string]any{"x": []any{1, 2}}, nil
		},
	})
	if !assert.NoError(t, err) {
		return
	}
	expr, err := argex.Parse("let {x: [a, b]} = source(): a + b")
	if !assert.NoError(t, err) {
		return
	}
	program, err := env.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	assert.Zero(t, calls)
	value, err := program.Eval(nil)
	assert.NoError(t, err)
	assert.Equal(t, float64(3), value)
	assert.Equal(t, 1, calls)
}

func TestRequiredPatternValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		input  any
	}{
		{
			name:   "object member",
			source: "let {x} = this: x",
			input:  map[string]any{"x": argex.Missing{}},
		},
		{
			name:   "array element",
			source: "let [x] = this: x",
			input:  []any{argex.Missing{}},
		},
		{
			name:   "pipeline object member",
			source: "[this] | map {x}: x",
			input:  map[string]any{"x": argex.Missing{}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := argex.Eval(tc.source, tc.input)
			assert.ErrorIs(t, err, argex.ErrMissing)
		})
	}
}

func TestPatternAssertionTiming(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{
			name:   "mixed source types",
			source: "let [] = case { valid: [], null }: 1",
		},
		{
			name: "nested member",
			source: "case { valid: 1, " +
				"(let {x: [y]} = {x: null}: y) }",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := argex.Parse(tc.source)
			if !assert.NoError(t, err) {
				return
			}
			program, err := argex.DefaultEnv().Compile(expr)
			if !assert.NoError(t, err) {
				return
			}
			value, err := program.Eval(map[string]any{"valid": true})
			assert.NoError(t, err)
			assert.Equal(t, float64(1), value)
			_, err = program.Eval(map[string]any{"valid": false})
			assert.ErrorIs(t, err, argex.ErrMissing)
		})
	}
}

func TestPatternSourceTypes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{name: "spread array", source: "let [_, x] = [...[0, absent], 3]: x"},
		{
			name:   "spread object",
			source: "let {x: []} = {x: [], ...{x: null}}: 1",
		},
		{
			name:   "computed key",
			source: "let {x: []} = {x: [], [key]: null}: 1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := argex.Eval(tc.source, map[string]any{"key": "x"})
			assert.ErrorIs(t, err, argex.ErrMissing)
		})
	}
	expr, err := argex.Parse("let {x: [y] = [2]} = {x: source}: y")
	if !assert.NoError(t, err) {
		return
	}
	for _, tc := range []struct {
		name  string
		value any
		want  any
		err   error
	}{
		{name: "missing", value: argex.Missing{}, want: float64(2)},
		{name: "null", err: argex.ErrMissing},
		{name: "array", value: []any{3}, want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, err := argex.DefaultEnv().Bind("source", tc.value)
			if !assert.NoError(t, err) {
				return
			}
			program, err := env.Compile(expr)
			if !assert.NoError(t, err) {
				return
			}
			value, err := program.Eval(nil)
			assert.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestPatternReuse(t *testing.T) {
	expr, err := argex.Parse("let {items: [x = offset] = []} = source: x + offset")
	if !assert.NoError(t, err) {
		return
	}
	for _, offset := range []int{1, 2, 3} {
		env, err := argex.DefaultEnv().Bind("offset", offset)
		if !assert.NoError(t, err) {
			return
		}
		t.Run("compile", func(t *testing.T) {
			t.Parallel()
			for range reuseIterations {
				program, err := env.Compile(expr)
				if !assert.NoError(t, err) {
					return
				}
				value, err := program.Eval(map[string]any{
					"source": map[string]any{"items": []any{10}},
				})
				assert.NoError(t, err)
				assert.Equal(t, float64(10+offset), value)
				value, err = program.Eval(map[string]any{
					"source": map[string]any{},
				})
				assert.NoError(t, err)
				assert.Equal(t, float64(2*offset), value)
				assert.Equal(t, []string{"source"}, env.Names(expr))
			}
		})
	}
}
