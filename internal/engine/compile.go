package engine

import (
	"fmt"
	"slices"

	"github.com/kode4food/argex/internal/isa"
)

type (
	compiler struct {
		program   *program
		scope     []binding
		loopRefs  []int
		hoisted   []string
		loopBase  int
		nextSlot  int
		nextLabel isa.Operand
		looping   bool
	}

	binding struct {
		name string
		slot int
	}
)

// rootSlot holds the program's this, so it compiles to an ordinary local Load
const rootSlot = 0

// Compile prepares an expression for repeated evaluation
func Compile(expr *Expression) (func(any) (any, error), error) {
	if expr == nil || expr.root == nil {
		return nil, fmt.Errorf("%w: empty expression", ErrSyntax)
	}
	c := &compiler{
		program:  &program{localCount: rootSlot + 1},
		nextSlot: rootSlot + 1,
	}
	if err := c.compile(expr.root, 0); err != nil {
		return nil, err
	}
	c.finalize()
	return c.program.evaluate, nil
}

func (c *compiler) compile(n *node, depth int) error {
	if depth > maxDepth {
		return ErrLimit
	}
	var err error
	switch n.kind {
	case nodeLiteral:
		c.emit(isa.Const, c.constant(n.value))

	case nodeName:
		c.name(n.text)

	case nodeRoot:
		c.emit(isa.Load, rootSlot)

	case nodeBinary:
		err = c.binary(n, depth)

	case nodeCase:
		err = c.branch(n, depth)

	case nodeLet:
		err = c.let(n, depth)

	case nodeMap, nodeFilter, nodeAny, nodeAll:
		err = c.collection(n, depth)

	case nodeIndex:
		err = c.index(n, depth)

	case nodeCall:
		err = c.call(n, depth)

	case nodeArray:
		err = c.array(n, depth)

	case nodeObject:
		err = c.object(n, depth)

	default:
		err = c.operation(n, depth)
	}
	return err
}

func (c *compiler) name(name string) {
	for _, b := range slices.Backward(c.scope) {
		if b.name == name {
			c.emit(isa.Load, isa.Operand(b.slot))
			return
		}
	}
	if c.looping {
		load := c.emit(isa.Load, c.loopSlot(name))
		c.loopRefs = append(c.loopRefs, load)
		return
	}
	c.emit(isa.Global, c.constant(name))
}

func (c *compiler) emit(op isa.Opcode, operand isa.Operand) int {
	index := len(c.program.code)
	c.program.code = append(c.program.code, op.New(operand))
	return index
}

func (c *compiler) constant(value any) isa.Operand {
	index := isa.Operand(len(c.program.constants))
	c.program.constants = append(c.program.constants, value)
	return index
}

// newLabel allocates a label for a jump. Emitting a Label with it marks the
// destination
func (c *compiler) newLabel() isa.Operand {
	label := c.nextLabel
	c.nextLabel++
	return label
}

// loopSlot returns the slot holding the input variable name inside the
// outermost loop being compiled, which loads it once before it starts
func (c *compiler) loopSlot(name string) isa.Operand {
	i := slices.Index(c.hoisted, name)
	if i < 0 {
		i = len(c.hoisted)
		c.hoisted = append(c.hoisted, name)
	}
	return isa.Operand(c.loopBase + i)
}
