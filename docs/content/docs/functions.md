---
title: "Functions"
weight: 70
---

# Functions

Only these named pure functions are available:

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

Call functions by name, as in `length(items)` or `startsWith(text, prefix)`. Unknown names and incorrect argument counts fail during compilation.

```javascript
lower(trim(payment.status)) == "approved"
join(split("inventory,orders", ","), " / ")
sum(results | map result: number(result.cost))
ceil(number(usage.tokens) / 1000)
keys(response) | any key: startsWith(key, "x-")
```

## Strings

String tests are case-sensitive and accept empty search strings. `includes` tests string substrings; use `in` for array membership. Case conversion uses locale-independent Unicode character mappings.

Splitting preserves empty fields. An empty separator splits into Unicode code points, with an empty input producing an empty array. Joining requires every element to be a string and returns an empty string for an empty array.

## Numbers

`round` rounds halfway values away from zero: `round(-1.5)` returns `-2` and `round(1.5)` returns `2`. `sum` requires finite numeric elements and returns `ErrArithmetic` if the running total overflows.

## Conversions

Conversions are explicit. `number` trims surrounding whitespace and requires the entire remaining string to be a valid JSON number. Empty strings, hexadecimal notation, leading zeros such as `"01"`, booleans, null, and non-finite results are rejected.

`string` preserves strings, formats finite numbers without unnecessary decimal digits, and converts booleans and null to `"true"`, `"false"`, and `"null"`. It rejects missing values, arrays, and objects. Invalid types or conversions return `ErrType`.

## Objects

Object inspection uses lexicographic property name order even for numeric-looking names. `keys({"2": 2, "10": 10})` returns `["10", "2"]`.
