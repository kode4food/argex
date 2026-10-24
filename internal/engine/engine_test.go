package engine_test

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

type evaluationCase struct {
	source string
	want   any
}

const (
	reuseWorkers      = 10
	reuseIterations   = 10
	maxFuzzSourceSize = 4096
	expressionDepth   = 256
)

func TestValues(t *testing.T) {
	variables := map[string]any{
		"customer": map[string]any{
			"address":  map[string]any{"country": "CH"},
			"name":     "Ada",
			"nullable": nil,
			"map":      "property",
		},
		"items":     []any{1, 5, 11, 101},
		"second":    []any{2, 4},
		"index":     1,
		"key":       "name",
		"offset":    3,
		"trueValue": 9,
		"世界":        "hello",
	}
	cases := []evaluationCase{
		{source: "null", want: nil},
		{source: "true", want: true},
		{source: "false", want: false},
		{source: "0", want: float64(0)},
		{source: "-1.5e+2", want: float64(-150)},
		{source: "2E-2", want: 0.02},
		{source: `"hello\n\u263a"`, want: "hello\n☺"},
		{source: `"\ud83d\ude00"`, want: "😀"},
		{source: `"a\\b\"c"`, want: "a\\b\"c"},
		{source: "世界", want: "hello"},
		{source: "trueValue", want: 9},
		{source: "customer.address.country", want: "CH"},
		{source: `customer["address"]["country"]`, want: "CH"},
		{source: "customer[key]", want: "Ada"},
		{source: "customer.map", want: "property"},
		{source: "items[index]", want: 5},
		{source: "[1, 2,][0]", want: float64(1)},
		{source: "[]", want: []any{}},
		{source: "{}", want: map[string]any{}},
		{
			source: `{x: 1, "y": [true, null],}`,
			want: map[string]any{
				"x": float64(1),
				"y": []any{true, nil},
			},
		},
		{source: "customer?.address?.country", want: "CH"},
		{source: "items?.[1]", want: 5},
		{source: `missing?.field ?? "fallback"`, want: "fallback"},
		{source: "customer.nullable ?? 4", want: float64(4)},
		{source: "missing ?? null ?? 4", want: float64(4)},
		{source: "false ?? true", want: false},
		{source: "0 ?? 4", want: float64(0)},
		{source: `"" ?? "fallback"`, want: ""},
		{source: "items?.[99] ?? 8", want: float64(8)},
		{source: "items?.[-1] ?? 8", want: float64(8)},
		{source: "null?.[1/0] ?? 8", want: float64(8)},
		{source: "null?.name ?? 8", want: float64(8)},
		{source: "customer?.absent ?? 8", want: float64(8)},
		{source: "missing == null", want: false},
		{source: "missing == another_missing", want: true},
		{source: "customer.nullable == null", want: true},
		{
			source: "{kept: null, omitted: missing}",
			want:   map[string]any{"kept": nil},
		},
		{
			source: "items.map item: item + offset",
			want: []any{
				float64(4), float64(8), float64(14), float64(104),
			},
		},
		{source: "items.filter item: item > 10", want: []any{11, 101}},
		{
			source: `(items.filter item: item > 10).map item: case {
				item > 100: "large", "medium",
			}`,
			want: []any{"medium", "large"},
		},
		{source: `(items.map item: item * 2).filter item: item > 10`,
			want: []any{float64(22), float64(202)}},
		{source: `items.map item: second.map other: item + other`,
			want: []any{
				[]any{float64(3), float64(5)},
				[]any{float64(7), float64(9)},
				[]any{float64(13), float64(15)},
				[]any{float64(103), float64(105)},
			}},
		{source: `second.map item: second.map item: item`,
			want: []any{[]any{2, 4}, []any{2, 4}}},
		{source: `[] . map item: item`, want: []any{}},
		{source: `[] . filter item: true`, want: []any{}},
		{source: "case { false: 1, true: 2, 3 }", want: float64(2)},
		{source: "case { false: 1, 3, }", want: float64(3)},
		{source: "case { true: 1, 1 / 0 }", want: float64(1)},
		{source: "case { false: 1 / 0, 3 }", want: float64(3)},
		{source: "case { true: 1, 1 / 0 > 2: 2, 3 }",
			want: float64(1)},
		{source: `case { false: 0, case { true: 2, 3 } }`,
			want: float64(2)},
		{source: "case { false: 0, -2 < 0: 4, 5 }",
			want: float64(4)},
		{source: `(case { true: [1], [], }).map x: x + 1`,
			want: []any{float64(2)}},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			value, err := argex.Eval(tc.source, variables)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestOperators(t *testing.T) {
	cases := []evaluationCase{
		{source: "2 + 3 * 4", want: float64(14)},
		{source: "(2 + 3) * 4", want: float64(20)},
		{source: "10 - 3 - 2", want: float64(5)},
		{source: "20 / 2 / 5", want: float64(2)},
		{source: "17 % 5", want: float64(2)},
		{source: "-17 % 5", want: float64(-2)},
		{source: "+2 * -3", want: float64(-6)},
		{source: "!false && true", want: true},
		{source: "true || false && false", want: true},
		{source: "2 + 3 == 5 && 1 < 2", want: true},
		{source: "1 != 2", want: true},
		{source: "1 <= 1", want: true},
		{source: "2 >= 1", want: true},
		{source: "2 > 1", want: true},
		{source: `"a" + "b"`, want: "ab"},
		{source: `"a" < "b"`, want: true},
		{source: `"a" <= "a"`, want: true},
		{source: `"b" > "a"`, want: true},
		{source: `"b" >= "b"`, want: true},
		{source: `"3" == 3`, want: false},
		{source: "[1, {a: 2}] == [1, {a: 2}]", want: true},
		{source: "[1] == []", want: false},
		{source: "[1] == 1", want: false},
		{source: "[1] == [2]", want: false},
		{source: "{} == []", want: false},
		{source: "{a:1} == {}", want: false},
		{source: "{a:1} == {b:1}", want: false},
		{source: "{a:1} == {a:2}", want: false},
		{source: "false && (1 / 0 > 0)", want: false},
		{source: "true || missing.field", want: true},
		{source: "1 ?? missing.field", want: float64(1)},
		{source: "false || true", want: true},
		{source: "true && false", want: false},
		{source: "null ?? 2 + 3 == 5", want: true},
		{source: "false ?? true || true", want: true},
		{source: "length([1,2])", want: float64(2)},
		{source: "length({a: 1})", want: float64(1)},
		{source: `length("é😀")`, want: float64(2)},
		{source: "abs(-5)", want: float64(5)},
		{source: "min(5, 1, 4)", want: float64(1)},
		{source: "max(5, 1, 9)", want: float64(9)},
		{source: "min(5)", want: float64(5)},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			value, err := argex.Eval(tc.source, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestSyntax(t *testing.T) {
	cases := []string{
		"", " ", "1 2", "01", "1e", "1e9999", "1.", "1..2",
		`"unterminated`, `"bad\x20"`, "\"bad\nstring\"",
		"'single'", "@", "$", "x = 2", "x++", "x => x", "x |> f",
		"[1", "[1 2]", "[1,,2]", "{x:}", "{x 1}", "{x:1 x:2}",
		"{x:1,x:2}", "{1:2}", "{true + 1: 2}", "{",
		"foo(", "foo(1 2)", "(1", "(1))", "x[]", "x[1", "x.",
		"x.[0]", "x.1", "x.method(1)",
		"x.map item item", "x.map true: 1",
		"x.map false: 1", "x.map null: 1", "x.map case: 1",
		"x?.map x: x", "x.map x:", "x.map |x| x",
		"x.map x -> x", "case { true -> 1, 2 }",
		"case {}", "case { 1 }", "case { true: 1 }",
		"case { true: 1, }", "case { true: 1 false: 2, 3 }",
		"case { true: 1, 2, false: 3 }", "case true: 1",
		"case { true: , 2 }", "1 < 2 < 3", "1 +", "!", "x ? 1 : 2",
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			_, err := argex.Parse(source)
			assert.ErrorIs(t, err, argex.ErrSyntax)
		})
	}
}

func TestErrors(t *testing.T) {
	cases := map[string]error{
		"missing.field":             argex.ErrMissing,
		"{}.field":                  argex.ErrMissing,
		"[1][5]":                    argex.ErrMissing,
		"null.field":                argex.ErrMissing,
		"1 + true":                  argex.ErrType,
		"true + 1":                  argex.ErrType,
		`"1" + 2`:                   argex.ErrType,
		`"1" - "2"`:                 argex.ErrType,
		"!1":                        argex.ErrType,
		"-true":                     argex.ErrType,
		"+true":                     argex.ErrType,
		"1 - true":                  argex.ErrType,
		"1 * true":                  argex.ErrType,
		"1 / true":                  argex.ErrType,
		"1 % true":                  argex.ErrType,
		"1 < true":                  argex.ErrType,
		"1 <= true":                 argex.ErrType,
		"1 > true":                  argex.ErrType,
		"1 >= true":                 argex.ErrType,
		`"1" < 2`:                   argex.ErrType,
		"1 && true":                 argex.ErrType,
		"false || 1":                argex.ErrType,
		"true && 1":                 argex.ErrType,
		"case { 1: 2, 3 }":          argex.ErrType,
		"[1].filter x: x":           argex.ErrType,
		"1 .map x: x":               argex.ErrType,
		"[1].map x: x.field":        argex.ErrType,
		"1 / 0":                     argex.ErrArithmetic,
		"0 / 0":                     argex.ErrArithmetic,
		"1 % 0":                     argex.ErrArithmetic,
		"1e308 * 1e308":             argex.ErrArithmetic,
		"-1e308 - 1e308":            argex.ErrArithmetic,
		"[1][0.5]":                  argex.ErrType,
		`[1]["x"]`:                  argex.ErrType,
		"{}[0]":                     argex.ErrType,
		"true?.field":               argex.ErrType,
		"customer?.absent.field":    argex.ErrMissing,
		"missing.field ?? 1":        argex.ErrMissing,
		"[1].map x: other.field":    argex.ErrMissing,
		"[1].filter x: other.field": argex.ErrMissing,
		"length(1)":                 argex.ErrType,
		"abs(true)":                 argex.ErrType,
		"min(1,true)":               argex.ErrType,
		"min(true,1)":               argex.ErrType,
		"max(1,true)":               argex.ErrType,
		"max(true,1)":               argex.ErrType,
		"abs()":                     argex.ErrFunction,
		"length(1,2)":               argex.ErrFunction,
		"min()":                     argex.ErrFunction,
		"max()":                     argex.ErrFunction,
		"unknown(1)":                argex.ErrFunction,
		"[1].map x: unknown()":      argex.ErrFunction,
	}
	for source, want := range cases {
		t.Run(source, func(t *testing.T) {
			_, err := argex.Eval(source, map[string]any{
				"customer": map[string]any{},
			})
			assert.ErrorIs(t, err, want)
		})
	}
	_, err := argex.Compile(nil)
	assert.ErrorIs(t, err, argex.ErrSyntax)
	_, err = argex.Compile(&argex.Expression{})
	assert.ErrorIs(t, err, argex.ErrSyntax)
}

func TestMissing(t *testing.T) {
	for _, source := range []string{"absent", "null?.x", "{}?.x"} {
		value, err := argex.Eval(source, nil)
		assert.NoError(t, err)
		assert.True(t, argex.IsMissing(value))
	}
	assert.False(t, argex.IsMissing(nil))
	assert.False(t, argex.IsMissing(false))
	_, err := json.Marshal(argex.Missing{})
	assert.ErrorIs(t, err, argex.ErrMissing)
	value, err := argex.Eval("[absent]", nil)
	assert.NoError(t, err)
	_, err = json.Marshal(value)
	assert.ErrorIs(t, err, argex.ErrMissing)
}

func TestNumericInputs(t *testing.T) {
	for _, value := range []any{1, float64(1), json.Number("1")} {
		result, err := argex.Eval("value + 2 == 3", map[string]any{
			"value": value,
		})
		assert.NoError(t, err)
		assert.Equal(t, true, result)
		result, err = argex.Eval(
			"{a: [value]} == {a: [1]}", map[string]any{"value": value},
		)
		assert.NoError(t, err)
		assert.Equal(t, true, result)
	}
	for _, value := range []any{
		json.Number("bad"), math.Inf(1), math.NaN(), "1",
	} {
		_, err := argex.Eval("value + 2", map[string]any{"value": value})
		assert.ErrorIs(t, err, argex.ErrType)
	}
}

func TestWhitespace(t *testing.T) {
	cases := []string{
		`([1,11].filter x:x>10).map x:case{x>100:"big","small",}`,
		` ( [ 1 , 11 ] . filter x : x > 10 ) . map x :
		case { x > 100 : "big" , "small" , } `,
		"([1,11].filter\nx\n:\nx>10)\n.map\nx\n:\n" +
			`case{x>100:"big","small",}`,
	}
	for _, source := range cases {
		result, err := argex.Eval(source, nil)
		assert.NoError(t, err)
		assert.Equal(t, []any{"small"}, result)
	}
}

func TestGrouping(t *testing.T) {
	value, err := argex.Eval(
		"([1,11].filter x: x>10).map x: x*2", nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(22)}, value)
	_, err = argex.Eval("[1,11].filter x: x>10.map x: x*2", nil)
	assert.Error(t, err)
	value, err = argex.Eval("[1].map x: x + 2 * 3", nil)
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(7)}, value)
}

func TestLimits(t *testing.T) {
	_, err := argex.Parse(
		strings.Repeat("(", expressionDepth) +
			"1" + strings.Repeat(")", expressionDepth),
	)
	assert.ErrorIs(t, err, argex.ErrLimit)
	expr, err := argex.Parse("1" + strings.Repeat("+1", expressionDepth+1))
	assert.NoError(t, err)
	_, err = argex.Compile(expr)
	assert.ErrorIs(t, err, argex.ErrLimit)
}

func TestProgramFunction(t *testing.T) {
	expr, err := argex.Parse("value + 1")
	assert.NoError(t, err)
	program, err := argex.Compile(expr)
	assert.NoError(t, err)
	var run func(map[string]any) (any, error) = program
	value, err := run(map[string]any{"value": 2})
	assert.NoError(t, err)
	assert.Equal(t, float64(3), value)
}

func TestReuse(t *testing.T) {
	expr, err := argex.Parse("(items.filter x: x > 1).map x: x + n")
	assert.NoError(t, err)
	program, err := argex.Compile(expr)
	assert.NoError(t, err)
	variables := map[string]any{"items": []any{1, 2}, "n": 3}
	var workers sync.WaitGroup
	for range reuseWorkers {
		workers.Go(func() {
			for range reuseIterations {
				result, err := program(variables)
				assert.NoError(t, err)
				assert.Equal(t, []any{float64(5)}, result)
				result.([]any)[0] = "changed"
			}
		})
	}
	workers.Wait()
	assert.Equal(t, []any{1, 2}, variables["items"])
	second, err := argex.Compile(expr)
	assert.NoError(t, err)
	value, err := second(map[string]any{"items": []any{5}, "n": 1})
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(6)}, value)
}

func FuzzExpression(f *testing.F) {
	for _, source := range []string{
		"1+2", "x?.y ?? 4", "case { true: 1, 2, }",
		"([1,2].filter x: x>1).map x: x+2", "{a:[true,null]}",
	} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > maxFuzzSourceSize {
			t.Skip()
		}
		_, _ = argex.Eval(source, nil)
	})
}
