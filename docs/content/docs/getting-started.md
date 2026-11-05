---
title: "Getting Started"
weight: 10
aliases:
  - /docs/
---

# Getting Started

## Install

```sh
go get github.com/kode4food/argex@latest
```

## Parse, Compile, and Run

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

`Parse` checks the expression's syntax. `Compile` resolves names and checks types. Call `program.Eval(input)` to reuse the compiled program, or use `argex.Eval(source, input)` for a single evaluation. `program.StaticType()` reports the possible result types.

Programs support concurrent evaluation. Do not mutate bound values while in use or input maps and arrays during evaluation. Custom functions must be safe for concurrent calls.

## Input

`this` refers to the input, even inside a collection operation:

```javascript
this | map item: item.id
this.items[0]
length(this)
```

For object input, `name` reads the top-level property unless a local binding or environment global has that name. Local bindings take precedence over globals. Use `this.name` to read the property directly. An unknown name returns missing, but accessing an absent property with `this.name` raises `ErrMissing`.

Use `env.Names(expression)` to check bare input names against a schema. It returns sorted names, excluding local bindings, globals, and access through `this`.

## Values

Argyll Expressions is designed to work directly with the values Go’s `encoding/json` produces when it unmarshals JSON into an `any`. Objects become `map[string]any`, arrays become `[]any`, and JSON numbers become `float64` by default. Strings, booleans, and null become `string`, `bool`, and `nil`. Arithmetic also accepts `int` and `json.Number` and produces `float64` results.

See [operators]({{< relref "/docs/operators" >}}) for decimal precision and rounding.

Use `argex.IsMissing` to distinguish `Missing{}` from JSON `null`. See [Missing Values]({{< relref "/docs/missing-values" >}}) for access and fallback rules.
