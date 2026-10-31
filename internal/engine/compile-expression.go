package engine

import (
	"fmt"

	"github.com/kode4food/argex/internal/isa"
)

var binaryOpcodes = map[string]isa.Opcode{
	opAdd:      isa.Add,
	opSub:      isa.Sub,
	opMul:      isa.Mul,
	opDiv:      isa.Div,
	opMod:      isa.Mod,
	opEq:       isa.Eq,
	opNeq:      isa.Neq,
	opIn:       isa.In,
	opLt:       isa.Lt,
	opLte:      isa.Lte,
	opGt:       isa.Gt,
	opGte:      isa.Gte,
	opAnd:      isa.And,
	opOr:       isa.Or,
	opCoalesce: isa.Coalesce,
}

func (c *compiler) operation(n *node, depth int) error {
	for _, child := range n.children {
		if err := c.compile(child, depth+1); err != nil {
			return err
		}
	}
	switch n.kind {
	case nodeUnary:
		op := isa.Pos
		switch n.text {
		case opNot:
			op = isa.Not

		case opSub:
			op = isa.Neg
		}
		c.emit(op, 0)

	case nodeAccess:
		op := isa.Access
		if n.optional {
			op = isa.OptionalAccess
		}
		c.emit(op, c.constant(n.text))

	default:
		return fmt.Errorf("%w: unknown expression", ErrSyntax)
	}
	return nil
}

// call resolves globals statically to avoid runtime lookups on direct calls
func (c *compiler) call(n *node, depth int) error {
	callee := n.children[0]
	args := n.children[1:]
	if fn, ok := callee.value.(*Function); ok {
		return c.callStatic(fn, args, depth)
	}
	if _, ok := c.local(callee.text); callee.kind == nodeName && !ok {
		return fmt.Errorf("%w: %s", ErrFunction, callee.text)
	}
	if err := c.compile(callee, depth+1); err != nil {
		return err
	}
	return c.callDynamic(args, depth)
}

func (c *compiler) callStatic(fn *Function, args []*node, depth int) error {
	spreads := countSpreads(args)
	fixed := len(args) - spreads
	if spreads == 0 && !fn.accepts(fixed) || !fn.Variadic && fixed > fn.Arity {
		return ErrFunction
	}
	if spreads > 0 {
		c.emit(isa.Const, c.constant(fn))
		return c.callWith(args, depth)
	}
	for _, child := range args {
		if err := c.compile(child, depth+1); err != nil {
			return err
		}
	}
	spec := callSpec{call: fn.Call, arity: fixed}
	c.emit(isa.Call, c.constant(spec))
	return nil
}

func (c *compiler) callDynamic(args []*node, depth int) error {
	if countSpreads(args) > 0 {
		return c.callWith(args, depth)
	}
	for _, arg := range args {
		if err := c.compile(arg, depth+1); err != nil {
			return err
		}
	}
	c.emit(isa.CallDynamic, isa.Operand(len(args)))
	return nil
}

// callWith shares array construction's nullish handling for argument spreads
func (c *compiler) callWith(args []*node, depth int) error {
	if err := c.array(args, depth); err != nil {
		return err
	}
	c.emit(isa.CallWith, 0)
	return nil
}

func (c *compiler) binary(n *node, depth int) error {
	if err := c.compile(n.children[0], depth+1); err != nil {
		return err
	}
	op := binaryOpcodes[n.text]
	if op != isa.And && op != isa.Or && op != isa.Coalesce {
		if err := c.compile(n.children[1], depth+1); err != nil {
			return err
		}
		c.emit(op, 0)
		return nil
	}
	end := c.newLabel()
	c.emit(op, end)
	if err := c.compile(n.children[1], depth+1); err != nil {
		return err
	}
	if op != isa.Coalesce {
		c.emit(isa.Boolean, 0)
	}
	c.emit(isa.Label, end)
	return nil
}

func (c *compiler) index(n *node, depth int) error {
	if err := c.compile(n.children[0], depth+1); err != nil {
		return err
	}
	if !n.optional {
		if err := c.compile(n.children[1], depth+1); err != nil {
			return err
		}
		c.emit(isa.Index, 0)
		return nil
	}
	end := c.newLabel()
	c.emit(isa.Optional, end)
	if err := c.compile(n.children[1], depth+1); err != nil {
		return err
	}
	c.emit(isa.OptionalIndex, 0)
	c.emit(isa.Label, end)
	return nil
}

func (c *compiler) branch(n *node, depth int) error {
	end := c.newLabel()
	for i := 0; i < len(n.children)-1; i += 2 {
		if err := c.compile(n.children[i], depth+1); err != nil {
			return err
		}
		next := c.newLabel()
		c.emit(isa.CondJump, next)
		if err := c.compile(n.children[i+1], depth+1); err != nil {
			return err
		}
		c.emit(isa.Jump, end)
		c.emit(isa.Label, next)
	}
	if err := c.compile(n.children[len(n.children)-1], depth+1); err != nil {
		return err
	}
	c.emit(isa.Label, end)
	return nil
}

// let stores its value in a fresh slot that only its body can see
func (c *compiler) let(n *node, depth int) error {
	if err := c.compile(n.children[0], depth+1); err != nil {
		return err
	}
	mark := c.nextSlot
	slot := c.allocSlot()
	c.emit(isa.Store, isa.Operand(slot))
	bound := []binding{{name: n.valueBinding, slot: slot}}
	err := c.scoped(bound, n.children[1], depth)
	c.nextSlot = mark
	return err
}

func countSpreads(children []*node) int {
	count := 0
	for _, child := range children {
		if child.kind == nodeSpread {
			count++
		}
	}
	return count
}
