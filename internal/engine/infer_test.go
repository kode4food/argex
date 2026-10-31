package engine_test

import (
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

func TestStaticType(t *testing.T) {
	cases := map[string]struct {
		src  string
		want string
	}{
		"case with all-bool arms": {
			src:  `case { str == "hello": true, false }`,
			want: "bool",
		},
		"case with mixed arms": {
			src:  `case { str == "hello": true, -1 }`,
			want: "bool|number",
		},
		"equality is always bool": {src: `str == "hello"`, want: "bool"},
		"bare name is any":        {src: `value.status`, want: "any"},
		"optional access adds null": {
			src:  `value?.status`,
			want: "missing|any",
		},
		"typed builtin result": {src: `length(value)`, want: "number"},
		"all is always bool": {
			src:  `checks | all check: check.passed`,
			want: "bool",
		},
		"add two numbers":          {src: `1 + 2`, want: "number"},
		"add two strings":          {src: `"a" + "b"`, want: "string"},
		"add an unknown left side": {src: `value + 1`, want: "number|string"},
		"coalesce with unknown":    {src: `a ?? "x"`, want: "string|any"},
		"array literal":            {src: `[1, 2, 3]`, want: "array"},
		"object literal":           {src: `{a: 1}`, want: "object"},
		"map is always array": {
			src:  `items | map v: v.x`,
			want: "array",
		},
		"let body type":          {src: `let x = 1: [x]`, want: "array"},
		"let name carries type":  {src: `let s = "a": s`, want: "string"},
		"later let sees earlier": {src: `let x = 1, y = x: y`, want: "number"},
		"stage shadows let": {
			src:  `let x = 1: (xs | map x: x)[0]`,
			want: "any",
		},
		"unbound name stays any": {src: `let x = 1: y`, want: "any"},
		"subject case arms": {
			src:  `case status { "a": 1, "b": "x", null }`,
			want: "number|string|null",
		},
		"global function":    {src: `length`, want: "function"},
		"let shadows global": {src: `let length = 1: length`, want: "number"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			expr, err := argex.Parse(tc.src)
			if !assert.NoError(t, err) {
				return
			}
			program, err := argex.DefaultEnv().Compile(expr)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, tc.want, program.StaticType().String())
		})
	}
}

func TestStaticTypeKnown(t *testing.T) {
	expr, err := argex.Parse(`str == "hello"`)
	if !assert.NoError(t, err) {
		return
	}
	program, err := argex.DefaultEnv().Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	result := program.StaticType()
	assert.True(t, result.Known())
	assert.True(t, result.Is(argex.TypeBool))

	expr, err = argex.Parse(`value.status`)
	if !assert.NoError(t, err) {
		return
	}
	program, err = argex.DefaultEnv().Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	assert.False(t, program.StaticType().Known())
}

func TestStaticTypeEnv(t *testing.T) {
	ns := argex.NewEnv()
	ns, err := ns.Bind("limit", 10.0)
	if !assert.NoError(t, err) {
		return
	}
	expr, err := argex.Parse(`limit`)
	if !assert.NoError(t, err) {
		return
	}
	program, err := ns.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, program.StaticType().Is(argex.TypeNumber))
	expr, err = argex.Parse(`limit + "x"`)
	if !assert.NoError(t, err) {
		return
	}
	_, err = ns.Compile(expr)
	assert.ErrorIs(t, err, argex.ErrType)
}

func TestCompileRejectsStaticMismatches(t *testing.T) {
	cases := map[string]string{
		"length of a number":       `length(-1)`,
		"bool plus number":         `true + 1`,
		"string plus number":       `"a" + 1`,
		"number plus string":       `1 + "a"`,
		"not a number":             `!1`,
		"negate a string":          `-"x"`,
		"number less than string":  `1 < "a"`,
		"string less than number":  `"a" < 1`,
		"and with a number":        `true && 1`,
		"number and a bool":        `1 && true`,
		"in with a string":         `1 in "x"`,
		"dot access on a bool":     `true.x`,
		"index an object by name":  `{a: 1}[1]`,
		"index an array by bool":   `[1, 2][true]`,
		"spread a number in array": `[...1]`,
		"spread an array in merge": `{...[1]}`,
		"case condition not bool":  `case { 1: true, false }`,
		"filter body not bool":     `[1, 2] | filter x: 1`,
		"computed key not string":  `{[1]: "x"}`,
		"case value never matches": `case "a" { 1: true, false }`,
		"unreachable operand":      `true || (1 && 2)`,
		"global function plus one": `length + 1`,
		"call a number":            `(1)(2)`,
		"call a non-function":      `true(1)`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, compileSource(t, src), argex.ErrType)
		})
	}
	t.Run("dot access on null", func(t *testing.T) {
		assert.ErrorIs(t, compileSource(t, "null.x"), argex.ErrMissing)
	})
}

func TestCompileAllowsRuntimeDependentValues(t *testing.T) {
	cases := map[string]string{
		"bare name access":         `value.status`,
		"optional chained access":  `value?.x.y`,
		"add to unknown value":     `value + 1`,
		"and with unknown value":   `value && true`,
		"in with unknown array":    `value in list`,
		"length of unknown value":  `length(value)`,
		"comprehension on access":  `value.items | any x: x.ok`,
		"spread unknown value":     `[...value]`,
		"merge unknown value":      `{...value}`,
		"case on unknown name":     `case { value: true, 1 }`,
		"index by unknown key":     `value[key]`,
		"chained unknown access":   `value.a.b.c`,
		"access into array index":  `[1][0].x`,
		"any over bound element":   `[1, 2] | any x: x`,
		"unknown left comparison":  `value < 1`,
		"ambiguous index receiver": `(case { true: [1], {a: 1} })[0]`,
		"apply an unknown value":   `value.f(1)`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			assert.NoError(t, compileSource(t, src))
		})
	}
}

func TestCompileAllowsSuccessfulOperands(t *testing.T) {
	cases := map[string]string{
		"both numbers compare":  `1 < 2`,
		"both strings compare":  `"a" < "b"`,
		"both numbers subtract": `1 - 2`,
		"equal any types":       `1 == "a"`,
		"not equal any types":   `1 != "a"`,
		"coalesce any types":    `1 ?? "a"`,
		"object key by name":    `{a: 1}["a"]`,
		"array key by number":   `[1, 2][0]`,
		"min over many numbers": `min(1, 2, 3)`,
		"functions are equal":   `length == length`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			assert.NoError(t, compileSource(t, src))
		})
	}
}

func TestStaticTypeDoesNotEvaluate(t *testing.T) {
	calls := 0
	fn := &argex.Function{
		Call: func(...any) (any, error) {
			calls++
			return float64(42), nil
		},
		Result: argex.TypeNumber,
	}
	env, err := argex.DefaultEnv().Bind("answer", fn)
	if !assert.NoError(t, err) {
		return
	}
	expr, err := argex.Parse("answer()")
	if !assert.NoError(t, err) {
		return
	}
	program, err := env.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, program.StaticType().Is(argex.TypeNumber))
	assert.Zero(t, calls)
	value, err := program.Eval(nil)
	assert.NoError(t, err)
	assert.Equal(t, float64(42), value)
	assert.Equal(t, 1, calls)
	assert.True(t, program.StaticType().Is(argex.TypeNumber))
	assert.Equal(t, 1, calls)
}

func compileSource(t *testing.T, src string) error {
	t.Helper()
	expr, err := argex.Parse(src)
	if err != nil {
		return err
	}
	_, err = argex.DefaultEnv().Compile(expr)
	return err
}
