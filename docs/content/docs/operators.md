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
| `??` | Missing/null fallback |
| `&&` | Boolean conjunction |
| `\|\|` | Boolean disjunction |
| `\|` | Collection operation (`map`, `filter`, `any`, `all`) |

`%` is a truncated remainder: the result takes the sign of the dividend, so `-5 % 3` returns `-2`. Like division, a zero divisor raises `ErrArithmetic`.

Binary operators associate left to right except `??`, which associates right to left. Ordering comparisons cannot be chained without parentheses. `|` is the lowest-precedence operator and chains left to right. A collection stage body extends up to the next `|`, so a nested pipeline must be parenthesized. A `let` shares the pipeline's level, so it too needs parentheses as an operand or stage body.

## Booleans

`&&`, `||`, and `!` require boolean operands, and `&&` and `||` short-circuit. Use a comparison to turn a number, string, object, or array into a condition:

```javascript
length(orders) > 0
```

## Equality and Ordering

Values of different types compare unequal, so `1 == "1"` is false. Objects and arrays compare structurally: `{id: 1} == {id: 1}` is true. Strings compare lexicographically. Missing equals missing and differs from null.

## Membership

`in` tests array membership using the same structural equality as `==`. It returns a boolean and stops at the first match. The right operand must be an array. Objects, strings, null, and missing values produce `ErrType`. Empty arrays return false. Like ordering comparisons, `in` requires parentheses when combined with another comparison.

```javascript
"domain:inventory" in value.tags || "domain:orders" in value.tags
item in allowed_items
!(item in excluded_items)
```

Use `includes(text, part)` to test a substring and `key in keys(object)` to test for an object key.
