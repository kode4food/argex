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
	MEM := values[:]
	var arguments []any
	if size := p.stackSize + p.localCount; size > len(MEM) {
		MEM = make([]any, size)
	}
	CODE := p.code
	LP := p.stackSize
	SP := 0
	for PC := 0; PC < len(CODE); PC++ {
		INST := CODE[PC]
		op := int(INST.Operand())
		switch INST.Opcode() {
		case isa.Const:
			MEM[SP] = p.constants[op]
			SP++

		case isa.Global:
			name := p.constants[op].(string)
			value, ok := variables[name]
			if !ok {
				value = Missing{}
			}
			MEM[SP] = value
			SP++

		case isa.Load:
			MEM[SP] = MEM[LP+op]
			SP++

		case isa.Store:
			SP--
			MEM[LP+op] = MEM[SP]

		case isa.Access:
			key := p.constants[op]
			value := MEM[SP-1]
			if nullish(value) {
				return nil, fmt.Errorf("%w: access %v", ErrMissing, key)
			}
			child, ok, err := accessChild(value, key)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("%w: key %v", ErrMissing, key)
			}
			MEM[SP-1] = child

		case isa.OptionalAccess:
			key := p.constants[op]
			value := MEM[SP-1]
			if nullish(value) {
				MEM[SP-1] = Missing{}
				continue
			}
			child, ok, err := accessChild(value, key)
			if err != nil {
				return nil, err
			}
			if !ok {
				MEM[SP-1] = Missing{}
				continue
			}
			MEM[SP-1] = child

		case isa.Index:
			SP--
			key := MEM[SP]
			value := MEM[SP-1]
			if nullish(value) {
				return nil, fmt.Errorf("%w: access %v", ErrMissing, key)
			}
			child, ok, err := accessChild(value, key)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("%w: key %v", ErrMissing, key)
			}
			MEM[SP-1] = child

		case isa.OptionalIndex:
			SP--
			key := MEM[SP]
			value := MEM[SP-1]
			if nullish(value) {
				MEM[SP-1] = Missing{}
				continue
			}
			child, ok, err := accessChild(value, key)
			if err != nil {
				return nil, err
			}
			if !ok {
				MEM[SP-1] = Missing{}
				continue
			}
			MEM[SP-1] = child

		case isa.Optional:
			if nullish(MEM[SP-1]) {
				MEM[SP-1] = Missing{}
				PC = op - 1
			}

		case isa.Not:
			value, ok := MEM[SP-1].(bool)
			if !ok {
				return nil, operandError("!")
			}
			MEM[SP-1] = !value

		case isa.Pos:
			value := number(MEM[SP-1])
			if value-value != 0 {
				return nil, operandError("+")
			}
			MEM[SP-1] = value

		case isa.Neg:
			value := number(MEM[SP-1])
			if value-value != 0 {
				return nil, operandError("-")
			}
			MEM[SP-1] = -value

		case isa.Add:
			if a, ok := MEM[SP-2].(string); ok {
				b, ok := MEM[SP-1].(string)
				if !ok {
					return nil, operandError("+")
				}
				SP--
				MEM[SP-1] = a + b
				continue
			}
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError("+")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError("+")
			}
			value := a + b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: +", ErrArithmetic)
			}
			SP--
			MEM[SP-1] = value

		case isa.Sub:
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError("-")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError("-")
			}
			value := a - b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: -", ErrArithmetic)
			}
			SP--
			MEM[SP-1] = value

		case isa.Mul:
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError("*")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError("*")
			}
			value := a * b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: *", ErrArithmetic)
			}
			SP--
			MEM[SP-1] = value

		case isa.Div:
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError("/")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError("/")
			}
			value := a / b
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: /", ErrArithmetic)
			}
			SP--
			MEM[SP-1] = value

		case isa.Mod:
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError("%")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError("%")
			}
			value := math.Mod(a, b)
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("%w: %%", ErrArithmetic)
			}
			SP--
			MEM[SP-1] = value

		case isa.Eq:
			left := MEM[SP-2]
			right := MEM[SP-1]
			SP--
			MEM[SP-1] = structuralEqual(left, right)

		case isa.Neq:
			left := MEM[SP-2]
			right := MEM[SP-1]
			SP--
			MEM[SP-1] = !structuralEqual(left, right)

		case isa.In:
			left := MEM[SP-2]
			candidates, ok := MEM[SP-1].([]any)
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
			SP--
			MEM[SP-1] = matched

		case isa.Lt:
			if a, ok := MEM[SP-2].(string); ok {
				b, ok := MEM[SP-1].(string)
				if !ok {
					return nil, operandError("<")
				}
				SP--
				MEM[SP-1] = a < b
				continue
			}
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError("<")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError("<")
			}
			SP--
			MEM[SP-1] = a < b

		case isa.Lte:
			if a, ok := MEM[SP-2].(string); ok {
				b, ok := MEM[SP-1].(string)
				if !ok {
					return nil, operandError("<=")
				}
				SP--
				MEM[SP-1] = a <= b
				continue
			}
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError("<=")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError("<=")
			}
			SP--
			MEM[SP-1] = a <= b

		case isa.Gt:
			if a, ok := MEM[SP-2].(string); ok {
				b, ok := MEM[SP-1].(string)
				if !ok {
					return nil, operandError(">")
				}
				SP--
				MEM[SP-1] = a > b
				continue
			}
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError(">")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError(">")
			}
			SP--
			MEM[SP-1] = a > b

		case isa.Gte:
			if a, ok := MEM[SP-2].(string); ok {
				b, ok := MEM[SP-1].(string)
				if !ok {
					return nil, operandError(">=")
				}
				SP--
				MEM[SP-1] = a >= b
				continue
			}
			a := number(MEM[SP-2])
			if a-a != 0 {
				return nil, operandError(">=")
			}
			b := number(MEM[SP-1])
			if b-b != 0 {
				return nil, operandError(">=")
			}
			SP--
			MEM[SP-1] = a >= b

		case isa.And:
			value, ok := MEM[SP-1].(bool)
			if !ok {
				return nil, operandError("Boolean operand")
			}
			if !value {
				PC = op - 1
			} else {
				SP--
			}

		case isa.Or:
			value, ok := MEM[SP-1].(bool)
			if !ok {
				return nil, operandError("Boolean operand")
			}
			if value {
				PC = op - 1
			} else {
				SP--
			}

		case isa.Coalesce:
			if !nullish(MEM[SP-1]) {
				PC = op - 1
			} else {
				SP--
			}

		case isa.Boolean:
			if _, ok := MEM[SP-1].(bool); !ok {
				return nil, operandError(p.constants[op].(string))
			}

		case isa.CondJump:
			SP--
			value, ok := MEM[SP].(bool)
			if !ok {
				return nil, operandError("case condition")
			}
			if !value {
				PC = op - 1
			}

		case isa.Jump:
			PC = op - 1

		case isa.Array:
			RES := SP - op
			result := append([]any{}, MEM[RES:SP]...)
			MEM[RES] = result
			SP = RES + 1

		case isa.Object:
			keys := p.constants[op].([]string)
			RES := SP - len(keys)
			result := make(map[string]any, len(keys))
			for i, key := range keys {
				if value := MEM[RES+i]; !IsMissing(value) {
					result[key] = value
				}
			}
			MEM[RES] = result
			SP = RES + 1

		case isa.Push:
			SP--
			value := MEM[SP]
			MEM[SP-1] = append(MEM[SP-1].([]any), value)

		case isa.Put:
			SP--
			value := MEM[SP]
			if !IsMissing(value) {
				key := p.constants[op].(string)
				MEM[SP-1].(map[string]any)[key] = value
			}

		case isa.Concat:
			SP--
			if nullish(MEM[SP]) {
				continue
			}
			source, ok := MEM[SP].([]any)
			if !ok {
				return nil, operandError("spread")
			}
			MEM[SP-1] = append(MEM[SP-1].([]any), source...)

		case isa.Merge:
			SP--
			if nullish(MEM[SP]) {
				continue
			}
			source, ok := MEM[SP].(map[string]any)
			if !ok {
				return nil, operandError("spread")
			}
			target := MEM[SP-1].(map[string]any)
			for key, value := range source {
				if !IsMissing(value) {
					target[key] = value
				}
			}

		case isa.Call:
			spec := p.constants[op].(callSpec)
			RES := SP - spec.arity
			if cap(arguments) < spec.arity {
				arguments = make([]any, spec.arity)
			}
			args := arguments[:spec.arity]
			for i := range args {
				value := MEM[RES+i]
				args[i] = value
			}
			result, err := spec.call(args...)
			if err != nil {
				return nil, err
			}
			MEM[RES] = result
			SP = RES + 1

		case isa.Iterate:
			spec := p.iterations[op]
			keys, values, err := collectionEntries(
				MEM[SP-1], spec.keyed,
			)
			if err != nil {
				return nil, err
			}
			MEM[SP-1] = &iterator{
				keys:   keys,
				values: values,
				result: []any{},
			}

		case isa.Fetch:
			spec := p.iterations[op]
			it := MEM[SP-1].(*iterator)
			if len(it.values) == 0 {
				PC = spec.end - 1
				continue
			}
			MEM[LP+spec.valueSlot] = it.values[0]
			it.values = it.values[1:]
			if spec.keyed {
				MEM[LP+spec.keySlot] = it.keys[0]
				it.keys = it.keys[1:]
			}

		case isa.Append:
			SP--
			value := MEM[SP]
			it := MEM[SP-1].(*iterator)
			it.result = append(it.result, value)

		case isa.Result:
			it := MEM[SP-1].(*iterator)
			MEM[SP-1] = it.result

		case isa.Pop:
			SP--
		}
	}
	return MEM[0], nil
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
