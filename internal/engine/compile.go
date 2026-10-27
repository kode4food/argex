package engine

import (
	"fmt"
	"slices"

	"github.com/kode4food/argex/internal/isa"
)

type (
	compiler struct {
		program  *program
		scope    []binding
		nextSlot int
		height   int
	}

	binding struct {
		name string
		slot int
	}

	stackChange int
)

// Compile prepares an expression for repeated evaluation
func Compile(expr *Expression) (func(map[string]any) (any, error), error) {
	if expr == nil || expr.root == nil {
		return nil, fmt.Errorf("%w: empty expression", ErrSyntax)
	}
	c := &compiler{program: &program{}}
	if err := c.compile(expr.root, 0); err != nil {
		return nil, err
	}
	return c.program.evaluate, nil
}

func (c *compiler) compile(n *node, depth int) error {
	if depth > maxDepth {
		return ErrLimit
	}
	var err error
	switch n.kind {
	case "literal":
		c.emit(isa.Const, c.constant(n.value), 1)

	case "name":
		c.name(n.text)

	case "binary":
		err = c.binary(n, depth)

	case "case":
		err = c.branch(n, depth)

	case "map", "filter", "any", "all":
		err = c.collection(n, depth)

	case "index":
		err = c.index(n, depth)

	case "call":
		err = c.call(n, depth)

	case "array":
		err = c.array(n, depth)

	case "object":
		err = c.object(n, depth)

	default:
		err = c.operation(n, depth)
	}
	return err
}

func (c *compiler) name(name string) {
	for _, b := range slices.Backward(c.scope) {
		if b.name == name {
			c.emit(isa.Load, isa.Operand(b.slot), 1)
			return
		}
	}
	c.emit(isa.Global, c.constant(name), 1)
}

func (c *compiler) emit(
	op isa.Opcode, operand isa.Operand, change stackChange,
) int {
	index := len(c.program.code)
	c.program.code = append(c.program.code, op.New(operand))
	c.height += int(change)
	c.program.stackSize = max(c.program.stackSize, c.height)
	return index
}

func (c *compiler) constant(value any) isa.Operand {
	index := isa.Operand(len(c.program.constants))
	c.program.constants = append(c.program.constants, value)
	return index
}

func (c *compiler) patch(index int) {
	inst := c.program.code[index]
	c.program.code[index] = inst.Opcode().New(isa.Operand(len(c.program.code)))
}
