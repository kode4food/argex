---
title: "Getting Started"
weight: 10
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
program, err := argex.Compile(expression)
if err != nil {
    return err
}
result, err := program(map[string]any{
    "items": []any{5, 20, 150},
})
// result: []any{"medium", "large"}
```

`Parse` returns an immutable `*Expression`, and `Compile` returns a `Program`, defined as `func(map[string]any) (any, error)`. Call that function directly and reuse it for repeated evaluation. `Eval(source, variables)` combines all three operations for one-off expressions.

Compiled functions are safe for concurrent use. Callers must not concurrently mutate their input maps or arrays.

## Values

Values use Go's JSON representation: `map[string]any`, `[]any`, strings, booleans, `float64`, and `nil`. Go `int` and `json.Number` inputs are also accepted anywhere a number is expected, including arithmetic, comparisons, indexes, and functions; computed numbers are always `float64`. Other Go types, such as `int64`, `float32`, `[]string`, or `map[string]string`, are rejected with `ErrType`. Numbers use finite IEEE 754 double precision. Division by zero and non-finite arithmetic results fail. Use `number(value)` and `string(value)` for explicit type conversions.

An absent variable produces `Missing{}`, which is distinct from JSON `null`. Use `IsMissing` to test for it. See [Missing Values]({{< relref "/docs/missing-values" >}}) for how missing values flow through expressions.

Expressions are pure and deterministic: their results depend only on the supplied values, and functions never change their arguments.
