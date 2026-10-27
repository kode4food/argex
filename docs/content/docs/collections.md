---
title: "Collections"
weight: 20
---

# Collections

## Map and Filter

`| map` and `| filter` bind each element of their receiver to a name. The expression after `:` is evaluated in that scope. It can read outer bindings, the input as `this`, and the input's properties by name. An inner binding with the same name shadows the outer one.

```javascript
orders | map order: order.total
orders | filter order: order.status == "paid"
```

`| map` preserves order. `| filter` requires a boolean predicate and preserves the relative order of retained elements. Both produce an array, including for empty inputs.

## Keys and Indexes

A binding may name two identifiers, `key, value`, to also bind each element's position: the numeric index for an array receiver, or the property name for an object receiver. Object properties are visited in sorted key order. Both forms accept arrays and objects. Any other receiver, including null or a missing value, raises `ErrType`.

```javascript
items | map index, item: {position: index, value: item}
totals | filter key, amount: amount > 100
```

In a later stage of a pipeline, the index is the position in the array that the previous stage produced, so `items | filter x: x.active | map i, x: i` numbers only the retained elements.

## Binding Forms

Every stage of `map`, `filter`, `any`, and `all` accepts the same binding forms:

| Form | Binds |
| --- | --- |
| `key, value: body` | The key or index and the incoming value |
| `value: body` | The incoming value, the same as `_, value: body` |
| `key, _: body` | The key or index only |
| `body` | Nothing, the same as `_, _: body` |

`_` discards a binding. Any stage may omit its bindings:

```javascript
items | map 0
items | filter item: item.active | any item.price > 100
```

## Pipelines

`|` is the lowest-precedence operator and chains left to right, so `a | filter x: p | map x: q` applies the `map` to the result of the `filter` and needs no parentheses.

```javascript
orders | filter order: order.status == "paid" | map order: order.id
```

Each stage opens a nested scope. Its body sees the bindings of earlier stages in the same pipeline, and a binding with the same name shadows an earlier one:

```javascript
[1, 2] | map x: x * 10 | map x + 1     // [2, 3]: x is the original element
[1, 2] | map x: x * 10 | map x: x + 1  // [11, 21]: x is the mapped value
```

With an object receiver:

```javascript
prices | filter name, price: price > 10
       | map {[name]: price}
       | filter index, entry: entry[name] > 20 && index > 0
       | map name
```

`name` and `price` are the original property name and value. `entry` is the mapped object, and `index` is its position in the second `filter`.

Bindings inside a parenthesized pipeline stay inside it.

A pipeline runs element by element: each element passes through every stage before the next element starts. Results are the same as running each stage over a complete array, but when a pipeline ends in `any` or `all`, evaluation stops as soon as the answer is known. Elements after that point never reach the earlier stages, so errors they would have raised are not reported:

```javascript
[1, "a"] | map x: x + 1 | any y: y > 0
```

This returns `true` without evaluating `"a" + 1`.

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

`| any` tests whether at least one element satisfies a condition. `| all` tests whether every element satisfies it. Both require boolean conditions, return booleans, and stop as soon as their result is known. For an empty collection, `any` returns false and `all` returns true without evaluating the condition. The same binding and grouping rules apply as for `map` and `filter`. At the end of a pipeline, stopping early also stops the earlier stages, as described under [Pipelines](#pipelines).

```javascript
tags | any tag: startsWith(tag, "market:")
results | all result: result.status == "approved"
(results | any result: result.status == "approved") && enabled
```

## Pipes, Properties, and OR

`.` and `?.` access properties. `|` introduces collection operations. `response.map`, `response.filter`, and `response.any` read properties with those names. `||` means boolean OR.

Use `??` to supply a fallback before mapping an optional array:

```javascript
(response?.items ?? []) | map item: item.id
```
