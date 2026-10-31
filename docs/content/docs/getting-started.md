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
program, err := argex.DefaultEnv().Compile(expression)
if err != nil {
    return err
}
result, err := program.Eval(map[string]any{
    "items": []any{5, 20, 150},
})
// result: []any{"medium", "large"}
```

`Parse` checks syntax and returns an immutable `*Expression`. `DefaultEnv().Compile(expression)` resolves names and checks types, returning a `*Program`. Call `program.Eval(this)` for repeated evaluation and `program.StaticType()` for the result types inferred during compilation. `DefaultEnv()` supplies the shared builtin environment, while `NewEnv()` creates an empty environment. `Eval(source, this)` combines all three operations using the default environment.

Compiled programs are safe for concurrent use. Callers must not mutate bound values while they are in use or concurrently mutate input maps or arrays. Custom functions must also be safe for concurrent calls.

## Input

The input can be any JSON value, and `this` refers to it. `this` always means the input, including inside collection operation bodies:

```javascript
this | map item: item.id
this.items[0]
length(this)
```

When the input is an object, its top-level properties can also be read by bare name. Names resolve to local bindings first, then environment globals, then input properties. Use `this.name` for an input property shadowed by a global, such as `this.length`. A bare name with no binding or matching input property evaluates to a missing value. `this.name` follows the ordinary access rules, so an absent property fails instead.

`env.Names(expression)` returns the input properties an expression reads by bare name, sorted, so a caller can check them against a schema before evaluating. Local bindings, environment globals, and reads through `this` are excluded.

## Values

Values use Go's JSON representation: `map[string]any`, `[]any`, strings, booleans, `float64`, and `nil`. Go `int` and `json.Number` inputs are also accepted anywhere a number is expected, including arithmetic, comparisons, indexes, and functions. Computed numbers are always `float64`. Other Go types, such as `int64`, `float32`, `[]string`, or `map[string]string`, are rejected with `ErrType`. Numbers use finite IEEE 754 double precision. Division by zero and non-finite arithmetic results fail. Use `number(value)` and `string(value)` for explicit type conversions.

A bare name with no value produces `Missing{}`, which is distinct from JSON `null`. Use `IsMissing` to test for it. See [Missing Values]({{< relref "/docs/missing-values" >}}) for how missing values flow through expressions.

The builtin functions are pure and deterministic. Custom functions should preserve those properties and must not change or retain their argument slice.
