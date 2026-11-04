---
title: "Local Bindings"
weight: 45
---

# Local Bindings

```javascript
let reason = case {
    age < 18: "age below minimum",
    income < 30000: "income below minimum",
    "all criteria met",
}: {
    eligible: reason == "all criteria met",
    reason: reason,
}
```

`let` assigns names to values and returns the value of the expression after `:`. Separate bindings with commas. Each value is evaluated once, in order.

A binding can use earlier bindings:

```javascript
let subtotal = price * quantity, tax = subtotal * 0.08: subtotal + tax
```

A binding shadows an input property or outer binding with the same name. The input stays reachable through `this`:

```javascript
let total = total ?? 0: total + this.shipping
```

`_` evaluates a value without binding it.

## Destructuring

Use an object or array pattern to bind several members of one value:

```javascript
let {name, age: years} = user: {name, years}
let [first, _, third] = items: first + third
let {address: {city}, points: [x, y]} = user: {city, x, y}
```

`age: years` names the `age` property `years`. Use `_` or an empty array position to skip a member. Skipped members are read only if they have a default. Patterns can nest and allow trailing commas, but names cannot repeat.

The source is evaluated once. Missing members without defaults, and null or missing containers, raise `ErrMissing`. Wrong container types raise `ErrType`. Extra members are ignored. Rest bindings and computed keys are not supported.

### Defaults

Add `= expression` to provide a default for a missing member:

```javascript
let {name = "anonymous", age: years = 0} = user: {name, years}
let [first = 0, second = first + 1] = items: [first, second]
let {address: {city = "unknown"} = {}} = user: city
```

A default runs only when a member is missing, not when it is null. It can use outer bindings and earlier names in the pattern. A nested pattern's default supplies its missing container.

Supply a fallback in the source expression when needed:

```javascript
let {name = "anonymous"} = user ?? {}: name
```

## Scope

A `let` body extends to the end of the expression or enclosing group. It can contain a pipeline:

```javascript
let limit = 100: orders | filter order: order.total > limit | map order: order.id
```

To bind a pipeline's result, parenthesize it:

```javascript
let paid = (orders | filter order: order.paid): length(paid)
```

Parenthesize `let` when using it as an operand or collection stage body:

```javascript
items | map item: (let cost = item.price * item.quantity: {item: item.id, cost: cost})
```

`let` is reserved as a binding name. It remains valid as a property name and object key, as in `this.let` and `{let: 1}`.

## Types

[Type checking](../static-types/) applies to bindings too. For example, `let s = "a": s - 1` fails during compilation.
