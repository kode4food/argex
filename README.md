# Argyll Expressions

<img src="./docs/img/argex-logo.png" align="right" height="100"/> ![Build Status](https://github.com/kode4food/argex/actions/workflows/build.yml/badge.svg) [![Code Coverage](https://qlty.sh/gh/kode4food/projects/argex/coverage.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![Maintainability](https://qlty.sh/gh/kode4food/projects/argex/maintainability.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![GitHub](https://img.shields.io/github/license/kode4food/argex)](https://github.com/kode4food/argex/blob/main/LICENSE)

Argyll Expressions is a language for querying and transforming JSON with JavaScript-style syntax. Use `this` to refer to the input. If the input is an object, you can also refer to its properties by name.

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
program, err := argex.DefaultEnv().Compile(expression)
if err != nil {
    return err
}
result, err := program.Eval(map[string]any{
    "items": []any{5, 20, 150},
})
// result: []any{"medium", "large"}
```

Compile once and call `program.Eval(input)` repeatedly or concurrently. For a single evaluation, use `argex.Eval(source, input)`. See [Getting started](https://kode4food.github.io/argex/docs/getting-started/) for input types and concurrency requirements.

## Command Line

```sh
go install github.com/kode4food/argex/cmd/aq@latest
echo '{"items":[1,20,150]}' | aq 'items | filter i: i > 10'
```

## Documentation

- [Getting started](https://kode4food.github.io/argex/docs/getting-started/) covers installation, the Go API, and value representation.
- [Command line](https://kode4food.github.io/argex/docs/command-line/) covers the `aq` tool and its flags.
- [Collections](https://kode4food.github.io/argex/docs/collections/) explains `map`, `filter`, `any`, `all`, and pipelines.
- [Arrays and objects](https://kode4food.github.io/argex/docs/construction/) covers literals, spreads, and computed keys.
- [Conditional selection](https://kode4food.github.io/argex/docs/case/) describes `case`, with conditions or a subject to match.
- [Local bindings](https://kode4food.github.io/argex/docs/let/) covers `let`, destructuring, and defaults.
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

The grammar is in [grammar.ebnf](docs/grammar.ebnf).

Add language behavior tests to [testdata/compliance](testdata/compliance). Each JSON case specifies an expression, input, and expected result or error. Run them with `go test ./...`.
