---
title: "Missing Values"
weight: 50
---

# Missing Values

An unknown name returns `Missing{}`. Use `argex.IsMissing` in Go to test for it. Missing equals missing, but differs from null.

## Ordinary Access

Ordinary access raises `ErrMissing` for absent properties, missing intermediate values, and out-of-range indexes. Negative indexes are out of range. A property containing null returns null.

```javascript
customer.address.country
```

## Optional Access

`?.name` and `?.[index]` return missing if the value being accessed or its member is absent. Access on null still raises `ErrMissing`. Invalid container or index types raise `ErrType`. A member containing null returns null.

```javascript
customer?.address?.country
response?.choices?.[0]?.message?.content ?? ""
items?.[index] ?? 0
```

Optional indexing skips the index expression when the container is missing.

## Fallbacks

`??` evaluates its right operand only when its left operand is missing.

[Destructuring defaults](../let/#defaults) apply only to missing members.

## Missing Values in Results

Objects omit missing fields, but arrays keep missing elements. Missing cannot be encoded as JSON. Use `?? null` or another fallback before encoding:

```javascript
records | map record: record?.email ?? null
```
