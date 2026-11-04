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
| `ErrLimit` | Parsing or compilation limits exceeded |

The compiler rejects known type errors, even in unused branches. Runtime errors occur only in expressions that run. See [Type Checking]({{< relref "/docs/static-types" >}}) for the rules.

## Limits

Expressions may nest up to 256 levels. Each binary operator adds a level, but array elements, object members, `case` arms, and pipeline stages do not. Exceeding the limit raises `ErrLimit`.

Parenthesize pipelines used as case conditions if ambiguous colons cause `ErrLimit`.

Input maps and arrays must not contain cycles.
