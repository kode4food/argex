package argex_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argex"
)

func TestArithmetic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		fn    func(float64, float64) (float64, bool)
		left  float64
		right float64
		want  float64
	}{
		{name: "add", fn: argex.Add, left: 0.1, right: 0.2, want: 0.3},
		{name: "add ints", fn: argex.Add, left: 2, right: 3, want: 5},
		{name: "sub", fn: argex.Sub, left: 0.3, right: 0.1, want: 0.2},
		{name: "mul", fn: argex.Mul, left: 0.1, right: 3, want: 0.3},
		{name: "div", fn: argex.Div, left: 0.3, right: 0.1, want: 3},
		{name: "mod", fn: argex.Mod, left: 0.5, right: 0.2, want: 0.1},
		{name: "mod negative", fn: argex.Mod, left: -0.5, right: 0.2, want: -0.1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, ok := tc.fn(tc.left, tc.right)
			assert.True(t, ok)
			assert.Equal(t, tc.want, res)
		})
	}
}

func TestArithmeticRefuses(t *testing.T) {
	inf := math.Inf(1)
	for _, tc := range []struct {
		name  string
		fn    func(float64, float64) (float64, bool)
		left  float64
		right float64
	}{
		{name: "add inf", fn: argex.Add, left: inf, right: 1},
		{name: "sub nan", fn: argex.Sub, left: 1, right: math.NaN()},
		{name: "mul inf", fn: argex.Mul, left: 2, right: inf},
		{name: "div zero", fn: argex.Div, left: 1, right: 0},
		{name: "div inf", fn: argex.Div, left: inf, right: 2},
		{name: "mod zero", fn: argex.Mod, left: 1, right: 0},
		{name: "mod nan", fn: argex.Mod, left: math.NaN(), right: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := tc.fn(tc.left, tc.right)
			assert.False(t, ok)
		})
	}
}

func TestArithmeticMatchesEval(t *testing.T) {
	res, err := argex.Eval("left + right", map[string]any{
		"left":  0.1,
		"right": 0.2,
	})
	assert.NoError(t, err)
	sum, ok := argex.Add(0.1, 0.2)
	assert.True(t, ok)
	assert.Equal(t, res, sum)
}
