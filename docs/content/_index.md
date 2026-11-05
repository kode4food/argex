---
title: "Argyll Expressions"
description: "A small, pure expression language for JSON values in Go"
---

# Argyll Expressions

Argyll Expressions is a language for querying and transforming JSON with JavaScript-style syntax.

Take a list of orders.

```json
{
  "orders": [
    {"id": "A", "paid": true,  "quantity": 3, "price": 3.95},
    {"id": "B", "paid": false, "quantity": 1, "price": 6.50}
  ]
}
```

| Expression | Result |
| --- | --- |
| `orders[0].id` | `"A"` |
| `orders[0].quantity * orders[0].price` | `11.85` |
| `orders[1].paid && orders[1].quantity > 0` | `false` |

To get the total for **every paid order**, pipe the array through `filter` and `map`.

```javascript
orders
    | filter order: order.paid
    | map order: {
        id: order.id,
        total: order.quantity * order.price,
    }
```

```json
[{"id": "A", "total": 11.85}]
```

## Use It in Go

Argyll Expressions is designed to work directly with the values Go’s `encoding/json` produces when it unmarshals JSON into an `any`.

Evaluate an expression directly, or compile it once and reuse it across inputs. Compiled expressions support concurrent evaluation.

```sh
go get github.com/kode4food/argex@latest
```

[Bind a Go function]({{< relref "/docs/functions#environments-in-go" >}}) to make it available inside an expression.

→ [Use Argyll Expressions from Go]({{< relref "/docs/getting-started" >}})

→ [Use the command line]({{< relref "/docs/command-line" >}})
