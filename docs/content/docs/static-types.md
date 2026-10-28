---
title: "Type Checking"
weight: 75
---

# Type Checking

`expression.StaticType()` infers the set of types an expression's result can take without evaluating it. Operators and `case` arms whose result type doesn't depend on the input, such as literals, comparisons, arithmetic, and builtin calls, narrow exactly. Anything that reads the input, such as a bare name, `this`, member or index access, or a call to a function with no fixed return type, widens the result to `TypeAny`, since argex has no schema for `this` to check that against.

```go
expression, _ := argex.Parse(`case { str == "hello": true, -1 }`)
expression.StaticType() // TypeSet: bool|number

expression, _ = argex.Parse(`str == "hello"`)
expression.StaticType().Is(argex.TypeBool) // true
```

`TypeSet` is a bitset of `TypeTag` values: `TypeBool`, `TypeNumber`, `TypeString`, `TypeArray`, `TypeObject`, `TypeNull`, `TypeMissing`, and `TypeAny`. `Has` tests membership, `Is` tests an exact match, `Known` reports whether the set excludes `TypeAny`, and `String` renders it as `"bool|number"`.

## Type Errors

`argex.Parse` rejects any operator, builtin call argument, or condition whose type is fully known and can never succeed, such as `length(-1)`, `true + 1`, or `false || "hello"`. Operands that read the input are checked when the expression is evaluated.

The check covers the whole expression, including branches that would never run. `true || (1 && 2)` is a type error even though `||` would skip its right side.

```go
_, err := argex.Parse(`false || "hello"`)
// ErrType: || right operand must be bool, not string
```
