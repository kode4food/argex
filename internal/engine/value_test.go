package engine_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argex"
)

type equalityCase struct {
	name  string
	left  any
	right any
	equal bool
}

func TestMissing(t *testing.T) {
	for _, source := range []string{"absent", "absent?.x", "{}?.x"} {
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

func TestEquality(t *testing.T) {
	for _, tc := range []equalityCase{
		{name: "numbers", left: 1.5, right: 1.5, equal: true},
		{name: "different numbers", left: 1.5, right: 2.5},
		{name: "NaN", left: math.NaN(), right: math.NaN()},
		{name: "infinity", left: math.Inf(1), right: math.Inf(1), equal: true},
		{name: "nulls", left: nil, right: nil, equal: true},
		{name: "null and false", left: nil, right: false},
		{name: "number and string", left: 1.0, right: "1"},
		{
			name:  "nested",
			left:  map[string]any{"a": []any{1.0, "b", true, nil}},
			right: map[string]any{"a": []any{1.0, "b", true, nil}},
			equal: true,
		},
		{name: "array order", left: []any{1.0, 2.0}, right: []any{2.0, 1.0}},
		{name: "missing key", left: map[string]any{"a": nil}, right: map[string]any{"b": nil}},
		{name: "ints", left: 7, right: 7, equal: true},
		{name: "int and float", left: 7, right: 7.0, equal: true},
		{
			name:  "equivalent numbers",
			left:  json.Number("7.0"),
			right: json.Number("7"),
			equal: true,
		},
		{name: "number and int", left: json.Number("7"), right: 7, equal: true},
		{name: "invalid number", left: json.Number("x"), right: math.NaN()},
		{name: "host values", left: int64(7), right: int64(7)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vars := map[string]any{"a": tc.left, "b": tc.right}
			value, err := argex.Eval("a == b", vars)
			assert.NoError(t, err)
			assert.Equal(t, tc.equal, value)
		})
	}
}
