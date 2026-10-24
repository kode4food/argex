package engine_test

import (
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

func TestSkippedCompilation(t *testing.T) {
	for _, source := range []string{
		"case { true: 1, unknown() }",
		"false && unknown()", "1 ?? unknown()", "null?.[unknown()]",
		"[] | map x: unknown()", "[] | filter x: unknown()",
	} {
		_, err := argex.Eval(source, nil)
		assert.ErrorIs(t, err, argex.ErrFunction)
	}
}
