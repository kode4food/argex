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
