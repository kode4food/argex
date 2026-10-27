---
title: "Collections"
weight: 20
---

# Collections

## Map and Filter

`| map` and `| filter` bind each element of their receiver to a name. The expression after `:` is evaluated in that scope. It can read outer bindings and input variables. An inner binding with the same name shadows the outer one.

```javascript
orders | map order: order.total
orders | filter order: order.status == "paid"
```

`| map` preserves order. `| filter` requires a boolean predicate and preserves the relative order of retained elements. Both produce an array, including for empty inputs.

## Keys and Indexes

A binding may name two identifiers, `key, value`, to also bind each element's position: the numeric index for an array receiver, or the property name for an object receiver, visited in sorted key order. The single-name form requires an array receiver; the two-name form accepts arrays and objects.

```javascript
items | map index, item: {position: index, value: item}
totals | filter key, amount: amount > 100
```

## Pipelines

`|` is the lowest-precedence operator and chains left to right, so `a | filter x: p | map x: q` applies the `map` to the result of the `filter` and needs no parentheses.

```javascript
orders | filter order: order.status == "paid" | map order: order.id
```

The binding body after `:` stops at the next `|`. To nest a pipeline inside a body, parenthesize it:

```javascript
customers | map customer: {
    id: customer.id,
    totals: (customer.orders | filter order: order.paid) | map order: order.total,
}
```

Pipelines can also appear as ordinary values, such as object fields:

```javascript
{
    first: list1 | map item: item.name,
    second: list2 | filter item: item.available,
}
```

## Any and All

`| any` tests whether at least one element satisfies a condition. `| all` tests whether every element satisfies it. Both require boolean conditions, return booleans, and stop as soon as their result is known. For an empty collection, `any` returns false and `all` returns true without evaluating the condition. The same binding and grouping rules apply as for `map` and `filter`.

```javascript
tags | any tag: startsWith(tag, "market:")
results | all result: result.status == "approved"
(results | any result: result.status == "approved") && enabled
```

## Pipes, Properties, and OR

The receiver must be an array, or an object with the two-name form. Any other receiver, including null or a missing value, raises `ErrType`.

`.` and `?.` access properties. `|` introduces collection operations. `response.map`, `response.filter`, and `response.any` read properties with those names. `||` means boolean OR.

Use `??` to supply a fallback before mapping an optional array:

```javascript
(response?.items ?? []) | map item: item.id
```
