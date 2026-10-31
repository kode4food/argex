---
title: "Functions"
weight: 70
---

# Functions

The default environment provides these pure functions:

| Function | Behavior |
| --- | --- |
| `length(value)` | Number of Unicode code points, array elements, or object members |
| `abs(number)` | Absolute value |
| `min(number, ...)` | Minimum of one or more numbers |
| `max(number, ...)` | Maximum of one or more numbers |
| `sum(array)` | Sum of numeric elements, or zero for an empty array |
| `round(number)` | Nearest integer, with halfway values rounded away from zero |
| `floor(number)` | Greatest integer less than or equal to the number |
| `ceil(number)` | Least integer greater than or equal to the number |
| `startsWith(text, prefix)` | Whether text begins with the prefix |
| `endsWith(text, suffix)` | Whether text ends with the suffix |
| `includes(text, part)` | Whether text contains the substring |
| `lower(text)` | Unicode lowercase form |
| `upper(text)` | Unicode uppercase form |
| `trim(text)` | Remove surrounding Unicode whitespace |
| `split(text, separator)` | Array of strings separated by the literal separator |
| `join(array, separator)` | Join an array of strings with the separator |
| `number(value)` | Finite number from a number or a string containing a JSON number |
| `string(value)` | String form of a string, finite number, boolean, or null |
| `keys(object)` | Array of property names in lexicographic order |
| `values(object)` | Array of values in the same property name order as `keys` |

Functions are ordinary values. Call them by name, bind them with `let`, select them with `case`, or read them from an array or object:

```javascript
let size = length: size(items)
(case { absolute: abs, round })(amount)
[min, max][1](1, 2, 3)
{ size: length }.size(items)
```

Names resolve to local bindings first, then environment globals, then input properties. A global such as `length` shadows an input property of the same name. Use `this.length` to read that property. Unknown names used directly as callees fail during compilation with `ErrFunction`. Calling a value known to be a nonfunction fails during compilation with `ErrType`.

Calls evaluate the callee first and arguments from left to right. Direct calls to global functions check argument counts during compilation. Calls through local bindings or other expressions check them at evaluation. Incorrect counts return `ErrFunction`, and a computed callee that is not a function returns `ErrType`.

```javascript
lower(trim(payment.status)) == "approved"
join(split("inventory,orders", ","), " / ")
sum(results | map result: number(result.cost))
ceil(number(usage.tokens) / 1000)
keys(response) | any key: startsWith(key, "x-")
```

Use `...` to spread an array into the arguments, as in `max(...prices)` or `max(0, ...prices)`. Spreading a missing or null value contributes no arguments, and spreading any other non-array value is an `ErrType`. Counts that depend on a spread's length are checked at evaluation and fail with `ErrFunction`. A direct global call with too many fixed arguments, such as `abs(1, 2, ...[])`, fails during compilation. Argument types after the first spread are checked at evaluation.

Functions compare by identity, so `length == length` is true and `length == abs` is false. They can appear in arrays and objects, but serializing a function to JSON returns `ErrType`.

## Environments in Go

`argex.DefaultEnv()` returns the shared environment containing the builtin functions. `argex.NewEnv()` creates an empty environment. `Bind` returns a new environment with the added or replaced binding, leaving the original unchanged:

```go
env, err := argex.DefaultEnv().Bind("limit", 10)
if err != nil {
    return err
}
expression, err := argex.Parse("amount < limit")
if err != nil {
    return err
}
program, err := env.Compile(expression)
```

Bind a `*argex.Function` to add a custom function. Set `Call` to the callback and `Arity` to the required argument count. With `Variadic: true`, `Arity` is the minimum count. Optional `Params` entries describe argument types for static checking, with the last entry repeated for later arguments. `Result` describes the return type, with zero meaning unknown. The callback must validate runtime argument types, propagate errors, and avoid changing or retaining its argument slice.

Bindings are immutable, and environments can be shared across compilation calls. Bound values are shared with programs. Do not mutate them while they are in use, and make custom callbacks safe for concurrent evaluation. Calling `Compile` on a nil environment returns `ErrFunction` for a valid expression.

## Strings

String tests are case-sensitive and accept empty search strings. `includes` tests string substrings. Use `in` for array membership. Case conversion uses locale-independent Unicode character mappings.

Splitting preserves empty fields. An empty separator splits into Unicode code points, with an empty input producing an empty array. Joining requires every element to be a string and returns an empty string for an empty array.

## Numbers

`round` rounds halfway values away from zero: `round(-1.5)` returns `-2` and `round(1.5)` returns `2`. `sum` requires finite numeric elements and returns `ErrArithmetic` if the running total overflows.

## Conversions

Conversions are explicit. `number` trims surrounding whitespace and requires the entire remaining string to be a valid JSON number. Empty strings, hexadecimal notation, leading zeros such as `"01"`, booleans, null, and non-finite results are rejected.

`string` preserves strings, formats finite numbers without unnecessary decimal digits, and converts booleans and null to `"true"`, `"false"`, and `"null"`. It rejects missing values, arrays, and objects. Invalid types or conversions return `ErrType`.

## Objects

Object inspection uses lexicographic property name order even for numeric-looking names. `keys({"2": 2, "10": 10})` returns `["10", "2"]`.
