# AGENTS.md

This file defines repository-specific conventions for agents changing code and markdown in `argex`.

## Critical: Honor The Actual Request

- Do exactly what the user asks, at its full size. Do not reinterpret the request into a smaller task or add unrelated improvements.
- When asked to fix violations, audit and fix all violations covered by the request. Age, authorship, and whether a line was already in the diff are not reasons to omit it.
- Report the result exactly as it stands. Passing tests do not prove that a style audit is complete, and a partial cleanup must be reported as partial.
- If work is genuinely blocked, name the concrete blocker and list what remains undone. Do not substitute explanations about scope or authorship for the blocker.

## Critical: Safe, Reviewable Edits

- Never use bulk scripted find-and-replace, regex rewriting, or custom mass AST/source-rewriting scripts to edit code. This includes scripts intended only to reorder declarations or rewrap signatures.
- Read each affected function and its callers before changing it. Use symbol-aware navigation when available and focused, explicit patches for edits.
- Make small, coherent changes. Inspect the exact diff and run the relevant checks before continuing to the next structural change.
- Standard `gofmt`, `goimports`, and `go fix` are allowed. Inspect their changes too; tool output is not automatically correct or compliant with these rules.
- If an edit damages code, stop further cleanup, restore the affected code from a verified intact version, and verify the repair before proceeding. Preserve the user's other work and all previously completed changes.
- Never use a passing build or test run as a substitute for reviewing signatures, declarations, call sites, and behavior in the diff.

## Baseline

- Target Go version: `1.27`.
- Keep changes idiomatic and minimal.
- Follow existing local patterns over generic style rules.

## Ownership And Boundaries

- Give every concept one authoritative owner. Keep its state, configuration, caches, and invariants with that owner rather than duplicating or shadowing them elsewhere.
- Organize files and packages around real concerns and reasons to change. File length alone is not a reason to split code.
- Reuse existing mechanisms before adding new ones. Avoid generic dumping-ground packages, forwarding layers, and abstractions that do not solve an actual dependency problem.
- Keep interfaces minimal and consumer-owned. A package boundary alone does not justify an interface.
- Keep the exported surface small. Do not re-export another package's identifiers through variable aliases; reference the owner directly.
- Compilation and optimization must leave caller-owned ASTs intact. Copy the necessary data before rewriting or capturing it; do not mutate shared nodes.

## File And Declaration Layout

Use this top-level order unless a file has a clear local reason to differ:

1. `package`
2. `import` block
3. `type` declarations
4. `const` declarations
5. `var` declarations
6. functions and methods

For `type` declarations, order by containership and usage priority:

- Put the primary API type first in the block.
- Put types contained by that type next.
- Put dependencies of those types after their users.
- Put unexported types after exported types, still following containership.

For grouped declarations:

- Use `type ( ... )`, `const ( ... )`, or `var ( ... )` only when there are multiple related declarations.
- Put exported types first, then unexported types, preserving containership and dependency order.
- Error sentinels must use `Err`-prefixed names in a `var` block.
- Keep compile-time interface assertions in `var` blocks.
- Never declare named types or constants inside function bodies, including tests. Args/Res bundles follow the placement exception below.

## Arguments, Results, And Struct Literals

- Functions taking five or more arguments must use an `Args` bundle. Functions returning three or more data values must use a `Res` bundle; trailing `bool` and `error` success indicators do not count and stay outside the bundle.
- Avoid ambiguous adjacent same-type parameters or results. Use named bundle fields or distinct named types so a positional swap cannot silently change the meaning.
- An Args/Res bundle belongs to one function boundary and is passed by value. Declare it immediately before that function, outside its body; group Args and Res together when both exist.
- Do not store or forward Args/Res bundles through other functions. A value shared across boundaries needs a descriptive domain name rather than an Args/Res suffix.
- For plain structs, use values up to 32 bytes; use values at 33–64 bytes unless they are read through three or more functions in a chain; use pointers above 64 bytes. Use pointers whenever mutation or identity requires them. A pointer does not imply a heap allocation; check escape behavior when it matters.
- Struct fields must make sense at the call site without reading the function body. Give adjacent same-type fields distinct, descriptive meanings.
- Use named fields in struct literals. Positional struct literals are forbidden except for a single-field wrapper whose field name adds no information.
- Keep a struct literal on one line if it fits. When it wraps, use one field per line.

## Function And Method Ordering

Within a file, order callables as follows:

1. Constructors (`New...`)
2. Exported methods of exported types
3. Unexported methods of exported types
4. Methods of unexported types
5. Exported functions
6. Unexported helper functions

For exported receiver types:

- Put exported methods before unexported methods.
- Keep related methods together by functionality.
- Within a group, order by call chain or first use.

## Naming

- Call the language **Argyll Expressions** in documentation and user-facing prose. Code identifiers may use **ArgEx**.
- Prefer **Argyll Expressions** in doc comments unless referring to a literal, package, or file named `argex`. Keep the documented identifier at the start of Go doc comments.
- Preserve literal package names, import paths, file names, URLs, and language keys.

- Receiver names: short, lowercase, and derived from type name (`e *Engine`, `c *Compiler`).
- Prefer short local names in tight scope (`i`, `n`, `ok`, `err`, `ctx`).
- Use `ok` for map/type assertion booleans.
- Constructor names must use `New`.
- Acronyms stay uppercase (`ID`, `URL`, `HTTP`).
- Name locals for their semantic subject rather than restating their type. Use longer names for wider scope and public boundaries.
- Name booleans for the condition they express. Use `Is`/`Has` only when they clarify the name; preserve standard-library naming conventions.
- Name meaningful constants instead of scattering magic values. Group related constants and use typed constants when the type carries meaning.

## Imports

- Run `goimports` on all files.
- Keep imports grouped and sorted per `goimports` output.

## Wrapping And Formatting

- Wrap Go source to 80 columns max (tabs count as 4 columns).
- The width limit applies to Go comments as well as code, but not Markdown prose.
- Keep signatures and calls on one line when they fit.
- When wrapping is required, pack as many args per line as fit before wrapping.
- For wrapped signatures and calls, break after `(` and keep trailing commas.
- For wrapped calls where first arg is `t *testing.T`, keep `t` on the first line before wrapping remaining args.
- In `switch` statements, leave one blank line after each non-empty `case` block when another `case` or `default` block follows.

## Markdown Style

- Do not hard-wrap Markdown text.
- Let lines run long in `.md` files.
- Preserve meaningful line breaks in lists, tables, code fences, and quotations.
- Exception: do not reformat `LICENSE.md`.
- User documentation should explain behavior, usage, and meaningful choices. Include implementation details only when they help the reader use the library.

## Implementation Style

- Prefer early returns and guard clauses.
- Use guards for real preconditions before substantial logic. For a short successful lookup or assertion, scope the value and `ok` in the `if` initializer and put the fallback afterward.
- Avoid deep nesting; one conditional nesting level max unless duplication avoidance clearly justifies more.
- Do not assign multiple variables from independent expressions in one statement. Write one assignment per line; receiving multiple results from a single call is allowed.
- Keep helpers near where they are used.
- Do not use panic in production code paths, except in `Must`-prefixed helpers whose documented contract is to panic on failure.

## Global State

- Mutable production package-level state is forbidden. Counters, caches, and mutable registries belong to the instance that owns their behavior.
- Package-level variables are reserved for error sentinels, compile-time interface assertions, and genuinely immutable lookup data. An immutable binding does not make the referenced map, slice, or object immutable.
- Never alias another package's variables to create a local re-export, including error sentinels. Import the owning package and use its identifier directly.

## Comments

- Document exported APIs concisely, describing behavior rather than implementation. Keep godoc to three lines when possible; sentinel messages and documented enum blocks need no redundant per-member comments.
- Avoid comments on self-explanatory private helpers. Explain non-obvious behavior or the reason for a decision in at most two lines.
- End the last sentence of godoc without a period.

## Error Handling

- Publicly relevant errors must be typed sentinels (`Err...`) and returned or wrapped.
- Wrap with `%w` first, then context (`fmt.Errorf("%w: %s", ErrX, detail)`).
- Do not return plain ad-hoc error strings from production paths when a typed sentinel applies.
- Return an existing sentinel directly when there is no context to add. Handle errors immediately.
- Test sentinel identity with `errors.Is` or `assert.ErrorIs`, rather than matching error-message strings.

## Testing Conventions

- Tests are black-box only: use external test packages (`<pkg>_test`).
- Use `testing` package with `TestXxx` naming.
- Keep tests deterministic and explicit.
- Verify both happy paths and error paths.
- Use table-driven cases and short scenario names. Keep test helpers marked with `t.Helper()` and align test files with the source concern they cover.
- Use `testify/assert`, not `testify/require`, and omit assertion message arguments.
- Target at least 90% coverage. A short or focused test run alone is not sufficient to declare the whole task complete.

## Runtime Performance

- Benchmark evaluator changes before and after with `-benchmem`; compare `ns/op`, `B/op`, and `allocs/op` on representative queries and documents.
- Prioritize repeated execution speed and runtime allocations. Extra compilation work is acceptable when it makes the compiled functions faster across many executions.
- Move repeatable preparation into compilation when semantics allow it. Preserve short-circuit order, missing-vs-null behavior, and custom-function effects.
- Profile uncertain bottlenecks instead of assuming fewer lines or calls are faster. Keep benchmark results and correctness evidence separate.

## Critical: Git Ownership

- Never commit unless the user explicitly requests a commit. Do not ask to commit as part of ordinary cleanup.
- Never stage, unstage, reset, restore the index, or stash unless the user explicitly requests that action. The staged set is the user's review record.
- When asked to stage a specific set, stage exactly that set. Do not unstage or otherwise disturb anything else.

## Change Checklist For Agents

Before finishing a change:

1. Ensure formatting is clean (`gofmt` and `goimports` as needed).
2. Ensure declaration and method order match this file.
3. Ensure typed errors and wrapping style are correct.
4. Ensure markdown style is respected (no hard-wrapped markdown text).
5. Ensure tests are black-box by default.
6. Run relevant tests (at minimum affected packages; ideally `go test ./...`).
7. Review the complete diff for accidental edits, API changes, and unfinished requirements. Run the full suite before claiming completion.
8. For runtime changes, report the measured performance impact. State any remaining violations or blockers explicitly.
