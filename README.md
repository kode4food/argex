# Argyll Expressions

<img src="./docs/img/argex-logo.png" align="right" height="100"/> ![Build Status](https://github.com/kode4food/argex/actions/workflows/build.yml/badge.svg) [![Code Coverage](https://qlty.sh/gh/kode4food/projects/argex/coverage.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![Maintainability](https://qlty.sh/gh/kode4food/projects/argex/maintainability.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![GitHub](https://img.shields.io/github/license/kode4food/argex)](https://github.com/kode4food/argex/blob/main/LICENSE)

Argyll Expressions is a language for querying and transforming JSON with JavaScript-style syntax.

Take a list of orders.

```json
{
  "orders": [
    {"id": "A", "paid": true,  "quantity": 3, "price": 3.95},
    {"id": "B", "paid": false, "quantity": 1, "price": 6.50}
  ]
}
```

| Expression | Result |
| --- | --- |
| `orders[0].id` | `"A"` |
| `orders[0].quantity * orders[0].price` | `11.85` |
| `orders[1].paid && orders[1].quantity > 0` | `false` |

To get the total for **every paid order**, pipe the array through `filter` and `map`.

```javascript
orders
    | filter order: order.paid
    | map order: {id: order.id, total: order.quantity * order.price}
```

```json
[{"id": "A", "total": 11.85}]
```

## Go API

Argyll Expressions is designed to work directly with the values Go’s `encoding/json` produces when it unmarshals JSON into an `any`.

```sh
go get github.com/kode4food/argex@latest
```

```go
total, err := argex.Eval(`quantity * price`, map[string]any{
    "quantity": 3, "price": 3.95,
})
// total: 11.85
```

[Compile and reuse a program](https://kode4food.github.io/argex/docs/getting-started/) or [bind Go functions](https://kode4food.github.io/argex/docs/functions/#environments-in-go) to make application-specific operations available to rules.

## Command Line

```sh
go install github.com/kode4food/argex/cmd/aq@latest
echo '{"quantity":3,"price":3.95}' | aq 'quantity * price'
# 11.85
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
- [Type checking](https://kode4food.github.io/argex/docs/static-types/) covers compile-time checks and result types.
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
