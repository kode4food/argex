---
title: "Type Checking"
weight: 75
---

# Type Checking

`program.StaticType()` returns the `TypeSet` inferred during compilation without evaluating the program or repeating inference. Operators and `case` arms whose result type does not depend on the input, such as literals, comparisons, arithmetic, and builtin calls, narrow exactly. Reads from input and calls without a known return type include `TypeAny`. A name bound by `let` carries its value's type, and an environment global carries the type of its bound value.

```go
expression, _ := argex.Parse(`case { str == "hello": true, -1 }`)
env := argex.DefaultEnv()
program, err := env.Compile(expression)
result := program.StaticType() // bool|number

expression, _ = argex.Parse(`str == "hello"`)
program, err = env.Compile(expression)
program.StaticType().Is(argex.TypeBool) // true
```

`TypeSet` is a bitset of `TypeTag` values: `TypeBool`, `TypeNumber`, `TypeString`, `TypeArray`, `TypeObject`, `TypeNull`, `TypeMissing`, `TypeFunction`, and `TypeAny`. `Has` tests membership, `Is` tests an exact match, `Known` reports whether the set excludes `TypeAny`, and `String` renders it as `"bool|number"`.

## Type Errors

`env.Compile` rejects any operator, global call argument with a known position, or condition whose type is fully known and can never succeed, such as `length(-1)`, `true + 1`, or `false || "hello"`. `Parse` checks syntax only. Operands that read the input are checked when the expression is evaluated. Arguments after a spread are checked against function parameters at evaluation; invalid expressions within those arguments, such as `true + 1`, still fail compilation.

The check covers the whole expression, including branches that would never run. `true || (1 && 2)` is a type error even though `||` would skip its right side.

```go
expression, _ := argex.Parse(`false || "hello"`)
_, err := argex.DefaultEnv().Compile(expression)
// ErrType: || right operand must be bool, not string
```
