package engine

import (
	"fmt"
	"math"
	"reflect"
	"unicode/utf8"

	"github.com/kode4food/argex/internal/isa"
)

type (
	// Numeric results use the parallel stack to avoid boxing intermediates.
	numericValue struct{}

	operands struct {
		left  any
		right any
	}

	iterationSpec struct {
		slot int
		end  int
	}

	iteration struct {
		items  []any
		result []any
		index  int
		slot   int
		start  int
		filter bool
	}
)

const (
	inlineValues = 32
	inlineLoops  = 4
)

const (
	floatFractionBits = 52
	floatImplicitBit  = uint64(1) << floatFractionBits
	floatFractionMask = floatImplicitBit - 1
	floatSignMask     = uint64(1) << 63
)

func (p *program) evaluate(variables map[string]any) (any, error) {
	var values [inlineValues]any
	memory := values[:]
	var numbers []float64
	if size := p.stackSize + p.localCount; size > len(memory) {
		memory = make([]any, size)
	}
	if p.numeric {
		var scalars [inlineValues]float64
		numbers = scalars[:]
		if p.stackSize > len(numbers) {
			numbers = make([]float64, p.stackSize)
		}
	}
	stack := memory[:p.stackSize]
	locals := memory[p.stackSize : p.stackSize+p.localCount]
	var frames [inlineLoops]iteration
	loops := frames[:0]
	if p.localCount > cap(loops) {
		loops = make([]iteration, 0, p.localCount)
	}
	sp := 0
	for pc := 0; pc < len(p.code); pc++ {
		inst := p.code[pc]
		oc := inst.Opcode()
		operand := int(inst.Operand())
		switch oc {
		case isa.Const:
			stack[sp] = p.constants[operand]
			sp++

		case isa.Global:
			name := p.constants[operand].(string)
			value, ok := variables[name]
			if !ok {
				value = Missing{}
			}
			stack[sp] = value
			sp++

		case isa.Load:
			stack[sp] = locals[operand]
			sp++

		case isa.Access, isa.OptionalAccess, isa.Index, isa.OptionalIndex:
			var key any
			if oc == isa.Index || oc == isa.OptionalIndex {
				sp--
				key = stack[sp]
			} else {
				key = p.constants[operand]
			}
			optional := oc == isa.OptionalAccess || oc == isa.OptionalIndex
			value := stack[sp-1]
			if nullish(value) {
				if !optional {
					return nil, fmt.Errorf(
						"%w: access %v", MissingError, key,
					)
				}
				stack[sp-1] = Missing{}
				continue
			}
			switch container := value.(type) {
			case map[string]any:
				name, ok := key.(string)
				if !ok {
					return nil, operandError("object key")
				}
				if child, ok := container[name]; ok {
					stack[sp-1] = child
					continue
				}

			case []any:
				var index float64
				if _, ok := key.(numericValue); ok {
					index = numbers[sp]
				} else {
					index = number(key)
				}
				if index-index != 0 || math.Trunc(index) != index {
					return nil, operandError("array index")
				}
				if index >= 0 && index < float64(len(container)) {
					stack[sp-1] = container[int(index)]
					continue
				}

			default:
				return nil, operandError("access")
			}
			if !optional {
				return nil, fmt.Errorf("%w: key %v", MissingError, key)
			}
			stack[sp-1] = Missing{}

		case isa.Optional:
			if nullish(stack[sp-1]) {
				stack[sp-1] = Missing{}
				pc = operand - 1
			}

		case isa.Not:
			value, ok := stack[sp-1].(bool)
			if !ok {
				return nil, operandError("!")
			}
			stack[sp-1] = !value

		case isa.Pos:
			value := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				value = number(stack[sp-1])
			}
			if value-value != 0 {
				return nil, operandError("+")
			}
			numbers[sp-1] = value
			stack[sp-1] = numericValue{}

		case isa.Neg:
			value := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				value = number(stack[sp-1])
			}
			if value-value != 0 {
				return nil, operandError("-")
			}
			numbers[sp-1] = -value
			stack[sp-1] = numericValue{}

		case isa.Add:
			if a, ok := stack[sp-2].(string); ok {
				b, ok := stack[sp-1].(string)
				if !ok {
					return nil, operandError("+")
				}
				sp--
				stack[sp-1] = a + b
				continue
			}
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError("+")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError("+")
			}
			value := a + b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: +", ArithmeticError)
			}
			sp--
			numbers[sp-1] = value
			stack[sp-1] = numericValue{}

		case isa.Sub:
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError("-")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError("-")
			}
			value := a - b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: -", ArithmeticError)
			}
			sp--
			numbers[sp-1] = value
			stack[sp-1] = numericValue{}

		case isa.Mul:
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError("*")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError("*")
			}
			value := a * b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: *", ArithmeticError)
			}
			sp--
			numbers[sp-1] = value
			stack[sp-1] = numericValue{}

		case isa.Div:
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError("/")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError("/")
			}
			value := a / b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: /", ArithmeticError)
			}
			sp--
			numbers[sp-1] = value
			stack[sp-1] = numericValue{}

		case isa.Mod:
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError("%")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError("%")
			}
			if b == 0 {
				return nil, fmt.Errorf("%w: %%", ArithmeticError)
			}
			value := a
			x := math.Float64bits(a) &^ floatSignMask
			y := math.Float64bits(b) &^ floatSignMask
			if x >= y {
				ex := int(x >> floatFractionBits)
				ey := int(y >> floatFractionBits)
				if ex == 0 {
					ex = 1
					for x < floatImplicitBit {
						x <<= 1
						ex--
					}
				} else {
					x = x&floatFractionMask | floatImplicitBit
				}
				if ey == 0 {
					ey = 1
					for y < floatImplicitBit {
						y <<= 1
						ey--
					}
				} else {
					y = y&floatFractionMask | floatImplicitBit
				}
				// Align significands and subtract without rounding a/b.
				for ex > ey && x != 0 {
					if x >= y {
						x -= y
					}
					x <<= 1
					ex--
				}
				if x >= y {
					x -= y
				}
				if x != 0 {
					for x < floatImplicitBit {
						x <<= 1
						ex--
					}
					if ex > 0 {
						x = x&floatFractionMask | uint64(ex)<<floatFractionBits
					} else {
						x >>= uint(1 - ex)
					}
				}
				sign := math.Float64bits(a) & floatSignMask
				value = math.Float64frombits(x | sign)
			}
			sp--
			numbers[sp-1] = value
			stack[sp-1] = numericValue{}

		case isa.Eq, isa.Neq, isa.In:
			_, leftNumeric := stack[sp-2].(numericValue)
			_, rightNumeric := stack[sp-1].(numericValue)
			if oc != isa.In && (leftNumeric || rightNumeric) {
				a := numbers[sp-2]
				if !leftNumeric {
					a = number(stack[sp-2])
				}
				b := numbers[sp-1]
				if !rightNumeric {
					b = number(stack[sp-1])
				}
				sp--
				stack[sp-1] = (a == b) == (oc == isa.Eq)
				continue
			}
			left := stack[sp-2]
			if leftNumeric {
				left = numbers[sp-2]
			}
			candidates := stack[sp-1 : sp]
			if oc == isa.In {
				var ok bool
				candidates, ok = stack[sp-1].([]any)
				if !ok {
					return nil, operandError("in")
				}
			}
			var pairs [inlineValues]operands
			matched := false
			for _, candidate := range candidates {
				pairs[0] = operands{left: left, right: candidate}
				pending := pairs[:1]
				matched = true
				for len(pending) > 0 && matched {
					last := len(pending) - 1
					pair := pending[last]
					pending = pending[:last]
					switch left := pair.left.(type) {
					case nil:
						matched = pair.right == nil

					case Missing:
						_, matched = pair.right.(Missing)

					case bool:
						right, ok := pair.right.(bool)
						matched = ok && left == right

					case string:
						right, ok := pair.right.(string)
						matched = ok && left == right

					case []any:
						right, ok := pair.right.([]any)
						if !ok || len(left) != len(right) {
							matched = false
							continue
						}
						for i, item := range left {
							pending = append(pending, operands{
								left:  item,
								right: right[i],
							})
						}

					case map[string]any:
						right, ok := pair.right.(map[string]any)
						if !ok || len(left) != len(right) {
							matched = false
							continue
						}
						for key, item := range left {
							child, ok := right[key]
							if !ok {
								matched = false
								break
							}
							pending = append(pending, operands{
								left:  item,
								right: child,
							})
						}

					default:
						a := number(pair.left)
						b := number(pair.right)
						if a-a == 0 {
							matched = b-b == 0 && a == b
							continue
						}
						matched = reflect.DeepEqual(
							pair.left, pair.right,
						)
					}
				}
				if matched {
					break
				}
			}
			sp--
			stack[sp-1] = matched != (oc == isa.Neq)

		case isa.Lt:
			if a, ok := stack[sp-2].(string); ok {
				b, ok := stack[sp-1].(string)
				if !ok {
					return nil, operandError("<")
				}
				sp--
				stack[sp-1] = a < b
				continue
			}
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError("<")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError("<")
			}
			sp--
			stack[sp-1] = a < b

		case isa.Lte:
			if a, ok := stack[sp-2].(string); ok {
				b, ok := stack[sp-1].(string)
				if !ok {
					return nil, operandError("<=")
				}
				sp--
				stack[sp-1] = a <= b
				continue
			}
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError("<=")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError("<=")
			}
			sp--
			stack[sp-1] = a <= b

		case isa.Gt:
			if a, ok := stack[sp-2].(string); ok {
				b, ok := stack[sp-1].(string)
				if !ok {
					return nil, operandError(">")
				}
				sp--
				stack[sp-1] = a > b
				continue
			}
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError(">")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError(">")
			}
			sp--
			stack[sp-1] = a > b

		case isa.Gte:
			if a, ok := stack[sp-2].(string); ok {
				b, ok := stack[sp-1].(string)
				if !ok {
					return nil, operandError(">=")
				}
				sp--
				stack[sp-1] = a >= b
				continue
			}
			a := numbers[sp-2]
			if _, ok := stack[sp-2].(numericValue); !ok {
				a = number(stack[sp-2])
			}
			if a-a != 0 {
				return nil, operandError(">=")
			}
			b := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				b = number(stack[sp-1])
			}
			if b-b != 0 {
				return nil, operandError(">=")
			}
			sp--
			stack[sp-1] = a >= b

		case isa.And, isa.Or:
			value, ok := stack[sp-1].(bool)
			if !ok {
				return nil, operandError("Boolean operand")
			}
			if oc == isa.And && !value || oc == isa.Or && value {
				pc = operand - 1
			} else {
				sp--
			}

		case isa.Coalesce:
			if !nullish(stack[sp-1]) {
				pc = operand - 1
			} else {
				sp--
			}

		case isa.Boolean:
			if _, ok := stack[sp-1].(bool); !ok {
				return nil, operandError(p.constants[operand].(string))
			}

		case isa.CondJump:
			sp--
			value, ok := stack[sp].(bool)
			if !ok {
				return nil, operandError("case condition")
			}
			if !value {
				pc = operand - 1
			}

		case isa.Jump:
			pc = operand - 1

		case isa.Array:
			base := sp - operand
			for i := base; i < sp; i++ {
				if _, ok := stack[i].(numericValue); ok {
					stack[i] = numbers[i]
				}
			}
			result := append([]any{}, stack[base:sp]...)
			stack[base] = result
			sp = base + 1

		case isa.Object:
			keys := p.constants[operand].([]string)
			base := sp - len(keys)
			result := make(map[string]any, len(keys))
			for i, key := range keys {
				if _, ok := stack[base+i].(numericValue); ok {
					stack[base+i] = numbers[base+i]
				}
				if value := stack[base+i]; !IsMissing(value) {
					result[key] = value
				}
			}
			stack[base] = result
			sp = base + 1

		case isa.Length:
			var count int
			switch value := stack[sp-1].(type) {
			case string:
				count = utf8.RuneCountInString(value)

			case []any:
				count = len(value)

			case map[string]any:
				count = len(value)

			default:
				return nil, operandError("length")
			}
			numbers[sp-1] = float64(count)
			stack[sp-1] = numericValue{}

		case isa.Abs:
			value := numbers[sp-1]
			if _, ok := stack[sp-1].(numericValue); !ok {
				value = number(stack[sp-1])
			}
			if value-value != 0 {
				return nil, operandError("abs")
			}
			numbers[sp-1] = math.Abs(value)
			stack[sp-1] = numericValue{}

		case isa.Min:
			base := sp - operand
			result := numbers[base]
			if _, ok := stack[base].(numericValue); !ok {
				result = number(stack[base])
			}
			if result-result != 0 {
				return nil, operandError("min")
			}
			for i := base + 1; i < sp; i++ {
				n := numbers[i]
				if _, ok := stack[i].(numericValue); !ok {
					n = number(stack[i])
				}
				if n-n != 0 {
					return nil, operandError("min")
				}
				result = min(result, n)
			}
			numbers[base] = result
			stack[base] = numericValue{}
			sp = base + 1

		case isa.Max:
			base := sp - operand
			result := numbers[base]
			if _, ok := stack[base].(numericValue); !ok {
				result = number(stack[base])
			}
			if result-result != 0 {
				return nil, operandError("max")
			}
			for i := base + 1; i < sp; i++ {
				n := numbers[i]
				if _, ok := stack[i].(numericValue); !ok {
					n = number(stack[i])
				}
				if n-n != 0 {
					return nil, operandError("max")
				}
				result = max(result, n)
			}
			numbers[base] = result
			stack[base] = numericValue{}
			sp = base + 1

		case isa.Map, isa.Filter:
			items, ok := stack[sp-1].([]any)
			if !ok {
				return nil, operandError("collection receiver")
			}
			spec := p.iterations[operand]
			if len(items) == 0 {
				stack[sp-1] = []any{}
				pc = spec.end - 1
				continue
			}
			loops = append(loops, iteration{
				items:  items,
				result: []any{},
				slot:   spec.slot,
				start:  pc + 1,
				filter: oc == isa.Filter,
			})
			locals[spec.slot] = items[0]
			sp--

		case isa.Next:
			frame := &loops[len(loops)-1]
			sp--
			value := stack[sp]
			if _, ok := value.(numericValue); ok {
				value = numbers[sp]
			}
			keep, ok := value.(bool)
			if frame.filter && !ok {
				return nil, operandError("filter predicate")
			}
			if frame.filter {
				value = frame.items[frame.index]
			}
			if !frame.filter || keep {
				frame.result = append(frame.result, value)
			}
			frame.index++
			if frame.index < len(frame.items) {
				locals[frame.slot] = frame.items[frame.index]
				pc = frame.start - 1
				continue
			}
			stack[sp] = frame.result
			sp++
			loops = loops[:len(loops)-1]
		}
	}
	if _, ok := stack[0].(numericValue); ok {
		return numbers[0], nil
	}
	return stack[0], nil
}
