package engine_test

import (
	"encoding/json"
	"math"
	"math/big"
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

type (
	signedZeroCase struct {
		source string
		signed bool
	}

	// operatorCase names the path its operands take through the arithmetic
	operatorCase struct {
		name  string
		op    string
		left  float64
		right float64
	}

	sumCase struct {
		name  string
		items []any
	}
)

const (
	remainderSamples = 1024
	oracleSamples    = 4096
	oracleSumLength  = 8

	// Operands reach past what an int64 coefficient and an exact power of ten
	// can hold, so the big.Rat fallback is exercised too
	maxOracleDigits   = 17
	maxOracleExponent = 25
)

var operators = []string{"+", "-", "*", "/", "%"}

func TestDecimalArithmetic(t *testing.T) {
	for _, tc := range []evaluationCase{
		{source: "0.1 + 0.2", want: 0.3},
		{source: "1299.99 * 3", want: 3899.97},
		{source: "1299.99 + 1299.99 + 1299.99", want: 3899.97},
		{source: "3899.97 - 1299.99", want: 2599.98},
		{source: "0.15 * 1299.99", want: 194.9985},
		{source: "0.3 / 0.1", want: float64(3)},
		{source: "1 / 3", want: 1.0 / 3},
		{source: "1.1 % 0.1", want: float64(0)},
		{source: "-5.5 % 2", want: -1.5},
		{source: "5.5 % -2", want: 1.5},
		{source: "sum([0.1, 0.2, 0.3])", want: 0.6},
		{source: "sum([1, 2, 3])", want: float64(6)},
		{source: "sum([9007199254740991, 2, -1])", want: float64(1 << 53)},
		{source: "-7 % 3", want: float64(-1)},
		{source: "9007199254740991 + 1", want: float64(1 << 53)},
		{source: "9007199254740992 + 1", want: float64(1 << 53)},
		{source: "4503599627370496 * 3", want: float64(3 << 52)},
		{source: "-9007199254740991 - 2", want: -9007199254740993.0},
		{source: "7 / 2", want: 3.5},
		{source: "2 / 3", want: 2.0 / 3},
		{source: "1e-30 + 1", want: 1 + 1e-30},
		{source: "1e20 * 1.5", want: 1.5e20},
	} {
		t.Run(tc.source, func(t *testing.T) {
			value, err := argex.Eval(tc.source, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestSignedZero(t *testing.T) {
	for _, tc := range []signedZeroCase{
		{source: "zero * -1", signed: true},
		{source: "zero / -1", signed: true},
		{source: "-zero + -zero", signed: true},
		{source: "-zero % 1", signed: true},
		{source: "-zero * 1.5", signed: true},
		{source: "-1.5 % 0.5", signed: true},
		{source: "1 - 1", signed: false},
		{source: "-1 + 1", signed: false},
		{source: "0.1 - 0.1", signed: false},
	} {
		t.Run(tc.source, func(t *testing.T) {
			value, err := argex.Eval(tc.source, map[string]any{"zero": 0.0})
			assert.NoError(t, err)
			assert.Equal(t, tc.signed, math.Signbit(value.(float64)))
		})
	}
}

func TestDecimalUnderflow(t *testing.T) {
	for _, source := range []string{
		"4.4e-323 * 0.05555555555555556",
		"4.4e-323 / 17.999999999999996",
	} {
		for _, sign := range []string{"", "-"} {
			t.Run(sign+source, func(t *testing.T) {
				value, err := argex.Eval(sign+source, nil)
				if !assert.NoError(t, err) {
					return
				}
				assert.Equal(t, float64(0), value)
				assert.Equal(t, sign == "-", math.Signbit(value.(float64)))
			})
		}
	}
}

func TestRemainder(t *testing.T) {
	expr, err := argex.Parse("left % right")
	if !assert.NoError(t, err) {
		return
	}
	program, err := argex.DefaultEnv().Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	values := []float64{
		0, math.Copysign(0, -1), 1, -1, 1.5, -1.5,
		math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64,
		math.MaxFloat64, -math.MaxFloat64,
		math.Ldexp(1, -1022), math.Ldexp(1, -1022) * 1.5,
	}
	variables := map[string]any{}
	for _, left := range values {
		for _, right := range values {
			if right == 0 {
				continue
			}
			variables["left"] = left
			variables["right"] = right
			value, err := program.Eval(variables)
			if !assert.NoError(t, err) {
				return
			}
			assertRemainder(t, left, right, value.(float64))
		}
	}
	random := rand.New(rand.NewPCG(1, 2))
	for range remainderSamples {
		left := math.Float64frombits(random.Uint64())
		right := math.Float64frombits(random.Uint64())
		if left-left != 0 || right-right != 0 || right == 0 {
			continue
		}
		variables["left"] = left
		variables["right"] = right
		value, err := program.Eval(variables)
		if !assert.NoError(t, err) {
			return
		}
		assertRemainder(t, left, right, value.(float64))
	}
}

// A decimal remainder differs from math.Mod's binary one, so this checks only
// what both share: finite, within the divisor, signed like the dividend
func assertRemainder(t *testing.T, left, right, value float64) {
	t.Helper()
	assert.False(t, math.IsInf(value, 0) || math.IsNaN(value))
	assert.LessOrEqual(t, math.Abs(value), math.Abs(right))
	assert.Equal(t, math.Signbit(left), math.Signbit(value))
}

func TestDecimalOracle(t *testing.T) {
	random := rand.New(rand.NewPCG(3, 4))
	for _, op := range operators {
		expr, err := argex.Parse("left " + op + " right")
		if !assert.NoError(t, err) {
			return
		}
		program, err := argex.DefaultEnv().Compile(expr)
		if !assert.NoError(t, err) {
			return
		}
		for range oracleSamples {
			left := randomDecimal(t, random)
			right := randomDecimal(t, random)
			if right == 0 && (op == "/" || op == "%") {
				continue
			}
			value, err := program.Eval(map[string]any{
				"left": left, "right": right,
			})
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, oracle(op, left, right), value.(float64))
		}
	}
}

func TestDecimalSumOracle(t *testing.T) {
	random := rand.New(rand.NewPCG(5, 6))
	expr, err := argex.Parse("sum(items)")
	if !assert.NoError(t, err) {
		return
	}
	program, err := argex.DefaultEnv().Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	for range oracleSamples {
		items := make([]any, oracleSumLength)
		total := new(big.Rat)
		for i := range items {
			n := randomDecimal(t, random)
			items[i] = n
			total.Add(total, exactRat(n))
		}
		value, err := program.Eval(map[string]any{"items": items})
		if !assert.NoError(t, err) {
			return
		}
		want, _ := total.Float64()
		assert.Equal(t, want, value.(float64))
	}
}

func TestDecimalPaths(t *testing.T) {
	for _, tc := range []operatorCase{
		{name: "add integers", op: "+", left: 2, right: 3},
		{name: "add past 2^53", op: "+", left: 1<<53 - 1, right: 2},
		{name: "add same scale", op: "+", left: 0.1, right: 0.2},
		{name: "add mixed scales", op: "+", left: 0.1, right: 0.25},
		{name: "add integer and fraction", op: "+", left: 1, right: 0.5},
		{name: "add opposite signs", op: "+", left: 0.1, right: -0.25},
		{name: "add scale gap", op: "+", left: 1e-20, right: 1},
		{
			name: "add rescale overflow", op: "+",
			left: 922337203685477500, right: 0.05,
		},
		{
			name: "add coefficient overflow", op: "+",
			left: 922337203685477500, right: 80.9,
		},
		{
			name: "add coefficient wraps", op: "+",
			left: 9.223372036854775e18, right: 9.223372036854775e18,
		},
		{name: "add scale past 22", op: "+", left: 1e-23, right: 2e-23},
		{
			name: "add result past 2^53", op: "+",
			left: 12345678901234.56, right: 0.001,
		},
		{
			name: "add rounded coefficient", op: "+",
			left: 18899558020595.22, right: 0.001,
		},
		{name: "add huge integer", op: "+", left: 1e20, right: 0.5},
		{name: "add subnormal", op: "+", left: 5e-324, right: 1},
		{name: "subtract to zero", op: "-", left: 0.1, right: 0.1},
		{name: "subtract integers", op: "-", left: -5, right: 3},
		{name: "subtract fractions", op: "-", left: 0.3, right: 0.1},
		{name: "subtract huge integer", op: "-", left: 1e20, right: 0.5},
		{name: "multiply integers", op: "*", left: 6, right: 7},
		{name: "multiply past 2^53", op: "*", left: 1 << 52, right: 4},
		{name: "multiply fractions", op: "*", left: 1.5, right: 1.5},
		{name: "multiply signs", op: "*", left: -1.5, right: 2.5},
		{name: "multiply money", op: "*", left: 1299.99, right: 0.15},
		{
			name: "multiply coefficient overflow", op: "*",
			left: 1234567.891, right: 9876543.211,
		},
		{name: "multiply scale past 22", op: "*", left: 1e-12, right: 1e-12},
		{name: "multiply huge left", op: "*", left: 1e20, right: 1.5},
		{name: "multiply huge right", op: "*", left: 1.5, right: 1e20},
		{name: "divide integers", op: "/", left: 7, right: 2},
		{name: "divide fractions", op: "/", left: 0.3, right: 0.1},
		{name: "divide repeating", op: "/", left: 1, right: 0.3},
		{name: "divide negative", op: "/", left: -0.5, right: 0.25},
		{
			name: "divide past 2^53", op: "/",
			left: 12345678901234.56, right: 0.001,
		},
		{
			name: "divide coefficient past 2^53", op: "/",
			left: 186079023766705.9, right: 5.71,
		},
		{
			name: "divide rounded coefficient", op: "/",
			left: 135250838509697.83, right: 0.88,
		},
		{name: "divide scale gap", op: "/", left: 1e-20, right: 3},
		{name: "divide huge integer", op: "/", left: 1e20, right: 0.7},
		{name: "remainder integers", op: "%", left: -7, right: 3},
		{name: "remainder to zero", op: "%", left: 1.1, right: 0.1},
		{name: "remainder negative divisor", op: "%", left: 5.5, right: -2},
		{name: "remainder fraction", op: "%", left: 7.5, right: 2.25},
		{name: "remainder scale gap", op: "%", left: 1e-20, right: 3},
		{name: "remainder huge integer", op: "%", left: 1e20, right: 0.7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertOracle(t, tc.op, tc.left, tc.right)
		})
	}
}

func TestDecimalBoundaries(t *testing.T) {
	magnitudes := []float64{
		0.1, 0.5, 1, 1.5, 7, 0.001, 0.15, 1299.99, 0.30000000000000004,
		1<<53 - 1, 1 << 53, 1<<52 + 0.5, 12345678901234.56,
		922337203685477500, 9.223372036854775e18, 1e20, 1e300,
		1e-20, 1e-23, 5e-324, 2.2250738585072014e-308, math.MaxFloat64,
	}
	values := make([]float64, 0, 2*len(magnitudes))
	for _, n := range magnitudes {
		values = append(values, n, -n)
	}
	for _, op := range operators {
		for _, left := range values {
			for _, right := range values {
				assertOracle(t, op, left, right)
			}
		}
	}
}

func TestDecimalSumPaths(t *testing.T) {
	for _, tc := range []sumCase{
		{name: "empty", items: []any{}},
		{name: "whole numbers", items: []any{1, 2, 3}},
		{name: "past 2^53", items: []any{1<<53 - 1, 2, -1}},
		{name: "fractions", items: []any{0.1, 0.2, 0.3}},
		{name: "mixed", items: []any{1, 0.5, 2.25}},
		{
			name:  "preserve integer prefix",
			items: []any{1<<53 - 1, 2, -(1<<53 - 1)},
		},
		{
			name:  "preserve fractional prefix",
			items: []any{0.1, 0.2, 1e20, -1e20},
		},
		{
			name:  "round accumulated coefficient once",
			items: []any{9007199254740991, 2, 0.1},
		},
		{
			name:  "cancel overflow",
			items: []any{math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64},
		},
		{name: "cancel to zero", items: []any{0.1, -0.1}},
		{
			name:  "decimal total overflow",
			items: []any{922337203685477500, 922337203685477500, 0.5},
		},
		{
			name:  "decimal total wraps",
			items: []any{0.1, 922337203685477500, 922337203685477500},
		},
		{name: "huge integer", items: []any{1e20, 0.5}},
		{name: "scale gap", items: []any{1e-20, 1}},
		{name: "overflow", items: []any{math.MaxFloat64, math.MaxFloat64}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			total := new(big.Rat)
			for _, item := range tc.items {
				total.Add(total, exactRat(toTestFloat(item)))
			}
			want, _ := total.Float64()
			value, err := argex.Eval("sum(items)", map[string]any{
				"items": tc.items,
			})
			if math.IsInf(want, 0) {
				assert.ErrorIs(t, err, argex.ErrArithmetic)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, want, value)
		})
	}
}

func TestDecimalSumValidation(t *testing.T) {
	for _, tc := range []sumCase{
		{name: "integer", items: []any{1, 2}},
		{name: "decimal", items: []any{0.1, 0.2}},
		{name: "rational", items: []any{1e20, 0.1}},
		{name: "overflow", items: []any{math.MaxFloat64, math.MaxFloat64}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, invalid := range []any{
				nil, "3", true, math.NaN(), math.Inf(1), json.Number("bad"),
			} {
				items := append(append([]any{}, tc.items...), invalid)
				_, err := argex.Eval("sum(this)", items)
				assert.ErrorIs(t, err, argex.ErrType)
			}
		})
	}
	value, err := argex.Eval("sum(this)", []any{
		json.Number("1"), json.Number("0.1"), json.Number("0.2"),
	})
	assert.NoError(t, err)
	assert.Equal(t, 1.3, value)
}

// assertOracle expects an operation to match the oracle, or to fail where
// the exact result overflows or the divisor is zero
func assertOracle(t *testing.T, op string, left, right float64) {
	t.Helper()
	value, err := argex.Eval("left "+op+" right", map[string]any{
		"left": left, "right": right,
	})
	if right == 0 && (op == "/" || op == "%") {
		assert.ErrorIs(t, err, argex.ErrArithmetic)
		return
	}
	want := oracle(op, left, right)
	if math.IsInf(want, 0) {
		assert.ErrorIs(t, err, argex.ErrArithmetic)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, want, value)
}

func toTestFloat(item any) float64 {
	if n, ok := item.(int); ok {
		return float64(n)
	}
	return item.(float64)
}

func randomDecimal(t *testing.T, random *rand.Rand) float64 {
	t.Helper()
	digits := 1 + random.IntN(maxOracleDigits)
	coef := random.Int64N(int64(math.Pow10(digits)))
	if random.IntN(2) == 0 {
		coef = -coef
	}
	exponent := random.IntN(2*maxOracleExponent+1) - maxOracleExponent
	text := strconv.FormatInt(coef, 10) + "e" + strconv.Itoa(exponent)
	n, err := strconv.ParseFloat(text, 64)
	assert.NoError(t, err)
	return n
}

// oracle computes each operation on the decimals the operands print as, with
// big.Rat alone
func oracle(op string, left, right float64) float64 {
	a := exactRat(left)
	b := exactRat(right)
	var r *big.Rat
	switch op {
	case "+":
		r = new(big.Rat).Add(a, b)

	case "-":
		r = new(big.Rat).Sub(a, b)

	case "*":
		r = new(big.Rat).Mul(a, b)

	case "/":
		r = new(big.Rat).Quo(a, b)

	default:
		quo := new(big.Rat).Quo(a, b)
		whole := new(big.Int).Quo(quo.Num(), quo.Denom())
		r = new(big.Rat).Sub(a, new(big.Rat).Mul(b, new(big.Rat).SetInt(whole)))
	}
	n, _ := r.Float64()
	return n
}

func exactRat(n float64) *big.Rat {
	r := new(big.Rat)
	r.SetString(strconv.FormatFloat(n, 'g', -1, 64))
	return r
}
