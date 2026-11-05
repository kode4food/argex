package engine

import (
	"bytes"
	"math"
	"math/big"
	"math/bits"
	"strconv"
)

type (
	// decimal is a finite number as the decimal it prints as, which is
	// coef × 10^-scale
	decimal struct {
		coef  int64
		scale int
	}

	// decimalPair holds two operands brought to the same scale
	decimalPair struct {
		left  decimal
		right decimal
	}
)

const (
	// Powers of ten up to 10^22 are exact as floats, so dividing an exact
	// coefficient by one rounds once, to the nearest float
	maxExactScale = 22

	decimalTextSize = 32
)

var intPow10 = [...]int64{
	1, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11, 1e12, 1e13,
	1e14, 1e15, 1e16, 1e17, 1e18,
}

func (d decimal) add(other decimal) (decimal, bool) {
	pair, ok := alignDecimals(d, other)
	if !ok {
		return decimal{}, false
	}
	coef, ok := addInt64(pair.left.coef, pair.right.coef)
	return decimal{coef: coef, scale: pair.left.scale}, ok
}

func (d decimal) float() (float64, bool) {
	if d.scale > maxExactScale || !isExactCoef(d.coef) {
		return 0, false
	}
	return float64(d.coef) / math.Pow10(d.scale), true
}

func (d decimal) rat() *big.Rat {
	var denominator big.Int
	denominator.Exp(big.NewInt(10), big.NewInt(int64(d.scale)), nil)
	return new(big.Rat).SetFrac(big.NewInt(d.coef), &denominator)
}

func decimalAdd(a, b float64) (float64, bool) {
	pair, ok := toDecimalPair(a, b)
	if !ok {
		return 0, false
	}
	coef, ok := addInt64(pair.left.coef, pair.right.coef)
	if !ok {
		return 0, false
	}
	return decimal{coef: coef, scale: pair.left.scale}.float()
}

func decimalMul(a, b float64) (float64, bool) {
	left, ok := toDecimal(a)
	if !ok {
		return 0, false
	}
	right, ok := toDecimal(b)
	if !ok {
		return 0, false
	}
	coef, ok := mulInt64(left.coef, right.coef)
	if !ok {
		return 0, false
	}
	return decimal{coef: coef, scale: left.scale + right.scale}.float()
}

// decimalQuo divides coefficients at a shared scale, so the quotient is a ratio
// of exact integers that IEEE division rounds once
func decimalQuo(a, b float64) (float64, bool) {
	pair, ok := toDecimalPair(a, b)
	if !ok {
		return 0, false
	}
	left := pair.left.coef
	right := pair.right.coef
	if !isExactCoef(left) || !isExactCoef(right) {
		return 0, false
	}
	return float64(left) / float64(right), true
}

func decimalMod(a, b float64) (float64, bool) {
	pair, ok := toDecimalPair(a, b)
	if !ok {
		return 0, false
	}
	coef := pair.left.coef % pair.right.coef
	return decimal{coef: coef, scale: pair.left.scale}.float()
}

func decimalSum(items []any, total decimal) (float64, error) {
	for i, item := range items {
		n := number(item)
		if n-n != 0 {
			return 0, operandError(fnSum)
		}
		d, ok := toDecimal(n)
		if !ok {
			return rationalSum(items[i:], total.rat())
		}
		next, ok := total.add(d)
		if !ok {
			return rationalSum(items[i:], total.rat())
		}
		total = next
	}
	if result, ok := total.float(); ok {
		return result, nil
	}
	return toFloat(total.rat()), nil
}

func toDecimalPair(a, b float64) (decimalPair, bool) {
	left, ok := toDecimal(a)
	if !ok {
		return decimalPair{}, false
	}
	right, ok := toDecimal(b)
	if !ok {
		return decimalPair{}, false
	}
	return alignDecimals(left, right)
}

func alignDecimals(left, right decimal) (decimalPair, bool) {
	left, ok := rescale(left, right.scale)
	if !ok {
		return decimalPair{}, false
	}
	right, ok = rescale(right, left.scale)
	return decimalPair{left: left, right: right}, ok
}

func rescale(d decimal, scale int) (decimal, bool) {
	if d.scale >= scale {
		return d, true
	}
	if scale-d.scale >= len(intPow10) {
		return decimal{}, false
	}
	coef, ok := mulInt64(d.coef, intPow10[scale-d.scale])
	return decimal{coef: coef, scale: scale}, ok
}

// toDecimal reads the shortest decimal a float prints as, the number a user
// wrote, without allocating
func toDecimal(n float64) (decimal, bool) {
	if isExactInt(n) {
		return decimal{coef: int64(n)}, true
	}
	var buf [decimalTextSize]byte
	text := strconv.AppendFloat(buf[:0], n, 'e', -1, 64)
	mark := bytes.IndexByte(text, 'e')
	coef, fraction := parseMantissa(text[:mark])
	scale := fraction - parseExponent(text[mark+1:])
	if scale >= 0 {
		return decimal{coef: coef, scale: scale}, true
	}
	return rescale(decimal{coef: coef, scale: scale}, 0)
}

// parseMantissa reads at most 17 significant digits, which fit an int64
func parseMantissa(text []byte) (int64, int) {
	negative := text[0] == '-'
	if negative {
		text = text[1:]
	}
	fraction := 0
	if point := bytes.IndexByte(text, '.'); point >= 0 {
		fraction = len(text) - point - 1
	}
	var coef int64
	for _, c := range text {
		if c == '.' {
			continue
		}
		coef = coef*10 + int64(c-'0')
	}
	if negative {
		return -coef, fraction
	}
	return coef, fraction
}

func parseExponent(text []byte) int {
	negative := text[0] == '-'
	exponent := 0
	for _, c := range text[1:] {
		exponent = exponent*10 + int(c-'0')
	}
	if negative {
		return -exponent
	}
	return exponent
}

func addInt64(a, b int64) (int64, bool) {
	sum := a + b
	return sum, (sum > a) == (b > 0)
}

func mulInt64(a, b int64) (int64, bool) {
	hi, lo := bits.Mul64(absInt64(a), absInt64(b))
	if hi != 0 || lo > math.MaxInt64 {
		return 0, false
	}
	if (a < 0) != (b < 0) {
		return -int64(lo), true
	}
	return int64(lo), true
}

func absInt64(n int64) uint64 {
	if n < 0 {
		return uint64(-n)
	}
	return uint64(n)
}

func isExactCoef(coef int64) bool {
	return coef > -exactIntLimit && coef < exactIntLimit
}
