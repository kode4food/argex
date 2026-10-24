package engine_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

func TestFunctionNumericInputs(t *testing.T) {
	for _, value := range []any{2, float64(2), json.Number("2")} {
		variables := map[string]any{"value": value}
		for _, source := range []string{
			"number(value)", "sum([value])", "round(value)",
			"floor(value)", "ceil(value)",
		} {
			result, err := argex.Eval(source, variables)
			assert.NoError(t, err)
			assert.Equal(t, float64(2), result)
		}
		result, err := argex.Eval("string(value)", variables)
		assert.NoError(t, err)
		assert.Equal(t, "2", result)
	}
	for _, value := range []any{
		math.NaN(), math.Inf(1), math.Inf(-1), json.Number("bad"),
	} {
		for _, source := range []string{
			"number(value)", "string(value)", "sum([value])",
			"round(value)", "floor(value)", "ceil(value)",
		} {
			_, err := argex.Eval(source, map[string]any{"value": value})
			assert.ErrorIs(t, err, argex.ErrType)
		}
	}
}
