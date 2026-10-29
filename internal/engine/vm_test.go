package engine_test

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

type evaluationCase struct {
	source string
	want   any
}

const (
	largeCollectionSize = 100001
	wideStackSize       = 80
	collectionDepth     = 6
	remainderSamples    = 1024
)

func TestColonContexts(t *testing.T) {
	source := `{
		results: (items | filter item: item > 10) | map item: {
			value: item,
			category: case {
				item > 100: { label: "large" },
				{ label: "medium" },
			},
		},
		fallback: case { false: 1, case { false: 2, { value: 3 } } },
	}`
	result, err := argex.Eval(source, map[string]any{
		"items": []any{1, 20, 150},
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"results": []any{
			map[string]any{
				"value":    20,
				"category": map[string]any{"label": "medium"},
			},
			map[string]any{
				"value":    150,
				"category": map[string]any{"label": "large"},
			},
		},
		"fallback": map[string]any{"value": float64(3)},
	}, result)
}

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
			source: "items | map item: item + offset",
			want: []any{
				float64(4), float64(8), float64(14), float64(104),
			},
		},
		{source: "items | filter item: item > 10", want: []any{11, 101}},
		{
			source: `(items | filter item: item > 10) | map item: case {
				item > 100: "large", "medium",
			}`,
			want: []any{"medium", "large"},
		},
		{
			source: `(items | map item: item * 2) | filter item: item > 10`,
			want:   []any{float64(22), float64(202)},
		},
		{
			source: `items | map item: (second | map other: item + other)`,
			want: []any{
				[]any{float64(3), float64(5)},
				[]any{float64(7), float64(9)},
				[]any{float64(13), float64(15)},
				[]any{float64(103), float64(105)},
			},
		},
		{
			source: `second | map item: (second | map item: item)`,
			want:   []any{[]any{2, 4}, []any{2, 4}},
		},
		{source: `[] | map item: item`, want: []any{}},
		{source: `[] | filter item: true`, want: []any{}},
		{source: "case { false: 1, true: 2, 3 }", want: float64(2)},
		{source: "case { false: 1, 3, }", want: float64(3)},
		{source: "case { true: 1, 1 / 0 }", want: float64(1)},
		{source: "case { false: 1 / 0, 3 }", want: float64(3)},
		{source: "case { true: 1, 1 / 0 > 2: 2, 3 }", want: float64(1)},
		{source: `case { false: 0, case { true: 2, 3 } }`, want: float64(2)},
		{source: "case { false: 0, -2 < 0: 4, 5 }", want: float64(4)},
		{
			source: `(case { true: [1], [], }) | map x: x + 1`,
			want:   []any{float64(2)},
		},
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
		{source: `endsWith("abc", "bc")`, want: true},
		{source: `includes("abc", "b")`, want: true},
		{source: `lower("ABC")`, want: "abc"},
		{source: `upper("abc")`, want: "ABC"},
		{source: `trim("  abc  ")`, want: "abc"},
		{
			source: `values({a: 1, b: 2})`,
			want:   []any{float64(1), float64(2)},
		},
		{
			source: "[10, 20] | map i, v: i + v",
			want:   []any{float64(10), float64(21)},
		},
		{
			source: `{a: 1, b: 2} | map k, v: k`,
			want:   []any{"a", "b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			value, err := argex.Eval(tc.source, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestComparisonJumps(t *testing.T) {
	variables := map[string]any{
		"n":     5,
		"s":     "m",
		"flag":  true,
		"items": []any{1, 5, 11},
	}
	cases := []evaluationCase{
		{source: `case { n < 5: "a", "b" }`, want: "b"},
		{source: `case { n <= 5: "a", "b" }`, want: "a"},
		{source: `case { n > 5: "a", "b" }`, want: "b"},
		{source: `case { n >= 5: "a", "b" }`, want: "a"},
		{source: `case { s < "n": "a", "b" }`, want: "a"},
		{source: `case { s >= "n": "a", "b" }`, want: "b"},
		{source: "items | filter x: x > 4", want: []any{5, 11}},
		{
			source: "(items | filter x: x >= 5) | map x: x * 2",
			want:   []any{float64(10), float64(22)},
		},
		{source: "items | any x: x > 10", want: true},
		{source: "items | any x: x > 11", want: false},
		{source: "items | all x: x >= 1", want: true},
		{source: "items | all x: x < 11", want: false},
		{source: `case { flag ?? n < 1: "a", "b" }`, want: "a"},
		{source: `case { case { n < 1: false, n > 1 }: "a", "b" }`, want: "a"},
		{source: `case { case { n > 1: false, n > 1 }: "a", "b" }`, want: "b"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			value, err := argex.Eval(tc.source, variables)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestComparisonJumpErrors(t *testing.T) {
	variables := map[string]any{
		"n":     5,
		"s":     "m",
		"flag":  true,
		"items": []any{1, 5, 11},
	}
	sources := []string{
		`case { s < n: "a", "b" }`,
		`case { flag < n: "a", "b" }`,
		`case { n < s: "a", "b" }`,
		`case { s <= n: "a", "b" }`,
		`case { flag > n: "a", "b" }`,
		`case { n >= s: "a", "b" }`,
		`items | filter x: x > s`,
		`items | any x: s >= x`,
		`items | all x: x >= s`,
	}
	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			_, err := argex.Eval(source, variables)
			assert.ErrorIs(t, err, argex.ErrType)
		})
	}
}

func TestLoopNames(t *testing.T) {
	variables := map[string]any{
		"t":     true,
		"f":     false,
		"x":     1,
		"r":     10,
		"items": []any{1, 2},
	}
	cases := []evaluationCase{
		{
			source: "[items | map i: i + r, r]",
			want:   []any{[]any{float64(11), float64(12)}, 10},
		},
		{source: "items | map i: items", want: []any{
			[]any{1, 2}, []any{1, 2},
		}},
		{source: "[[] | map i: x, x]", want: []any{[]any{}, 1}},
		{
			source: "case { t: items | map i: r, 0 }",
			want:   []any{10, 10},
		},
		{source: "case { f: items | map i: r, x }", want: 1},
		{
			source: "items | map i: (items | map j: i * r)",
			want: []any{
				[]any{float64(10), float64(10)},
				[]any{float64(20), float64(20)},
			},
		},
		{
			source: "items | map i: (let r = 2: i * r)",
			want:   []any{float64(2), float64(4)},
		},
		{
			source: "items | map i: case { i > 1: x, absent }",
			want:   []any{argex.Missing{}, 1},
		},
		{
			source: "[items | map i: r, items | filter i: i < x]",
			want:   []any{[]any{10, 10}, []any{}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			value, err := argex.Eval(tc.source, variables)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestWideStack(t *testing.T) {
	var elements []string
	want := make([]any, wideStackSize)
	for i := range want {
		elements = append(elements, fmt.Sprint(i))
		want[i] = float64(i)
	}
	source := "[" + strings.Join(elements, ",") + "]"
	expr, err := argex.Parse(source)
	if !assert.NoError(t, err) {
		return
	}
	program, err := argex.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	first, err := program(nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, want, first)
	first.([]any)[0] = "changed"
	second, err := program(nil)
	assert.NoError(t, err)
	assert.Equal(t, want, second)
}

func TestDeepCollections(t *testing.T) {
	source := "x0"
	for i := collectionDepth - 1; i >= 0; i-- {
		source = fmt.Sprintf("[1] | map x%d: (%s)", i, source)
	}
	value, err := argex.Eval(source, nil)
	assert.NoError(t, err)
	want := any(float64(1))
	for range collectionDepth {
		want = []any{want}
	}
	assert.Equal(t, want, value)
}

func TestDeepQuantifiers(t *testing.T) {
	for _, operation := range []string{"any", "all"} {
		source := "true"
		for range collectionDepth {
			source = "[1] | " + operation + " item: (" + source + ")"
		}
		value, err := argex.Eval(source, nil)
		assert.NoError(t, err)
		assert.Equal(t, true, value)
	}
}

func TestWideEquality(t *testing.T) {
	left := make([]any, wideStackSize)
	right := make([]any, wideStackSize)
	for i := range left {
		left[i] = map[string]any{"value": i}
		right[i] = map[string]any{"value": float64(i)}
	}
	variables := map[string]any{"left": left, "right": right}
	value, err := argex.Eval("left == right", variables)
	assert.NoError(t, err)
	assert.Equal(t, true, value)
	right[len(right)-1] = nil
	value, err = argex.Eval("left != right", variables)
	assert.NoError(t, err)
	assert.Equal(t, true, value)
}

func TestNumericIntermediates(t *testing.T) {
	cases := []evaluationCase{
		{source: "(1 + 2) * (4 - 1) / 3 % 2", want: float64(1)},
		{source: "-(-(1 + 2))", want: float64(3)},
		{source: "+(1 + 2)", want: float64(3)},
		{source: "(1 + 2) < 4 && (2 + 2) <= 4", want: true},
		{source: "4 > (1 + 2) && 4 >= (2 + 2)", want: true},
		{source: "(1 + 2) == (4 - 1)", want: true},
		{source: "3 == (1 + 2)", want: true},
		{source: "(1 + 2) != true", want: true},
		{source: "(1 + 2) ?? 9", want: float64(3)},
		{source: "[7][1 - 1]", want: float64(7)},
		{source: "[7]?.[1 - 1]", want: float64(7)},
		{source: "{a: 1 + 2}.a", want: float64(3)},
		{source: "abs(1 - 3) + length([1, 2])", want: float64(4)},
		{source: "min(1 + 2, 5 - 1)", want: float64(3)},
		{source: "max(1 + 2, 5 - 1)", want: float64(4)},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			value, err := argex.Eval(tc.source, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
	source := "[" + strings.Repeat("1 + 2,", wideStackSize) + "]"
	want := make([]any, wideStackSize)
	for i := range want {
		want[i] = float64(3)
	}
	value, err := argex.Eval(source, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, value)
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

func TestRemainder(t *testing.T) {
	expr, err := argex.Parse("left % right")
	if !assert.NoError(t, err) {
		return
	}
	program, err := argex.Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	values := []float64{
		0, math.Copysign(0, -1), 1, -1, 1.5, -1.5,
		math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64,
		math.MaxFloat64, -math.MaxFloat64,
		math.Ldexp(1, -1022), math.Ldexp(1, -1022) * 1.5,
	}
	variables := map[string]any{}
	for _, left := range values {
		for _, right := range values {
			if right == 0 {
				continue
			}
			variables["left"] = left
			variables["right"] = right
			value, err := program(variables)
			if !assert.NoError(t, err) {
				return
			}
			want := math.Float64bits(math.Mod(left, right))
			assert.Equal(t, want, math.Float64bits(value.(float64)))
		}
	}
	random := rand.New(rand.NewPCG(1, 2))
	for range remainderSamples {
		left := math.Float64frombits(random.Uint64())
		right := math.Float64frombits(random.Uint64())
		if left-left != 0 || right-right != 0 || right == 0 {
			continue
		}
		variables["left"] = left
		variables["right"] = right
		value, err := program(variables)
		if !assert.NoError(t, err) {
			return
		}
		want := math.Float64bits(math.Mod(left, right))
		assert.Equal(t, want, math.Float64bits(value.(float64)))
	}
}

func TestLocalLifetime(t *testing.T) {
	source := `[
		items | map x: (items | map y: x + y)[0] + x,
		items | map x: x + 1,
		x,
	]`
	value, err := argex.Eval(source, map[string]any{
		"items": []any{1, 2}, "x": "outside",
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{
		[]any{float64(3), float64(5)},
		[]any{float64(2), float64(3)},
		"outside",
	}, value)
	value, err = argex.Eval("[1] | map x: [2] | map x: x", nil)
	assert.NoError(t, err)
	assert.Equal(t, []any{[]any{float64(2)}}, value)
}

func TestLargeCollections(t *testing.T) {
	items := make([]any, largeCollectionSize)
	for i := range items {
		items[i] = i
	}
	for _, source := range []string{
		"items | map item: item", "items | filter item: true",
	} {
		value, err := argex.Eval(source, map[string]any{"items": items})
		assert.NoError(t, err)
		assert.Equal(t, items, value)
	}
}
