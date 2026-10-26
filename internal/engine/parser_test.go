package engine_test

import (
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

const maxFuzzSourceSize = 4096

func TestSyntax(t *testing.T) {
	cases := []string{
		"", " ", "1 2", "01", "1e", "1e9999", "1.", "1..2",
		`"unterminated`, `"bad\x20"`, "\"bad\nstring\"",
		"'single'", "@", "$", "x = 2", "x++", "x => x", "x |> f",
		"[1", "[1 2]", "[1,,2]", "{x:}", "{x 1}", "{x:1 x:2}",
		"{x:1,x:2}", "{1:2}", "{true + 1: 2}", "{",
		"foo(", "foo(1 2)", "(1", "(1))", "x[]", "x[1", "x.",
		"x.[0]", "x.1", "x.method(1)",
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
