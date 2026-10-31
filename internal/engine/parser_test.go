package engine_test

import (
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

type namesCase struct {
	source string
	want   []string
}

const maxFuzzSourceSize = 4096

func TestNames(t *testing.T) {
	ns := argex.DefaultEnv()
	cases := []namesCase{
		{source: "1 + 2"},
		{source: "a + b", want: []string{"a", "b"}},
		{source: "a + a", want: []string{"a"}},
		{source: "a.b.c", want: []string{"a"}},
		{source: "length(xs)", want: []string{"xs"}},
		{source: "this.a + this[k]", want: []string{"k"}},
		{source: "{[k]: v, ...rest}", want: []string{"k", "rest", "v"}},
		{source: "case { a: b, c }", want: []string{"a", "b", "c"}},
		{source: "case s { a: b, c }", want: []string{"a", "b", "c", "s"}},
		{source: "items | map i: i.id", want: []string{"items"}},
		{
			source: "items | map i: i + offset",
			want:   []string{"items", "offset"},
		},
		{source: "x | map x: x", want: []string{"x"}},
		{source: "obj | map k, v: [k, v, w]", want: []string{"obj", "w"}},
		{source: "xs | map x: x | map y: x", want: []string{"xs"}},
		{source: "xs | filter x: x | map x * k", want: []string{"k", "xs"}},
		{source: "(xs | map x: x) | map y: x", want: []string{"x", "xs"}},
		{source: "obj | map _, v: [v, _]", want: []string{"_", "obj"}},
		{source: "xs | map ys", want: []string{"xs", "ys"}},
		{source: "xs | filter x: x | any y: y > z", want: []string{"xs", "z"}},
		{
			source: "xs | map x: (ys | map y: x + y + z)",
			want:   []string{"xs", "ys", "z"},
		},
		{source: "let x = a: x + b", want: []string{"a", "b"}},
		{source: "let x = x: x", want: []string{"x"}},
		{source: "let x = 1, y = x: y + z", want: []string{"z"}},
		{source: "(let x = 1: x) + x", want: []string{"x"}},
		{source: "let x = 1: xs | map y: x + y", want: []string{"xs"}},
		{source: "[length, n]", want: []string{"n"}},
		{source: "let length = a: length", want: []string{"a"}},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			expr, err := argex.Parse(tc.source)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, tc.want, ns.Names(expr))
		})
	}
	assert.Empty(t, ns.Names(&argex.Expression{}))
	assert.Empty(t, ns.Names(nil))
}

func TestParseDefersSemanticChecks(t *testing.T) {
	cases := map[string]error{
		"true + 1":             argex.ErrType,
		`case 1 { "a": 1, 2 }`: argex.ErrType,
		"{[1]: 2}":             argex.ErrType,
		"length(1)":            argex.ErrType,
		"unknown()":            argex.ErrFunction,
	}
	for source, want := range cases {
		t.Run(source, func(t *testing.T) {
			expr, err := argex.Parse(source)
			if !assert.NoError(t, err) {
				return
			}
			_, err = argex.DefaultEnv().Compile(expr)
			assert.ErrorIs(t, err, want)
		})
	}
}

func TestSyntax(t *testing.T) {
	cases := []string{
		"", " ", "1 2", "01", "1e", "1e9999", "1.", "1..2",
		`"unterminated`, `"bad\x20"`, "\"bad\nstring\"",
		"\"bad\xffutf8\"", `"\ud800"`, "\"tab\there\"",
		"'single'", "@", "$", "x = 2", "x++", "x => x", "x |> f",
		"[1", "[1 2]", "[1,,2]", "{x:}", "{x 1}", "{x:1 x:2}",
		"{x:1,x:2}", "{1:2}", "{true + 1: 2}", "{",
		"foo(", "foo(1 2)", "(1", "(1))", "x[]", "x[1", "x.",
		"x.[0]", "x.1",
		"x | map item item", "x | map true: 1",
		"x | map false: 1", "x | map null: 1", "x | map case: 1",
		"x?.map x: x", "x | map x:", "x | map |x| x",
		"x | map x -> x", "case { true -> 1, 2 }",
		"x.map item: item", "x.filter item: true", "x | unknown x: x",
		"x|", "x?.", "x | map", "x | filter",
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

func TestWhitespace(t *testing.T) {
	cases := []string{
		`([1,11]|filter x:x>10)|map x:case{x>100:"big","small",}`,
		` ( [ 1 , 11 ] | filter x : x > 10 ) | map x :
		case { x > 100 : "big" , "small" , } `,
		"([1,11]|filter\nx\n:\nx>10)\n|map\nx\n:\n" +
			`case{x>100:"big","small",}`,
	}
	for _, source := range cases {
		result, err := argex.Eval(source, nil)
		assert.NoError(t, err)
		assert.Equal(t, []any{"small"}, result)
	}
}

func TestGrouping(t *testing.T) {
	chained, err := argex.Eval("[1,11] | filter x: x>10 | map x: x*2", nil)
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(22)}, chained)
	grouped, err := argex.Eval("([1,11] | filter x: x>10) | map x: x*2", nil)
	assert.NoError(t, err)
	assert.Equal(t, chained, grouped)
	value, err := argex.Eval("[1] | map x: x + 2 * 3", nil)
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(7)}, value)
}

func FuzzExpression(f *testing.F) {
	for _, source := range []string{
		"1+2", "x?.y ?? 4", "case { true: 1, 2, }",
		"([1,2] | filter x: x>1) | map x: x+2", "{a:[true,null]}",
		`["market:europe"] | any tag: startsWith(tag, "market:")`,
		"[[1],[2]] | all xs: xs | any x: x>0",
		`join(split("a,b", ","), " / ")`,
		`sum(["1", "2"] | map x: number(x))`,
		`keys({a: string(round(1.5))})`,
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
