---
title: "Operators"
weight: 60
---

# Operators

## Precedence

Highest first:

| Operators | Meaning |
| --- | --- |
| `.`, `?.`, `[]`, `?.[]`, calls | Access and calls |
| `!`, unary `+`, unary `-` | Boolean negation and numeric signs |
| `*`, `/`, `%` | Multiplication, division, remainder |
| `+`, `-` | Addition and subtraction, with `+` also joining two strings |
| `<`, `<=`, `>`, `>=`, `in` | Numeric or string ordering and array membership |
| `==`, `!=` | Equality by type and contents |
| `??` | Missing-value fallback |
| `&&` | Boolean conjunction |
| `\|\|` | Boolean disjunction |
| `\|` | Collection operation (`map`, `filter`, `any`, `all`) |

Arithmetic uses each number's shortest decimal representation: `0.1 + 0.2` returns `0.3`. Inputs and each result are limited to `float64` precision, so large values and long fractions can lose digits. Use `sum` for lists to avoid rounding intermediate totals. Division by zero and overflow raise `ErrArithmetic`.

`%` is a truncated remainder: the result takes the sign of the dividend, so `-5 % 3` returns `-2` and `1.1 % 0.1` returns `0`. Like division, a zero divisor raises `ErrArithmetic`.

Binary operators group left to right, except `??`, which groups right to left. Parenthesize chained comparisons, nested pipelines, and `let` used as an operand or collection body.

## Booleans

`&&`, `||`, and `!` require booleans. `&&` skips the right operand when the left is false. `||` skips it when the left is true.

## Equality and Ordering

Values of different types compare unequal, so `1 == "1"` is false. Objects and arrays compare structurally: `{id: 1} == {id: 1}` is true. Strings compare lexicographically. Missing equals missing and differs from null.

## Membership

`in` checks whether an array contains a value, using `==` to compare elements. It returns false for an empty array and raises `ErrType` if the right operand is not an array. Parenthesize `in` when combining it with another comparison.

```javascript
"domain:inventory" in value.tags || "domain:orders" in value.tags
item in allowed_items
!(item in excluded_items)
```

Use `includes(text, part)` to test a substring and `key in keys(object)` to test for an object key.
