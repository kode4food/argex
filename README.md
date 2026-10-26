# Argyll Expressions

![Build Status](https://github.com/kode4food/argex/actions/workflows/build.yml/badge.svg) [![Code Coverage](https://qlty.sh/gh/kode4food/projects/argex/coverage.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![Maintainability](https://qlty.sh/gh/kode4food/projects/argex/maintainability.svg)](https://qlty.sh/gh/kode4food/projects/argex) [![GitHub](https://img.shields.io/github/license/kode4food/argex)](https://github.com/kode4food/argex/blob/main/LICENSE)

Argyll Expressions is a small, pure expression language for JSON values with familiar JavaScript-style operators and access syntax. Expressions support arithmetic, comparisons, optional access, object and array construction, conditional selection, and explicit list operations.

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

Import `github.com/kode4food/argex`. `Parse` returns an immutable `*Expression`, and `Compile` returns a `Program`, defined as `func(map[string]any) (any, error)`. Call that function directly and reuse it for repeated evaluation. `Eval(source, variables)` combines all three operations for one-off expressions. Compiled functions are safe for concurrent use. Callers must not concurrently mutate their input maps or arrays.

Values use Go's JSON representation: `map[string]any`, `[]any`, strings, booleans, `float64`, and `nil`. Arithmetic also accepts Go `int` and `json.Number` inputs and returns `float64`. Numbers use finite IEEE 754 double precision. Division by zero and non-finite arithmetic results fail. Use `number(value)` and `string(value)` for explicit type conversions.

## Collections and grouping

`| map` and `| filter` bind each element of their receiver to a name. The expression after `:` is evaluated in that scope. It can read outer bindings and input variables. An inner binding with the same name shadows the outer one.

A binding may name two identifiers, `key, value`, to also bind each element's position: the numeric index for an array receiver, or the property name for an object receiver, visited in sorted key order. The single-name form requires an array receiver; the two-name form accepts arrays and objects.

```text
orders | map order: order.total
orders | filter order: order.status == "paid"

(orders | filter order: order.status == "paid") | map order: order.id

items | map index, item: {position: index, value: item}
totals | filter key, amount: amount > 100

customers | map customer: {
    id: customer.id,
    totals: (customer.orders | filter order: order.paid) | map order: order.total,
}

{
    first: list1 | map item: item.name,
    second: list2 | filter item: item.available,
}
```

The binding body is a complete expression. Parentheses end a collection expression before applying another operation to its result. `| map` preserves order. `| filter` requires a boolean predicate and preserves the relative order of retained elements. Both produce an array, including for empty inputs.

`.` and `?.` access properties. `|` introduces collection operations. `response.map`, `response.filter`, and `response.any` read properties with those names. `??` provides a fallback for missing or null values. Use `(response?.items ?? []) | map item: item.id` to map an optional array. `||` means boolean OR.

`| any` tests whether at least one element satisfies a condition. `| all` tests whether every element satisfies it. Both require boolean conditions, return booleans, and stop as soon as their result is known. For an empty collection, `any` returns false and `all` returns true without evaluating the condition. The same binding and grouping rules apply as for `map` and `filter`.

```javascript
tags | any tag: startsWith(tag, "market:")
results | all result: result.status == "approved"
(results | any result: result.status == "approved") && enabled
```

## Constructing arrays and objects

Array and object literals may include a `...` spread. Spreading an array into an array inserts its elements; spreading an object into an object copies its entries. Later entries win, so a field or spread overrides an earlier key. Spreading a missing or null value contributes nothing. Spreading any other type, or spreading an array into an object or an object into an array, is an error.

```javascript
[first, ...rest, last]
{...defaults, ...overrides, id: record.id}
(response?.tags ?? []) | map tag: tag.name
```

Each spread produces a fresh value and never mutates its source. Object construction still omits fields whose value is missing, including entries drawn from a spread.

## Conditional selection

```text
case {
    score >= 0.9: "accept",
    score >= 0.5: "review",
    "reject",
}
```

A `case` has one or more `condition: result` arms and a required final bare fallback expression. Commas separate arms, and a trailing comma is allowed. Conditions must be booleans. The first true condition selects its result. Later conditions and unselected results are not evaluated. The fallback is evaluated only if every condition is false. Nested cases, arrays, and objects are ordinary result expressions.

## Missing values and optional access

An absent variable produces `Missing{}`, distinguishable from explicit JSON `null` using `IsMissing`. Reading a missing property, a missing intermediate value, or an out-of-range index through ordinary access fails with `ErrMissing`. A present property whose value is null returns null.

```text
customer.address.country
customer?.address?.country
response.choices?.[0]?.message?.content ?? ""
values?.[index] ?? 0
```

`?.name` and `?.[index]` return missing for a null or absent receiver or an absent member. A wrong receiver type or an invalid index type produces `ErrType`. Optionality applies to each marked access: use `a?.b?.c` when both accesses may be absent. Optional indexing skips evaluation of the index when the receiver is null or missing.

`??` evaluates its right operand only when its left operand is missing or null. Zero, false, and the empty string are preserved. Object construction omits fields whose value is missing. Arrays retain missing elements. A missing value cannot be JSON-encoded: explicitly use `?? null` or another fallback where a serializable value is required.

## Operators and functions

Precedence, highest first:

| Operators | Meaning |
| --- | --- |
| `.`, `?.`, `[]`, `?.[]`, calls | Access and calls |
| `!`, unary `+`, unary `-` | boolean negation and numeric signs |
| `*`, `/`, `%` | Multiplication, division, remainder |
| `+`, `-` | Addition and subtraction, with `+` also joining two strings |
| `<`, `<=`, `>`, `>=`, `in` | Numeric or string ordering and array membership |
| `==`, `!=` | Equality by type and contents |
| `??` | Missing/null fallback |
| `&&` | boolean conjunction |
| `\|\|` | boolean disjunction |

Binary operators associate left to right except `??`, which associates right to left. Ordering comparisons cannot be chained without parentheses. A collection binding body consumes a full expression regardless of the precedence of the expression containing the collection. `&&` and `||` require boolean operands and short-circuit. Strings compare lexicographically. Missing equals missing and differs from null.

`in` tests array membership using the same structural equality as `==`. It returns a boolean and stops at the first match. The right operand must be an array. Objects, strings, null, and missing values produce `ErrType`. Empty arrays return false. Like ordering comparisons, `in` requires parentheses when combined with another comparison.

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
| `sum(array)` | Sum of numeric elements, or zero for an empty array |
| `round(number)` | Nearest integer, with halfway values rounded away from zero |
| `floor(number)` | Greatest integer less than or equal to the number |
| `ceil(number)` | Least integer greater than or equal to the number |
| `startsWith(text, prefix)` | Whether text begins with the prefix |
| `endsWith(text, suffix)` | Whether text ends with the suffix |
| `includes(text, part)` | Whether text contains the substring |
| `lower(text)` | Unicode lowercase form |
| `upper(text)` | Unicode uppercase form |
| `trim(text)` | Remove surrounding Unicode whitespace |
| `split(text, separator)` | Array of strings separated by the literal separator |
| `join(array, separator)` | Join an array of strings with the separator |
| `number(value)` | Finite number from a number or a string containing a JSON number |
| `string(value)` | String form of a string, finite number, boolean, or null |
| `keys(object)` | Array of property names in lexicographic order |
| `values(object)` | Array of values in the same property name order as `keys` |

Call functions by name, as in `length(items)` or `startsWith(text, prefix)`. String tests are case-sensitive and accept empty search strings. `includes` tests string substrings. Use `in` for array membership. Case conversion uses locale-independent Unicode character mappings. Splitting preserves empty fields. An empty separator splits into Unicode code points, with an empty input producing an empty array. Joining requires every element to be a string and returns an empty string for an empty array.

Conversions are explicit. `number` trims surrounding whitespace and requires the entire remaining string to be a valid JSON number. Empty strings, hexadecimal notation, leading zeros such as `"01"`, booleans, null, and non-finite results are rejected. `string` preserves strings, formats finite numbers without unnecessary decimal digits, and converts booleans and null to `"true"`, `"false"`, and `"null"`. It rejects missing values, arrays, and objects. Invalid types or conversions return `ErrType`. `sum` requires finite numeric elements and returns `ErrArithmetic` if the running total overflows. Functions never change their arguments.

```javascript
lower(trim(payment.status)) == "approved"
join(split("inventory,orders", ","), " / ")
sum(results | map result: number(result.cost))
ceil(number(usage.tokens) / 1000)
keys(response) | any key: startsWith(key, "x-")
```

`round` rounds halfway values away from zero: `round(-1.5)` returns `-2` and `round(1.5)` returns `2`. Object inspection uses lexicographic property name order even for numeric-looking names. `keys({"2": 2, "10": 10})` returns `["10", "2"]`.

Expressions are pure and deterministic: their results depend on the supplied values. Collection operations use `| map`, `| filter`, `| any`, or `| all` followed by a `name: expression` or `key, name: expression` binding.

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
