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

You can call functions by name, assign them to bindings, or select them from other values:

```javascript
let size = length: size(items)
(case { absolute: abs, round })(amount)
[min, max][1](1, 2, 3)
{ size: length }.size(items)
```

A builtin such as `length` takes precedence over an input property with that name. Use `this.length` to read the property.

Calls evaluate the function first, then its arguments from left to right. An unknown function or wrong argument count raises `ErrFunction`. Calling a nonfunction raises `ErrType`. See [Type Checking](../static-types/) for when these errors are detected.

```javascript
lower(trim(payment.status)) == "approved"
join(split("inventory,orders", ","), " / ")
sum(results | map result: number(result.cost))
ceil(number(usage.tokens) / 1000)
keys(response) | any key: startsWith(key, "x-")
```

Use `...` to pass an array's elements as arguments, as in `max(0, ...prices)`. Missing and null contribute no arguments. Other non-array values raise `ErrType`.

Functions compare by identity: `length == length` is true and `length == abs` is false. Functions cannot be serialized to JSON.

## Environments in Go

`argex.DefaultEnv()` provides the builtins. `argex.NewEnv()` creates an empty environment. `Bind` adds or replaces a binding in a new environment without changing the original:

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

Bind a `*argex.Function` to add a custom function:

| Field | Purpose |
| --- | --- |
| `Call` | Go callback |
| `Arity` | Required argument count, or minimum count when `Variadic` is true |
| `Params` | Optional argument types for compile-time checks. The last type applies to remaining arguments |
| `Result` | Return type, or zero if unknown |

Callbacks must validate argument types and propagate errors. They must not modify or retain the argument slice. Make callbacks safe for concurrent use, and do not mutate bound values while programs use them.

## Strings

String tests are case-sensitive and accept empty search strings. Use `includes` for substrings and `in` for array membership. Case conversion follows Unicode rules without locale-specific mappings.

`split` preserves empty fields. With an empty separator it splits into Unicode code points, returning an empty array for empty input. `join` requires string elements and returns `""` for an empty array.

## Numbers

`round` rounds halfway values away from zero: `round(-1.5)` returns `-2` and `round(1.5)` returns `2`.

`sum([0.1, 0.2, 0.3])` returns `0.6`. It adds decimals without rounding intermediate totals, then returns a `float64`. Elements must be finite numbers. Only overflow of the final total raises `ErrArithmetic`.

## Conversions

`number` accepts a finite number or a string containing a JSON number, with optional surrounding whitespace. It returns a `float64`. Empty strings, hexadecimal notation, leading zeros such as `"01"`, booleans, and null are rejected.

`string` accepts strings, finite numbers, booleans, and null. Numbers omit unnecessary decimal digits. Invalid conversions return `ErrType`.

## Objects

`keys` sorts names as strings, so `keys({"2": 2, "10": 10})` returns `["10", "2"]`. `values` uses the same order.
