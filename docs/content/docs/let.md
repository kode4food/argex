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

`let` names one or more values, then evaluates a body that can read them. A `let` is an expression, and its result is the body's result. Commas separate bindings, and a colon ends them. Each value is evaluated once, in order, before the body.

A later value sees every earlier binding:

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

`age: years` binds the `age` property as `years`. Patterns can nest. Use `_` to skip a member, or leave an array position empty, as in `[first, , third]`. Skipped members are not read unless they have a default. Trailing commas are allowed; names cannot repeat within a pattern.

The source is evaluated once. Members without defaults are required; a missing member or a null or missing container produces `ErrMissing`. A wrong container type produces `ErrType`. Extra members are ignored. Rest bindings and computed keys are not supported.

### Defaults

Add `= expression` to provide a default for a missing member:

```javascript
let {name = "anonymous", age: years = 0} = user: {name, years}
let [first = 0, second = first + 1] = items: [first, second]
let {address: {city = "unknown"} = {}} = user: city
```

A default is evaluated only for a missing member. It can use outer bindings and earlier names in the pattern. A default on a nested pattern supplies the missing container.

Supply a fallback in the source expression when needed:

```javascript
let {name = "anonymous"} = user ?? {}: name
```

## Scope

A `let` has the same precedence as a pipeline, so its body extends to the end of the expression or the enclosing group, and it can be a whole pipeline:

```javascript
let limit = 100: orders | filter order: order.total > limit | map order: order.id
```

To bind a pipeline's result, parenthesize it:

```javascript
let paid = (orders | filter order: order.paid): length(paid)
```

A `let` can stand anywhere a full expression can: the whole program, a parenthesized group, a `case` arm, an array element, an object field value, or a function argument. As an operand or a collection stage body, it needs parentheses:

```javascript
items | map item: (let cost = item.price * item.quantity: {item: item.id, cost: cost})
```

`let` is reserved as a binding name. It remains valid as a property name and object key, as in `this.let` and `{let: 1}`.

## Types

A bound name carries the static type of its value, so [Type Checking](../static-types/) still catches mistakes in the body. `let s = "a": s - 1` fails during compilation, just as `"a" - 1` does.
