---
title: "Missing Values"
weight: 50
---

# Missing Values

A bare name with no matching input property, or any bare name when the input is not an object, produces `Missing{}`, distinguishable from explicit JSON `null` using `IsMissing`. Missing equals missing and differs from null.

## Ordinary Access

Reading a missing property, a missing intermediate value, or an out-of-range index through ordinary access fails with `ErrMissing`. Negative indexes are always out of range. A present property whose value is null returns null.

```javascript
customer.address.country
```

## Optional Access

`?.name` and `?.[index]` return missing for a null or absent receiver or an absent member. A wrong receiver type or an invalid index type produces `ErrType`.

```javascript
customer?.address?.country
response.choices?.[0]?.message?.content ?? ""
values?.[index] ?? 0
```

Optionality applies to each marked access: use `a?.b?.c` when both accesses may be absent. Optional indexing skips evaluation of the index when the receiver is null or missing.

## Fallbacks

`??` evaluates its right operand only when its left operand is missing or null. Zero, false, and the empty string are preserved.

## Missing Values in Results

Object construction omits fields whose value is missing. Arrays retain missing elements. A missing value cannot be JSON-encoded: explicitly use `?? null` or another fallback where a serializable value is required.

```javascript
records | map record: record?.email ?? null
```
