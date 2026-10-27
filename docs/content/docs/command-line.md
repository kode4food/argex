---
title: "Command Line"
weight: 15
---

# Command Line

`argex` evaluates an expression against JSON input.

```sh
go install github.com/kode4food/argex/cmd/argex@latest
```

```sh
argex [-c] [-r] [-n] expression [file ...]
```

Each JSON value read from the named files, or from standard input when none are given, becomes `this` for one evaluation. Results print as indented JSON.

```sh
echo '{"items":[{"id":1,"price":12},{"id":2,"price":4}]}' |
  argex 'items | filter i: i.price > 10 | map i: i.id'
```

| Flag | Effect |
| --- | --- |
| `-c` | Print compact JSON |
| `-r` | Print string results without quotes |
| `-n` | Evaluate once with `this` as null, reading no input |

A missing result prints nothing. Numbers keep their original text when they pass through unchanged, so `1.10` and large integers are not reformatted. Errors are reported on standard error with exit status 1.
