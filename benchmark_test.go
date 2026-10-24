package argex_test

import (
	"testing"

	"github.com/kode4food/argex"
)

type benchmarkCase struct {
	name   string
	source string
}

const benchmarkCollectionSize = 256

func BenchmarkExpressions(b *testing.B) {
	variables := map[string]any{
		"a":     map[string]any{"b": map[string]any{"c": true}},
		"items": []any{1, 5, 11, 101},
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
			program, err := argex.Compile(expr)
			if err != nil {
				b.Fatal(err)
			}
			b.Run("compiled", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, err := program(variables)
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

func BenchmarkWorkloads(b *testing.B) {
	items := make([]any, benchmarkCollectionSize)
	for i := range items {
		items[i] = float64(i)
	}
	variables := map[string]any{
		"items":         items,
		"input_tokens":  1000,
		"output_tokens": 250,
		"input_price":   0.01,
		"output_price":  0.03,
		"tax":           0.2,
		"rate":          2,
		"offset":        3,
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
		{
			name:   "nested",
			source: `items | map item: [1,2,3,4] | map inner: item + inner`,
		},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			expr, err := argex.Parse(tc.source)
			if err != nil {
				b.Fatal(err)
			}
			program, err := argex.Compile(expr)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				_, err := program(variables)
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
				program, err := argex.Compile(expr)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				for b.Loop() {
					if _, err := program(variables); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
