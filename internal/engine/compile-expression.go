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
		c.emit(op, 0, 0)

	case nodeAccess:
		op := isa.Access
		if n.optional {
			op = isa.OptionalAccess
		}
		c.emit(op, c.constant(n.text), 0)

	default:
		return fmt.Errorf("%w: unknown expression", ErrSyntax)
	}
	return nil
}

func (c *compiler) call(n *node, depth int) error {
	fn, ok := builtins[n.text]
	if !ok {
		return fmt.Errorf("%w: %s", ErrFunction, n.text)
	}
	arity := len(n.children)
	if arity < fn.arity || !fn.variadic && arity != fn.arity {
		return fmt.Errorf("%w: %s", ErrFunction, n.text)
	}
	for _, child := range n.children {
		if err := c.compile(child, depth+1); err != nil {
			return err
		}
	}
	spec := callSpec{call: fn.call, arity: arity}
	c.emit(isa.Call, c.constant(spec), stackChange(1-arity))
	return nil
}

func (c *compiler) binary(n *node, depth int) error {
	if err := c.compile(n.children[0], depth+1); err != nil {
		return err
	}
	op := binaryOpcodes[n.text]
	jump := -1
	if op == isa.And || op == isa.Or || op == isa.Coalesce {
		jump = c.emit(op, 0, -1)
	}
	if err := c.compile(n.children[1], depth+1); err != nil {
		return err
	}
	if jump < 0 {
		c.emit(op, 0, -1)
		return nil
	}
	if op != isa.Coalesce {
		c.emit(isa.Boolean, c.constant(n.text), 0)
	}
	c.patch(jump)
	return nil
}

func (c *compiler) index(n *node, depth int) error {
	if err := c.compile(n.children[0], depth+1); err != nil {
		return err
	}
	jump := -1
	if n.optional {
		jump = c.emit(isa.Optional, 0, 0)
	}
	if err := c.compile(n.children[1], depth+1); err != nil {
		return err
	}
	op := isa.Index
	if n.optional {
		op = isa.OptionalIndex
	}
	c.emit(op, 0, -1)
	if jump >= 0 {
		c.patch(jump)
	}
	return nil
}

func (c *compiler) branch(n *node, depth int) error {
	base := c.height
	var exits []int
	for i := 0; i < len(n.children)-1; i += 2 {
		if err := c.compile(n.children[i], depth+1); err != nil {
			return err
		}
		next := c.emit(isa.CondJump, 0, -1)
		if err := c.compile(n.children[i+1], depth+1); err != nil {
			return err
		}
		exits = append(exits, c.emit(isa.Jump, 0, 0))
		c.height = base
		c.patch(next)
	}
	if err := c.compile(n.children[len(n.children)-1], depth+1); err != nil {
		return err
	}
	for _, exit := range exits {
		c.patch(exit)
	}
	return nil
}

// let stores its value in a fresh slot that only its body can see
func (c *compiler) let(n *node, depth int) error {
	if err := c.compile(n.children[0], depth+1); err != nil {
		return err
	}
	mark := c.nextSlot
	slot := c.allocSlot()
	c.emit(isa.Store, isa.Operand(slot), -1)
	bound := []binding{{name: n.valueBinding, slot: slot}}
	err := c.scoped(bound, n.children[1], depth)
	c.nextSlot = mark
	return err
}
