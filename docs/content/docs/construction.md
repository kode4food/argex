---
title: "Arrays and Objects"
weight: 30
---

# Arrays and Objects

## Literals

Object keys can be identifiers or double-quoted strings, and trailing commas are allowed. Duplicate literal keys are a syntax error.

```javascript
{
  customer_id: customer.id,
  country: customer?.address?.country ?? "CH",
  channels: ["email", "sms"],
}
```

`{reason}` is shorthand for `{reason: reason}`. Reserved names such as `this` cannot use this shorthand.

Object construction omits fields whose value is missing. Arrays retain missing elements.

## Spreads

Use `...` to copy array elements into an array or object entries into an object. Later entries override earlier keys. Null and missing values contribute nothing. Other types and mismatched containers raise `ErrType`.

```javascript
[first, ...rest, last]

{
  ...defaults,
  ...overrides,
  id: record.id,
}
```

## Computed Keys

A key in brackets uses an expression's value. It must be a string or evaluation raises `ErrType`.

```javascript
{[record.kind]: record.value}
entries | map e: {[e.name]: e.value}
```

Computed keys follow the same rule as spreads: the last entry wins.
