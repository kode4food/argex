# Argyll Expressions

![Build Status](https://github.com/kode4food/argex/actions/workflows/build.yml/badge.svg) [![Code Coverage](https://qlty.sh/gh/kode4food/projects/argex/coverage.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![Maintainability](https://qlty.sh/gh/kode4food/projects/argex/maintainability.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![GitHub](https://img.shields.io/github/license/kode4food/argex)](https://github.com/kode4food/argex/blob/main/LICENSE)

Argyll Expressions is a small, pure expression language for JSON values with familiar JavaScript-style operators and access syntax. Expressions support arithmetic, comparisons, optional access, object and array construction, conditional selection, and explicit collection operations.

```javascript
customer?.address?.country ?? "CH"
usage.input_tokens * input_price + usage.output_tokens * output_price

items | filter item: item.price > 10 | map item: {
    id: item.id,
    category: case {
        item.price > 100: "large",
        "medium",
    },
}
```

## Go API

```go
expression, err := argex.Parse(`
    items | filter item: item > 10 | map item: case {
        item > 100: "large",
        "medium",
    }
`)
if err != nil {
    return err
}
program, err := argex.Compile(expression)
if err != nil {
    return err
}
result, err := program(map[string]any{
    "items": []any{5, 20, 150},
})
// result: []any{"medium", "large"}
```

`Parse` returns an immutable `*Expression`, `Compile` returns a reusable `Program` that is safe for concurrent use, and `Eval(source, variables)` does all three for one-off expressions.

## Documentation

- [Getting started](https://kode4food.github.io/argex/docs/getting-started/) covers installation, the Go API, and value representation.
- [Collections](https://kode4food.github.io/argex/docs/collections/) explains `map`, `filter`, `any`, `all`, and pipelines.
- [Arrays and objects](https://kode4food.github.io/argex/docs/construction/) covers literals, spreads, and computed keys.
- [Conditional selection](https://kode4food.github.io/argex/docs/case/) describes `case`.
- [Missing values](https://kode4food.github.io/argex/docs/missing-values/) explains optional access, `??`, and missing versus null.
- [Operators](https://kode4food.github.io/argex/docs/operators/) lists precedence, equality, and membership rules.
- [Functions](https://kode4food.github.io/argex/docs/functions/) documents every built-in function.
- [Errors and limits](https://kode4food.github.io/argex/docs/errors/) lists error sentinels and evaluation limits.
- [Grammar](https://kode4food.github.io/argex/docs/grammar/) gives the complete syntax in EBNF.

## Development

```sh
make pre-commit
go test -race ./...
go test ./internal/engine -run '^$' -fuzz FuzzExpression -fuzztime=10s
go test -run '^$' -bench . -benchmem
```

The grammar is in [grammar.ebnf](docs/grammar.ebnf). The documentation site is a Hugo project in [docs](docs); run `hugo server` there to preview it.

The compliance corpus is in [testdata/compliance](testdata/compliance). Each JSON case specifies an expression, input variables, and an expected result, missing value, or error phase. `go test ./...` runs the corpus through both `Eval` and compiled programs, checks repeated execution, and verifies that inputs are unchanged.
