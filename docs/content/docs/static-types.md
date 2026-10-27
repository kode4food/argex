---
title: "Static Type Checking"
weight: 75
---

# Static Type Checking

`expression.StaticType()` infers the set of types an expression's result can take without evaluating it. Operators and `case` arms whose result type doesn't depend on the input - literals, comparisons, arithmetic, builtin calls - narrow exactly. Anything that reads the input - a bare name, `this`, member or index access, or a call to a function with no fixed return type - widens the result to `TypeAny`, since argex has no schema for `this` to check that against.

```go
expression, _ := argex.Parse(`case { str == "hello": true, -1 }`)
expression.StaticType() // TypeSet: bool|number

expression, _ = argex.Parse(`str == "hello"`)
expression.StaticType().Is(argex.TypeBool) // true
```

`TypeSet` is a bitset of `TypeTag` values: `TypeBool`, `TypeNumber`, `TypeString`, `TypeArray`, `TypeObject`, `TypeNull`, `TypeMissing`, and `TypeAny`. `Has` tests membership, `Is` tests an exact match, `Known` reports whether the set excludes `TypeAny`, and `String` renders it as `"bool|number"`.

## Check

`expression.Check()` statically validates every operator and builtin call argument whose type is fully known without the input, catching a mismatch that can never succeed - `length(-1)`, `true + 1`, `false || "hello"` - without evaluating anything. It returns `nil` when nothing is provably wrong, including when the expression depends entirely on runtime input.

`Check` is separate from `Compile`: compiling never fails for a mismatch `Check` would catch, since some of those mismatches only become real once real input flows through `this`. Call `Check` explicitly wherever failing fast, with a precise message, matters more than deferring to `ErrType` at evaluation time - a CLI compiling a user-supplied expression, for instance.

```go
expression, _ := argex.Parse(`false || "hello"`)
expression.Check() // ErrType: || right operand must be bool, not string
```
