---
title: "Type Checking"
weight: 75
---

# Type Checking

`program.StaticType()` reports the possible result types without running the expression. Bindings keep their values' types. Unknown types, including input properties, use `TypeAny`.

```go
expression, _ := argex.Parse(`case { str == "hello": true, -1 }`)
env := argex.DefaultEnv()
program, err := env.Compile(expression)
result := program.StaticType() // bool|number

expression, _ = argex.Parse(`str == "hello"`)
program, err = env.Compile(expression)
program.StaticType().Is(argex.TypeBool) // true
```

`TypeSet` can contain `TypeBool`, `TypeNumber`, `TypeString`, `TypeArray`, `TypeObject`, `TypeNull`, `TypeMissing`, `TypeFunction`, and `TypeAny`.

| Method | Result |
| --- | --- |
| `Has(type)` | Whether the set includes the type |
| `Is(type)` | Whether the set contains only that type |
| `Known()` | Whether the set excludes `TypeAny` |
| `String()` | A label such as `"bool|number"` |

## Type Errors

`Compile` rejects known type errors such as `length(-1)` and `true + 1`, even in unused branches. Input-dependent types are checked during evaluation.

Direct global calls also check function names and fixed argument counts during compilation. Calls through other expressions check counts during evaluation. Spreads defer count checks and parameter checks for arguments after the first spread, but invalid expressions such as `true + 1` still fail compilation.

```go
expression, _ := argex.Parse(`false || "hello"`)
_, err := argex.DefaultEnv().Compile(expression)
// ErrType: || right operand must be bool, not string
```
