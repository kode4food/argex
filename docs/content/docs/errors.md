---
title: "Errors and Limits"
weight: 80
---

# Errors and Limits

## Errors

Use `errors.Is` with these sentinels:

| Sentinel | Raised for |
| --- | --- |
| `ErrSyntax` | Invalid syntax, with the character offset of the error in the source |
| `ErrFunction` | Unknown function names and incorrect argument counts |
| `ErrType` | Operands, receivers, indexes, keys, or arguments of the wrong type |
| `ErrMissing` | Ordinary access to a missing property, value, or index |
| `ErrArithmetic` | Division by zero and non-finite results |
| `ErrLimit` | Expressions nested beyond the limit |

An `ErrType` mismatch between operands whose types are known without input fails during compilation, as described in [Type Checking]({{< relref "/docs/static-types" >}}). Call arguments after the first spread are checked against function parameters at evaluation because their positions depend on the spread; invalid expressions within those arguments still fail compilation. Unknown direct callees and invalid fixed argument counts in direct global calls also fail during compilation, including in an unchosen branch. Calls through computed values and counts that depend on spread lengths are checked at evaluation, even for literal spreads such as `abs(...[1, 2])`. Runtime errors occur only in expressions actually evaluated.

## Limits

Parsing and compilation limit expression nesting to 256 levels and return `ErrLimit` beyond that. Each binary operator in a chain adds a level, so `a + b + ...` with more than 256 operators exceeds the limit. Array elements, object members, `case` arms, and `|` stages do not add levels.

Inputs must be JSON-shaped, acyclic values.
