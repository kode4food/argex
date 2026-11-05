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

A `case` returns the result of the first true condition, or the required final fallback. Conditions must be booleans. Only the selected result is evaluated. Separate arms with commas, including an optional trailing comma.

## Matching a Subject

Put an expression after `case` to match its value against the arms. It is evaluated once. The first match using `==` selects the result, or the fallback if none match.

```javascript
case order.status {
    "paid": "ship",
    "pending": "wait",
    "cancelled",
}
```

Arm values can be expressions. A value with an incompatible known type fails compilation, as in `case 1 { "a": true, false }`.

Parenthesize a pipeline or object used as the subject. This example matches the names of failed checks:

```javascript
case (checks | filter check: !check.passed | map check.name) {
    []: "approved",
    ["lint"]: "approved with warnings",
    "rejected",
}
```
