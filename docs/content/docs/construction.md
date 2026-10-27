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

Object construction omits fields whose value is missing. Arrays retain missing elements.

## Spreads

Array and object literals may include a `...` spread. Spreading an array into an array inserts its elements. Spreading an object into an object copies its entries. Later entries win, so a field or spread overrides an earlier key. Spreading a missing or null value contributes nothing. Spreading any other type, or spreading an array into an object or an object into an array, is an error.

```javascript
[first, ...rest, last]
{...defaults, ...overrides, id: record.id}
```

Each spread produces a fresh value and never mutates its source. Entries drawn from a spread are omitted when their value is missing.

## Computed Keys

A key in brackets is computed from an expression. It must evaluate to a string. Any other value, including missing or null, is an `ErrType`.

```javascript
{[record.kind]: record.value}
entries | map e: {[e.name]: e.value}
```

Entries apply in the order written, so a computed key and a field with the same name follow the same later-wins rule as spreads.
