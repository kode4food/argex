package engine_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

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

func TestWideStack(t *testing.T) {
	var elements []string
	want := make([]any, wideStackSize)
	for i := range want {
		elements = append(elements, fmt.Sprint(i))
		want[i] = float64(i)
	}
	source := "[" + strings.Join(elements, ",") + "]"
	expr, err := argex.Parse(source)
	assert.NoError(t, err)
	program, err := argex.Compile(expr)
	assert.NoError(t, err)
	first, err := program(nil)
	assert.NoError(t, err)
	assert.Equal(t, want, first)
	first.([]any)[0] = "changed"
	second, err := program(nil)
	assert.NoError(t, err)
	assert.Equal(t, want, second)
}

func TestDeepCollections(t *testing.T) {
	source := "x0"
	for i := collectionDepth - 1; i >= 0; i-- {
		source = fmt.Sprintf("[1] | map x%d: %s", i, source)
	}
	value, err := argex.Eval(source, nil)
	assert.NoError(t, err)
	want := any(float64(1))
	for range collectionDepth {
		want = []any{want}
	}
	assert.Equal(t, want, value)
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

func TestSkippedCompilation(t *testing.T) {
	for _, source := range []string{
		"case { true: 1, unknown() }",
		"false && unknown()", "1 ?? unknown()", "null?.[unknown()]",
		"[] | map x: unknown()", "[] | filter x: unknown()",
	} {
		_, err := argex.Eval(source, nil)
		assert.ErrorIs(t, err, argex.ErrFunction)
	}
}
