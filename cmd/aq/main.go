// Command aq evaluates an Argyll Expression against JSON input
package main

import (
	"fmt"
	"os"

	app "github.com/kode4food/argex/cmd/aq/internal"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "aq:", err)
		os.Exit(1)
	}
}
