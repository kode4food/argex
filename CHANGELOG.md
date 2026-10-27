# Changelog

Notable changes to Argyll Expressions.

## 0.1.1

- Pipeline stages see earlier bindings
- Bindings are optional, and `_` discards one
- `map v:` works on objects

## 0.1

First public release.

- Query and reshape JSON with familiar JavaScript-style syntax
- Chain `map`, `filter`, `any`, and `all` into pipelines that transform collections in one expression
- Build new objects and arrays from input data, with spreads and computed keys
- Pick values conditionally with `case`
- Navigate uncertain data safely with optional access and `??` defaults
- Pure and sandboxed, so expressions from configs or users can run safely inside Go programs
- Compile once and evaluate many times, concurrently
- `aq` brings the same expressions to the command line
