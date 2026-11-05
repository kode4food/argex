---
title: "Command Line"
weight: 15
---

# Command Line

`aq` runs Argyll Expressions against JSON from files or standard input and prints the results.

```sh
go install github.com/kode4food/argex/cmd/aq@latest
```

```sh
aq [-c] [-r] [-n] expression [file ...]
```

Each input value becomes `this` for one evaluation. By default, results print as indented JSON.

```sh
echo '{"items":[{"id":1,"price":12},{"id":2,"price":4}]}' |
  aq 'items | filter i: i.price > 10 | map i: i.id'
```

| Flag | Effect |
| --- | --- |
| `-c` | Print compact JSON |
| `-r` | Print string results without quotes |
| `-n` | Evaluate once without reading input |

Missing results print nothing. Numbers that pass through unchanged keep their original text, including `1.10` and large integers. `aq` reports errors on standard error and exits with status 1.
