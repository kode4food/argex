---
title: "Argyll Expressions"
---

# Argyll Expressions

Argyll Expressions is a small, pure expression language for JSON values with familiar JavaScript-style operators and access syntax. An expression reads its input as `this`, and the top-level properties of an object input by name. Expressions support arithmetic, comparisons, optional access, object and array construction, conditional selection, and explicit collection operations.

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
- Optional access and a `??` fallback that distinguish missing values from JSON `null`
- Object and array construction with spreads and computed keys
- `case` for conditional selection
- `map`, `filter`, `any`, and `all` collection operations chained with `|`
- A fixed set of pure functions for strings, numbers, and objects
- Compilation to a reusable Go function that is safe for concurrent use

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
