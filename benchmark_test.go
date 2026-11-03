package argex_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/kode4food/argex"
)

type (
	benchmarkCase struct {
		name   string
		source string
	}

	equalityCase struct {
		name   string
		items  []any
		needle any
	}
)

const benchmarkCollectionSize = 256

func BenchmarkCompile(b *testing.B) {
	for _, tc := range []benchmarkCase{
		{name: "literal", source: "1"},
		{name: "input", source: "this"},
		{name: "access", source: "a?.b?.c ?? false"},
		{
			name:   "arithmetic",
			source: "(input * price + output * rate) * (1 + tax)",
		},
		{
			name:   "pipeline",
			source: "items | map x: x * 2 | filter y: y > 128 | any z: z > 256",
		},
		{
			name:   "match",
			source: `case status { "paid": "ship", "pending": "wait", "stop" }`,
		},
		{
			name: "functions",
			source: "let size = length: " +
				"{count: size(items), peak: max(...items)}",
		},
		{
			name:   "object",
			source: `{["name"]: user.name, active: user.active, ...defaults}`,
		},
		{
			name: "array256",
			source: "[" +
				strings.Repeat("item,", benchmarkCollectionSize) + "]",
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			expr, err := argex.Parse(tc.source)
			if err != nil {
				b.Fatal(err)
			}
			env := argex.DefaultEnv()
			b.ReportAllocs()
			for b.Loop() {
				if _, err := env.Compile(expr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkExpressions(b *testing.B) {
	variables := map[string]any{
		"a":     map[string]any{"b": map[string]any{"c": true}},
		"items": []any{1.0, 5.0, 11.0, 101.0},
	}
	for _, source := range []string{
		"a.b.c",
		"a?.b?.c ?? false",
		`(items | filter x: x > 10) | map x: case {
			x > 100: "large", "medium",
		}`,
	} {
		b.Run(source, func(b *testing.B) {
			expr, err := argex.Parse(source)
			if err != nil {
				b.Fatal(err)
			}
			program, err := argex.DefaultEnv().Compile(expr)
			if err != nil {
				b.Fatal(err)
			}
			b.Run("compiled", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, err := program.Eval(variables)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("parse-compile-eval", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, err := argex.Eval(source, variables)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkBindingPatterns(b *testing.B) {
	items := make([]any, benchmarkCollectionSize)
	for i := range items {
		items[i] = map[string]any{"x": float64(i), "y": float64(i + 1)}
	}
	variables := map[string]any{
		"input": map[string]any{"x": 2.0, "y": 3.0},
		"pair":  []any{2.0, 3.0},
		"items": items,
		"empty": map[string]any{},
		"none":  []any{},
	}
	for _, tc := range []benchmarkCase{
		{
			name:   "object/manual",
			source: "let item = input, x = item.x, y = item.y: x + y",
		},
		{name: "object/pattern", source: "let {x, y} = input: x + y"},
		{name: "object/known", source: "let {x, y} = {x: 2, y: 3}: x + y"},
		{
			name:   "object/default-present",
			source: "let {x = 2, y = 3} = input: x + y",
		},
		{
			name:   "object/default-missing",
			source: "let {x = 2, y = 3} = empty: x + y",
		},
		{
			name:   "array/manual",
			source: "let item = pair, x = item[0], y = item[1]: x + y",
		},
		{name: "array/pattern", source: "let [x, y] = pair: x + y"},
		{name: "array/known", source: "let [x, y] = [2, 3]: x + y"},
		{
			name:   "array/default-present",
			source: "let [x = 2, y = 3] = pair: x + y",
		},
		{
			name:   "array/default-missing",
			source: "let [x = 2, y = 3] = none: x + y",
		},
		{name: "map256/direct", source: "items | map item: item.x + item.y"},
		{
			name: "map256/manual",
			source: "items | map item: " +
				"(let x = item.x, y = item.y: x + y)",
		},
		{name: "map256/pattern", source: "items | map {x, y}: x + y"},
		{
			name:   "map256/default-present",
			source: "items | map {x = 0, y = 0}: x + y",
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			expr, err := argex.Parse(tc.source)
			if err != nil {
				b.Fatal(err)
			}
			program, err := argex.DefaultEnv().Compile(expr)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := program.Eval(variables); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkWorkloads(b *testing.B) {
	items := make([]any, benchmarkCollectionSize)
	for i := range items {
		items[i] = float64(i)
	}
	variables := map[string]any{
		"items":         items,
		"input_tokens":  1000.0,
		"output_tokens": 250.0,
		"input_price":   0.01,
		"output_price":  0.03,
		"tax":           0.2,
		"rate":          2.0,
		"offset":        3.0,
		"quality":       0.95,
	}
	cases := []benchmarkCase{
		{
			name: "arithmetic",
			source: `(input_tokens * input_price +
				output_tokens * output_price) * (1 + tax)`,
		},
		{
			name: "branch",
			source: `case { quality >= 0.9: "accept",
				quality >= 0.5: "review", "reject" }`,
		},
		{name: "map256", source: "items | map item: item * rate + offset"},
		{name: "filter256", source: "items | filter item: item > 128"},
		{name: "any256", source: "items | any item: item > 128"},
		{name: "all256", source: "items | all item: item >= 0"},
		{name: "sum256", source: "sum(items)"},
		{
			name:   "chain256",
			source: "items | map x: x * 2 | filter y: y > 128 | map z: z + 1",
		},
		{
			name:   "keyedFirst256",
			source: "items | map i, x: i * x | filter y: y > 100",
		},
		{
			name:   "keyedLater256",
			source: "items | filter x: x > 64 | map i, x: i + x",
		},
		{name: "anyChain256", source: "items | map x: x * 2 | any y: y > 256"},
		{
			name:   "allChain256",
			source: "items | filter x: x > 10 | all y: y >= 0",
		},
		{
			name:   "nested",
			source: `items | map item: ([1,2,3,4] | map inner: item + inner)`,
		},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			expr, err := argex.Parse(tc.source)
			if err != nil {
				b.Fatal(err)
			}
			program, err := argex.DefaultEnv().Compile(expr)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				_, err := program.Eval(variables)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMembership(b *testing.B) {
	items := []any{
		"domain:inventory", "domain:orders", "tier:gold",
		"market:europe", "environment:production", "example",
	}
	for _, needle := range []string{"domain:inventory", "example", "absent"} {
		variables := map[string]any{"needle": needle, "items": items}
		for _, tc := range []benchmarkCase{
			{name: "in", source: "needle in items"},
			{
				name:   "filter",
				source: "length(items | filter item: item == needle) > 0",
			},
		} {
			b.Run(needle+"/"+tc.name, func(b *testing.B) {
				expr, err := argex.Parse(tc.source)
				if err != nil {
					b.Fatal(err)
				}
				program, err := argex.DefaultEnv().Compile(expr)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				for b.Loop() {
					if _, err := program.Eval(variables); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkEquality(b *testing.B) {
	floats := make([]any, benchmarkCollectionSize)
	numbers := make([]any, benchmarkCollectionSize)
	objects := make([]any, benchmarkCollectionSize)
	for i := range benchmarkCollectionSize {
		floats[i] = float64(i)
		numbers[i] = json.Number(strconv.Itoa(i))
		objects[i] = map[string]any{"id": float64(i), "tags": []any{"a"}}
	}
	last := benchmarkCollectionSize - 1
	expr, err := argex.Parse("length(items | filter item: item == needle)")
	if err != nil {
		b.Fatal(err)
	}
	program, err := argex.DefaultEnv().Compile(expr)
	if err != nil {
		b.Fatal(err)
	}
	for _, tc := range []equalityCase{
		{name: "float", items: floats, needle: float64(last)},
		{name: "json-number", items: numbers, needle: json.Number("255")},
		{name: "object", items: objects, needle: objects[last]},
	} {
		variables := map[string]any{"items": tc.items, "needle": tc.needle}
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := program.Eval(variables); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFunctions(b *testing.B) {
	variables := map[string]any{
		"status": " APPROVED ",
		"cost":   "0.025",
		"object": map[string]any{"b": 2.0, "a": 1.0},
		"prices": []any{3.0, 1.0, 2.0},
	}
	for _, source := range []string{
		`startsWith(status, " ")`,
		`lower(trim(status)) == "approved"`,
		`number(cost) * 1000`,
		`keys(object)`,
		`max(...prices)`,
	} {
		b.Run(source, func(b *testing.B) {
			expr, err := argex.Parse(source)
			if err != nil {
				b.Fatal(err)
			}
			program, err := argex.DefaultEnv().Compile(expr)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := program.Eval(variables); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
