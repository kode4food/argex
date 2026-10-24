# argex

![Build Status](https://github.com/kode4food/argex/actions/workflows/build.yml/badge.svg) [![Code Coverage](https://qlty.sh/gh/kode4food/projects/argex/coverage.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![Maintainability](https://qlty.sh/gh/kode4food/projects/argex/maintainability.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![GitHub](https://img.shields.io/github/license/kode4food/argex)](https://github.com/kode4food/argex/blob/main/LICENSE)

A small, pure expression language for JSON values with familiar JavaScript-style operators and access syntax. Expressions support arithmetic, comparisons, optional access, object and array construction, conditional selection, and explicit list operations.

```text
customer.address.country
customer?.address?.country ?? "CH"
usage.input_tokens * input_price + usage.output_tokens * output_price

(items | filter item: item.price > 10) | map item: {
    id: item.id,
    category: case {
        item.price > 100: "large",
        "medium",
    },
}
```

## Go API

```go
expression, err := argex.Parse(`
    (items | filter item: item > 10) | map item: case {
        item > 100: "large",
        "medium",
    }
`)
if err != nil {
    return err
}
program, err := argex.Compile(expression)
if err != nil {
    return err
}
result, err := program(map[string]any{
    "items": []any{5, 20, 150},
})
// result: []any{"medium", "large"}
```

Import `github.com/kode4food/argex`. `Parse` returns an immutable `*Expression`, and `Compile` returns a `Program`, defined as `func(map[string]any) (any, error)`. Call that function directly and reuse it for repeated evaluation. `Eval(source, variables)` combines all three operations for one-off expressions. Compiled functions are safe for concurrent use; callers must not concurrently mutate their input maps or arrays.

Values use Go's JSON representation: `map[string]any`, `[]any`, strings, booleans, `float64`, and `nil`. Arithmetic also accepts Go `int` and `json.Number` inputs and returns `float64`. Numbers use finite IEEE 754 double precision; division by zero and non-finite arithmetic results fail. There is no implicit conversion between strings, numbers, and booleans.

## Collections and grouping

`| map` and `| filter` bind a name to each element of their receiver. The expression after `:` is evaluated in that scope. It can read outer bindings and input variables; an inner binding with the same name shadows the outer one. Bindings cannot be assigned to or returned as functions.

```text
orders | map order: order.total
orders | filter order: order.status == "paid"

(orders | filter order: order.status == "paid") | map order: order.id

customers | map customer: {
    id: customer.id,
    totals: (customer.orders | filter order: order.paid) | map order: order.total,
}

{
    first: list1 | map item: item.name,
    second: list2 | filter item: item.available,
}
```

The binding body is a complete expression. Parentheses end a collection expression before applying another operation to its result. Newlines and indentation have no grammatical meaning. There are no implicit operands or standalone anonymous functions. `| map` preserves order; `| filter` requires a Boolean predicate and preserves the relative order of retained elements. Both require an array receiver and produce an array, including for empty inputs.

`.` and `?.` access properties; `|` introduces collection operations. `response.map`, `response.filter`, and `response.any` are ordinary paths. Optional access and coalescing remain `?.` and `??`; use `(response?.items ?? []) | map item: item.id` to map an optional array. `||` remains Boolean OR. There are no ternary or bitwise operators.

## Conditional selection

```text
case {
    score >= 0.9: "accept",
    score >= 0.5: "review",
    "reject",
}
```

A `case` has one or more `condition: result` arms and a required final bare fallback expression. Commas separate arms; a trailing comma is allowed. Conditions must be Boolean. The first true condition selects its result. Later conditions and unselected results are not evaluated. The fallback is evaluated only if every condition is false. Nested cases, arrays, and objects are ordinary result expressions.

## Missing values and optional access

An absent variable produces `Missing{}`, distinguishable from explicit JSON `null` using `IsMissing`. Reading a missing property, a missing intermediate value, or an out-of-range index through ordinary access fails with `ErrMissing`. A present property whose value is null returns null.

```text
customer.address.country
customer?.address?.country
response.choices?.[0]?.message?.content ?? ""
values?.[index] ?? 0
```

`?.name` and `?.[index]` return missing for a null or absent receiver or an absent member. They do not hide a wrong receiver type or an invalid index type. Optionality applies to each marked access: use `a?.b?.c` when both accesses may be absent. Optional indexing skips evaluation of the index when the receiver is null or missing.

`??` evaluates its right operand only when its left operand is missing or null. Zero, false, and the empty string are preserved. Object construction omits fields whose value is missing. Arrays retain missing elements. A missing value cannot be JSON-encoded: explicitly use `?? null` or another fallback where a serializable value is required.

## Operators and functions

Precedence, highest first:

| Operators | Meaning |
| --- | --- |
| `.`, `?.`, `[]`, `?.[]`, calls | Access and calls |
| `!`, unary `+`, unary `-` | Boolean negation and numeric signs |
| `*`, `/`, `%` | Multiplication, division, remainder |
| `+`, `-` | Addition and subtraction; `+` also joins two strings |
| `<`, `<=`, `>`, `>=`, `in` | Numeric or string ordering; array membership |
| `==`, `!=` | Structural equality, without coercion |
| `??` | Missing/null fallback |
| `&&` | Boolean conjunction |
| `\|\|` | Boolean disjunction |

Binary operators associate left to right except `??`, which associates right to left. Ordering comparisons cannot be chained without parentheses. A collection binding body consumes a full expression regardless of the precedence of the expression containing the collection. `&&` and `||` require Boolean operands and short-circuit. Strings compare lexicographically. Missing equals missing and differs from null.

`in` tests array membership using the same structural equality as `==`. It returns a Boolean and stops at the first match. The right operand must be an array; objects, strings, null, and missing values produce `ErrType`. Empty arrays return false. Like ordering comparisons, `in` requires parentheses when combined with another comparison.

```text
"domain:inventory" in value.tags || "domain:orders" in value.tags
item in allowed_items
!(item in excluded_items)
```

Only these named pure functions are available:

| Function | Behavior |
| --- | --- |
| `length(value)` | Number of Unicode code points, array elements, or object members |
| `abs(number)` | Absolute value |
| `min(number, ...)` | Minimum of one or more numbers |
| `max(number, ...)` | Maximum of one or more numbers |

There are no assignments, statements, arbitrary method calls, I/O, ambient globals, clock access, or randomness. Collection operations use `| map` or `| filter` followed by a `name: expression` binding.

## Errors and limits

Use `errors.Is` with `ErrSyntax`, `ErrType`, `ErrMissing`, `ErrArithmetic`, `ErrFunction`, and `ErrLimit`. Syntax errors include the character offset. Unknown function names and incorrect argument counts fail during compilation, even in an unchosen branch. Runtime errors occur only in expressions actually evaluated.

Parsing and compilation limit expression nesting to 256 levels. Inputs must be JSON-shaped, acyclic values.

## Development

```sh
make pre-commit
go test -race ./...
go test ./internal/engine -run '^$' -fuzz FuzzExpression -fuzztime=10s
go test -run '^$' -bench . -benchmem
```

The grammar is in [grammar.ebnf](docs/grammar.ebnf).

The compliance corpus is in [testdata/compliance](testdata/compliance). Each JSON case specifies an expression, input variables, and an expected result, missing value, or error phase. `go test ./...` runs the corpus through both `Eval` and compiled programs, checks repeated execution, and verifies that inputs are unchanged.
