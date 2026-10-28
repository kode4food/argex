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

A bound name carries the static type of its value, so [Type Checking](../static-types/) still catches mistakes in the body. `let s = "a": s - 1` fails during parsing, just as `"a" - 1` does.
