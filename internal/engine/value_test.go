package engine_test

import (
	"encoding/json"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

func TestMissing(t *testing.T) {
	for _, source := range []string{"absent", "null?.x", "{}?.x"} {
		value, err := argex.Eval(source, nil)
		assert.NoError(t, err)
		assert.True(t, argex.IsMissing(value))
	}
	assert.False(t, argex.IsMissing(nil))
	assert.False(t, argex.IsMissing(false))
	_, err := json.Marshal(argex.Missing{})
	assert.ErrorIs(t, err, argex.ErrMissing)
	value, err := argex.Eval("[absent]", nil)
	assert.NoError(t, err)
	_, err = json.Marshal(value)
	assert.ErrorIs(t, err, argex.ErrMissing)
}
