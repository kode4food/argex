---
title: "Argyll Expressions"
---

# Argyll Expressions

Argyll Expressions is a language for querying and transforming JSON with JavaScript-style syntax. Use `this` to refer to the input. If the input is an object, you can also refer to its properties by name.

```javascript
customer?.address?.country ?? "CH"
usage.input_tokens * input_price + usage.output_tokens * output_price

items | filter item: item.price > 10 | map item: {
    id: item.id,
    category: case {
        item.price > 100: "large",
        "medium",
    },
}
```

## What It Provides

- JavaScript-style paths, arithmetic, comparisons, and boolean logic
- Optional access and `??` fallbacks
- Object and array construction with spreads and computed keys
- `case` for conditional selection
- `let` bindings with destructuring and defaults
- `map`, `filter`, `any`, and `all` collection operations chained with `|`
- Builtin and custom functions
- Compile once and evaluate repeatedly or concurrently

## Quick Start

```sh
go get github.com/kode4food/argex@latest
```

```go
result, err := argex.Eval(`order.total * 1.2`, map[string]any{
    "order": map[string]any{"total": 50.0},
})
// result: 60.0
```

→ [Use Argyll Expressions from Go]({{< relref "/docs/getting-started" >}})

→ [Filter and transform collections]({{< relref "/docs/collections" >}})
