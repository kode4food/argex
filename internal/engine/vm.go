package engine

import (
	"fmt"
	"math"
	"slices"

	"github.com/kode4food/argex/internal/isa"
)

type (
	program struct {
		code       isa.Instructions
		constants  []any
		iterations []iterationSpec
		stackSize  int
		localCount int
	}

	iterationSpec struct {
		valueSlot int
		keySlot   int
		end       int
		keyed     bool
	}

	callSpec struct {
		call  function
		arity int
	}

	// iterator is a comprehension in progress, held as one operand-stack
	// value: Fetch advances it, Append grows result, Result extracts it
	iterator struct {
		keys   []any
		values []any
		result []any
	}
)

const inlineValues = 32

func (p *program) evaluate(variables map[string]any) (any, error) {
	var values [inlineValues]any
	memory := values[:]
	var arguments []any
	if size := p.stackSize + p.localCount; size > len(memory) {
		memory = make([]any, size)
	}
	stack := memory[:p.stackSize]
	locals := memory[p.stackSize : p.stackSize+p.localCount]
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

		case isa.Access:
			key := p.constants[operand]
			value := stack[sp-1]
			if nullish(value) {
				return nil, fmt.Errorf("%w: access %v", ErrMissing, key)
			}
			child, found, err := accessChild(value, key)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, fmt.Errorf("%w: key %v", ErrMissing, key)
			}
			stack[sp-1] = child

		case isa.OptionalAccess:
			key := p.constants[operand]
			value := stack[sp-1]
			if nullish(value) {
				stack[sp-1] = Missing{}
				continue
			}
			child, found, err := accessChild(value, key)
			if err != nil {
				return nil, err
			}
			if !found {
				stack[sp-1] = Missing{}
				continue
			}
			stack[sp-1] = child

		case isa.Index:
			sp--
			key := stack[sp]
			value := stack[sp-1]
			if nullish(value) {
				return nil, fmt.Errorf("%w: access %v", ErrMissing, key)
			}
			child, found, err := accessChild(value, key)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, fmt.Errorf("%w: key %v", ErrMissing, key)
			}
			stack[sp-1] = child

		case isa.OptionalIndex:
			sp--
			key := stack[sp]
			value := stack[sp-1]
			if nullish(value) {
				stack[sp-1] = Missing{}
				continue
			}
			child, found, err := accessChild(value, key)
			if err != nil {
				return nil, err
			}
			if !found {
				stack[sp-1] = Missing{}
				continue
			}
			stack[sp-1] = child

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
			value := number(stack[sp-1])
			if value-value != 0 {
				return nil, operandError("+")
			}
			stack[sp-1] = value

		case isa.Neg:
			value := number(stack[sp-1])
			if value-value != 0 {
				return nil, operandError("-")
			}
			stack[sp-1] = -value

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
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError("+")
			}
			b := number(stack[sp-1])
			if b-b != 0 {
				return nil, operandError("+")
			}
			value := a + b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: +", ErrArithmetic)
			}
			sp--
			stack[sp-1] = value

		case isa.Sub:
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError("-")
			}
			b := number(stack[sp-1])
			if b-b != 0 {
				return nil, operandError("-")
			}
			value := a - b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: -", ErrArithmetic)
			}
			sp--
			stack[sp-1] = value

		case isa.Mul:
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError("*")
			}
			b := number(stack[sp-1])
			if b-b != 0 {
				return nil, operandError("*")
			}
			value := a * b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: *", ErrArithmetic)
			}
			sp--
			stack[sp-1] = value

		case isa.Div:
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError("/")
			}
			b := number(stack[sp-1])
			if b-b != 0 {
				return nil, operandError("/")
			}
			value := a / b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: /", ErrArithmetic)
			}
			sp--
			stack[sp-1] = value

		case isa.Mod:
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError("%")
			}
			b := number(stack[sp-1])
			if b-b != 0 {
				return nil, operandError("%")
			}
			value := math.Mod(a, b)
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: %%", ErrArithmetic)
			}
			sp--
			stack[sp-1] = value

		case isa.Eq:
			left := stack[sp-2]
			right := stack[sp-1]
			sp--
			stack[sp-1] = structuralEqual(left, right)

		case isa.Neq:
			left := stack[sp-2]
			right := stack[sp-1]
			sp--
			stack[sp-1] = !structuralEqual(left, right)

		case isa.In:
			left := stack[sp-2]
			candidates, ok := stack[sp-1].([]any)
			if !ok {
				return nil, operandError("in")
			}
			matched := false
			for _, candidate := range candidates {
				if structuralEqual(left, candidate) {
					matched = true
					break
				}
			}
			sp--
			stack[sp-1] = matched

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
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError("<")
			}
			b := number(stack[sp-1])
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
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError("<=")
			}
			b := number(stack[sp-1])
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
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError(">")
			}
			b := number(stack[sp-1])
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
			a := number(stack[sp-2])
			if a-a != 0 {
				return nil, operandError(">=")
			}
			b := number(stack[sp-1])
			if b-b != 0 {
				return nil, operandError(">=")
			}
			sp--
			stack[sp-1] = a >= b

		case isa.And:
			value, ok := stack[sp-1].(bool)
			if !ok {
				return nil, operandError("Boolean operand")
			}
			if !value {
				pc = operand - 1
			} else {
				sp--
			}

		case isa.Or:
			value, ok := stack[sp-1].(bool)
			if !ok {
				return nil, operandError("Boolean operand")
			}
			if value {
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
			result := append([]any{}, stack[base:sp]...)
			stack[base] = result
			sp = base + 1

		case isa.Object:
			keys := p.constants[operand].([]string)
			base := sp - len(keys)
			result := make(map[string]any, len(keys))
			for i, key := range keys {
				if value := stack[base+i]; !IsMissing(value) {
					result[key] = value
				}
			}
			stack[base] = result
			sp = base + 1

		case isa.Push:
			sp--
			value := stack[sp]
			stack[sp-1] = append(stack[sp-1].([]any), value)

		case isa.Put:
			sp--
			value := stack[sp]
			if !IsMissing(value) {
				key := p.constants[operand].(string)
				stack[sp-1].(map[string]any)[key] = value
			}

		case isa.Concat:
			sp--
			if nullish(stack[sp]) {
				continue
			}
			source, ok := stack[sp].([]any)
			if !ok {
				return nil, operandError("spread")
			}
			stack[sp-1] = append(stack[sp-1].([]any), source...)

		case isa.Merge:
			sp--
			if nullish(stack[sp]) {
				continue
			}
			source, ok := stack[sp].(map[string]any)
			if !ok {
				return nil, operandError("spread")
			}
			target := stack[sp-1].(map[string]any)
			for key, value := range source {
				if !IsMissing(value) {
					target[key] = value
				}
			}

		case isa.Call:
			spec := p.constants[operand].(callSpec)
			base := sp - spec.arity
			if cap(arguments) < spec.arity {
				arguments = make([]any, spec.arity)
			}
			args := arguments[:spec.arity]
			for i := range args {
				value := stack[base+i]
				args[i] = value
			}
			result, err := spec.call(args...)
			if err != nil {
				return nil, err
			}
			stack[base] = result
			sp = base + 1

		case isa.Iterate:
			spec := p.iterations[operand]
			keys, values, err := collectionEntries(
				stack[sp-1], spec.keyed,
			)
			if err != nil {
				return nil, err
			}
			stack[sp-1] = &iterator{
				keys:   keys,
				values: values,
				result: []any{},
			}

		case isa.Fetch:
			spec := p.iterations[operand]
			it := stack[sp-1].(*iterator)
			if len(it.values) == 0 {
				pc = spec.end - 1
				continue
			}
			locals[spec.valueSlot] = it.values[0]
			it.values = it.values[1:]
			if spec.keyed {
				locals[spec.keySlot] = it.keys[0]
				it.keys = it.keys[1:]
			}

		case isa.Append:
			sp--
			value := stack[sp]
			it := stack[sp-1].(*iterator)
			it.result = append(it.result, value)

		case isa.Result:
			it := stack[sp-1].(*iterator)
			stack[sp-1] = it.result

		case isa.Pop:
			sp--
		}
	}
	return stack[0], nil
}

func accessChild(container, key any) (any, bool, error) {
	switch c := container.(type) {
	case map[string]any:
		name, ok := key.(string)
		if !ok {
			return nil, false, operandError("object key")
		}
		child, ok := c[name]
		return child, ok, nil

	case []any:
		index := number(key)
		if index-index != 0 || math.Trunc(index) != index {
			return nil, false, operandError("array index")
		}
		if index >= 0 && index < float64(len(c)) {
			return c[int(index)], true, nil
		}
		return nil, false, nil

	default:
		return nil, false, operandError("access")
	}
}

// collectionEntries materializes a comprehension receiver into parallel
// key/value slices; keys is nil when the binding form doesn't need it
func collectionEntries(receiver any, keyed bool) ([]any, []any, error) {
	switch c := receiver.(type) {
	case []any:
		var keys []any
		if keyed {
			keys = make([]any, len(c))
			for i := range c {
				keys[i] = float64(i)
			}
		}
		return keys, c, nil

	case map[string]any:
		if !keyed {
			return nil, nil, operandError("collection receiver")
		}
		names := make([]string, 0, len(c))
		for name := range c {
			names = append(names, name)
		}
		slices.Sort(names)
		keys := make([]any, len(names))
		values := make([]any, len(names))
		for i, name := range names {
			keys[i] = name
			values[i] = c[name]
		}
		return keys, values, nil

	default:
		return nil, nil, operandError("collection receiver")
	}
}
