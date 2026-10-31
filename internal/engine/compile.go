package engine

import (
	"fmt"
	"slices"

	"github.com/kode4food/argex/internal/isa"
)

type (
	compiler struct {
		program   *Program
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

const (
	// rootSlot holds this so it compiles to an ordinary local Load
	rootSlot = 0

	// caseSubject is a subject case's hidden binding, outside identifier syntax
	caseSubject = "case subject"
)

// Compile resolves an expression's names, checks its types, and
// prepares it for repeated evaluation
func (e *Env) Compile(expr *Expression) (*Program, error) {
	if expr == nil || expr.root == nil {
		return nil, fmt.Errorf("%w: empty expression", ErrSyntax)
	}
	root, err := expr.resolve(e)
	if err != nil {
		return nil, err
	}
	c := &compiler{
		program: &Program{
			localCount: rootSlot + 1,
			resultType: root.inferred,
		},
		nextSlot: rootSlot + 1,
	}
	if err := c.compile(root, 0); err != nil {
		return nil, err
	}
	c.finalize()
	return c.program, nil
}

func (c *compiler) compile(n *node, depth int) error {
	if depth > maxDepth {
		return ErrLimit
	}
	switch n.kind {
	case nodeLiteral:
		c.emit(isa.Const, c.constant(n.value))
		return nil

	case nodeName:
		c.name(n.text)
		return nil

	case nodeRoot:
		c.emit(isa.Load, rootSlot)
		return nil

	case nodeBinary:
		return c.binary(n, depth)

	case nodeCase:
		return c.branch(n, depth)

	case nodeLet:
		return c.let(n, depth)

	case nodeMap, nodeFilter, nodeAny, nodeAll:
		return c.collection(n, depth)

	case nodeIndex:
		return c.index(n, depth)

	case nodeCall:
		return c.call(n, depth)

	case nodeArray:
		return c.array(n.children, depth)

	case nodeObject:
		return c.object(n, depth)

	default:
		return c.operation(n, depth)
	}
}

func (c *compiler) name(name string) {
	if slot, ok := c.local(name); ok {
		c.emit(isa.Load, isa.Operand(slot))
		return
	}
	if c.looping {
		load := c.emit(isa.Load, c.loopSlot(name))
		c.loopRefs = append(c.loopRefs, load)
		return
	}
	c.emit(isa.Global, c.constant(name))
}

func (c *compiler) local(name string) (int, bool) {
	for _, b := range slices.Backward(c.scope) {
		if b.name == name {
			return b.slot, true
		}
	}
	return 0, false
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

// lowerNode rewrites source forms on the compiler's copy before type checking
func lowerNode(n *node) {
	for _, child := range n.children {
		lowerNode(child)
	}
	switch n.kind {
	case nodeMatch:
		n.kind = nodeLet
		n.valueBinding = caseSubject
		arms := n.children[1]
		for i := 0; i < len(arms.children)-1; i += 2 {
			name := &node{kind: nodeName, text: caseSubject}
			arms.children[i] = &node{
				kind:     nodeBinary,
				text:     opEq,
				children: []*node{name, arms.children[i]},
			}
		}

	case nodeComputed:
		key := n.children[0]
		if name, ok := key.value.(string); key.kind == nodeLiteral && ok {
			n.kind = nodeField
			n.text = name
			n.children = n.children[1:]
		}
	}
}
