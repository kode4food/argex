---
title: "Conditional Selection"
weight: 40
---

# Conditional Selection

```javascript
case {
    score >= 0.9: "accept",
    score >= 0.5: "review",
    "reject",
}
```

A `case` has one or more `condition: result` arms and a required final bare fallback expression. Commas separate arms, and a trailing comma is allowed. Conditions must be booleans.

The first true condition selects its result. Later conditions and unselected results are not evaluated. The fallback is evaluated only if every condition is false.

Nested cases, arrays, and objects are ordinary result expressions:

```javascript
case {
    request.requires_reasoning: {model: "reasoning", max_tokens: 4000},
    {model: "fast", max_tokens: 1000},
}
```

## Matching a Subject

A subject between `case` and `{` turns each arm into a `value: result` pair. The subject is evaluated once, then compared with each value using `==`, in order. The first equal value selects its result, and the fallback is evaluated only if no value matches.

```javascript
case order.status {
    "paid": "ship",
    "pending": "wait",
    "cancelled",
}
```

Values are ordinary expressions, so they can read the input or compute a result. Later values and unselected results are not evaluated. Values of different types are never equal, so an arm whose value can never have the subject's type fails during parsing, as in `case 1 { "a": true, false }`.

The subject has the same precedence as an operand of `||`, so a pipeline subject needs parentheses. A subject that begins with `{` also needs parentheses, since `case {` always starts the boolean form. This selects a result from the names of the failed checks:

```javascript
case (checks | filter check: !check.passed | map check.name) {
    []: "approved",
    ["lint"]: "approved with warnings",
    "rejected",
}
```
