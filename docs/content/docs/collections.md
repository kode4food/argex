---
title: "Collections"
weight: 20
---

# Collections

## Map and Filter

`map` transforms each element. `filter` keeps elements that match a condition. Name the element before `:` and write the expression after it:

```javascript
orders | map order: order.total
orders | filter order: order.status == "paid"
```

Both return arrays and preserve element order. A `filter` condition must return a boolean.

To map an array that may be absent, supply an empty array as the fallback.

```javascript
(response?.items ?? []) | map item: item.id
```

## Keys and Indexes

Use `key, value` to access an array index or object property name. Objects are visited in sorted key order. Collection operations accept arrays and objects. Other inputs, including null and missing, raise `ErrType`.

```javascript
items | map index, item: {
    position: index,
    value: item,
}

totals | filter key, amount: amount > 100
```

Indexes refer to the previous stage's output. For example, `items | filter x: x.active | map i, x: i` numbers the retained elements from zero.

## Binding Forms

Every stage of `map`, `filter`, `any`, `all`, and `reduce` accepts the same binding forms:

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

The value binding can also be an object or array pattern, using the same [destructuring rules](../let/#destructuring) as `let`:

```javascript
orders | map {id, total}: {
    id,
    total,
}

pairs | map index, [key, value]: {
    index,
    key,
    value,
}

orders | filter {total}: total > 100

orders | map {id, total = 0}: {
    id,
    total,
}
```

Names from patterns remain available in later stages. `filter` keeps the original elements. Defaults can use earlier bindings and the current key or index.

## Pipelines

Pipelines run from left to right. This expression keeps paid orders, then returns their IDs:

```javascript
orders | filter order: order.status == "paid" | map order: order.id
```

Each stage can use earlier bindings, outer bindings, and `this`. Reusing a name replaces the earlier binding within that stage and those that follow:

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

A pipeline processes one element at a time through all stages. If `any` or `all` determines the result, the entire pipeline stops. This returns `true` without evaluating `"a" + 1`:

```javascript
[1, "a"] | map x: x + 1 | any y: y > 0
```

The binding body after `:` stops at the next `|`. To nest a pipeline inside a body, parenthesize it:

```javascript
customers | map customer: {
    id: customer.id,
    totals: (customer.orders | filter order: order.paid) | map order: order.total,
}
```

## Any and All

`any` returns true if at least one element matches. `all` returns true if every element matches. Conditions must return booleans. For an empty collection, `any` returns false and `all` returns true.

```javascript
tags | any tag: startsWith(tag, "market:")
results | all result: result.status == "approved"
(results | any result: result.status == "approved") && enabled
```

## Reduce

`reduce` combines elements into one value. Name an accumulator and give its initial value, then bind the element as in other stages. Each result of the body becomes the accumulator for the next element:

```javascript
orders | reduce byID = {}, order: {
    ...byID,
    [order.id]: order,
}

prices | reduce names = [], name, _: [...names, name]
```

The initial value is required and is evaluated once, before the first element. It sees outer bindings but not stage bindings. An empty collection returns the initial value. The accumulator can be a pattern, and element bindings with the same name shadow it:

```javascript
scores | reduce {total, count} = {
    total: 0,
    count: 0,
}, score: {
    total: total + score,
    count: count + 1,
}
```

Like `any` and `all`, `reduce` ends its pipeline, so later stages don't see its bindings. For plain totals and counts, `sum` and `length` are shorter.
