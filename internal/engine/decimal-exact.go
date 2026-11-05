package engine

import (
	"math"
	"math/big"
	"strconv"
)

// exactIntLimit bounds the integers that are exact as floats, so IEEE
// arithmetic on two of them already rounds the exact result to nearest
const exactIntLimit = 1 << 53

// Arithmetic runs on the decimal each finite operand prints as, so 0.1 + 0.2
// is 0.3. A zero result keeps the sign IEEE 754 arithmetic would give it

func exactAdd(a, b float64) float64 {
	if isExactInt(a) && isExactInt(b) {
		return a + b
	}
	if r, ok := decimalAdd(a, b); ok {
		return nonZeroOr(r, a+b)
	}
	return nonZeroOr(toFloat(new(big.Rat).Add(toExact(a), toExact(b))), a+b)
}

func exactSub(a, b float64) float64 {
	if isExactInt(a) && isExactInt(b) {
		return a - b
	}
	if r, ok := decimalAdd(a, -b); ok {
		return nonZeroOr(r, a-b)
	}
	return nonZeroOr(toFloat(new(big.Rat).Sub(toExact(a), toExact(b))), a-b)
}

func exactMul(a, b float64) float64 {
	if isExactInt(a) && isExactInt(b) {
		return a * b
	}
	if r, ok := decimalMul(a, b); ok {
		return nonZeroOr(r, a*b)
	}
	return nonZeroOr(toFloat(new(big.Rat).Mul(toExact(a), toExact(b))), a*b)
}

// exactQuo requires a non-zero divisor. IEEE division of two exact integers
// already rounds the exact quotient to nearest
func exactQuo(a, b float64) float64 {
	if isExactInt(a) && isExactInt(b) {
		return a / b
	}
	if r, ok := decimalQuo(a, b); ok {
		return nonZeroOr(r, a/b)
	}
	return nonZeroOr(toFloat(new(big.Rat).Quo(toExact(a), toExact(b))), a/b)
}

// exactMod truncates the quotient, so the result takes the dividend's sign as
// math.Mod does. It requires a non-zero divisor
func exactMod(a, b float64) float64 {
	if isExactInt(a) && isExactInt(b) {
		return math.Mod(a, b)
	}
	zero := math.Copysign(0, a)
	if r, ok := decimalMod(a, b); ok {
		return nonZeroOr(r, zero)
	}
	dividend := toExact(a)
	divisor := toExact(b)
	quo := new(big.Rat).Quo(dividend, divisor)
	whole := new(big.Int).Quo(quo.Num(), quo.Denom())
	taken := new(big.Rat).Mul(divisor, new(big.Rat).SetInt(whole))
	return nonZeroOr(toFloat(taken.Sub(dividend, taken)), zero)
}

// exactSum totals finite numbers exactly and rounds once, at one shared scale
// while the total fits an int64
func exactSum(items []any) (float64, error) {
	total := 0.0
	for i, item := range items {
		n := number(item)
		if n-n != 0 {
			return 0, operandError(fnSum)
		}
		next := total + n
		if !isExactInt(n) || !isExactInt(next) {
			return decimalSum(items[i:], decimal{coef: int64(total)})
		}
		total = next
	}
	return total, nil
}

func rationalSum(items []any, total *big.Rat) (float64, error) {
	for _, item := range items {
		n := number(item)
		if n-n != 0 {
			return 0, operandError(fnSum)
		}
		total.Add(total, toExact(n))
	}
	return toFloat(total), nil
}

func nonZeroOr(r, zero float64) float64 {
	if r == 0 {
		return math.Copysign(0, zero)
	}
	return r
}

func isExactInt(n float64) bool {
	return n == math.Trunc(n) && math.Abs(n) < exactIntLimit
}

// toExact reads the decimal a finite float prints as, which big.Rat always
// parses
func toExact(n float64) *big.Rat {
	r := new(big.Rat)
	r.SetString(strconv.FormatFloat(n, 'g', -1, 64))
	return r
}

func toFloat(r *big.Rat) float64 {
	n, _ := r.Float64()
	return n
}
