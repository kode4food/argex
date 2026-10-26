package engine

import (
	"fmt"
	"slices"

	"github.com/kode4food/argex/internal/isa"
)

type (
	compiler struct {
		program  *program
		bindings []string
		height   int
	}

	stackChange int
)

func (c *compiler) compile(n *node, depth int) error {
	if depth > maxDepth {
		return ErrLimit
	}
	switch n.kind {
	case "literal":
		c.emit(isa.Const, c.constant(n.value), 1)

	case "name":
		c.name(n.text)

	case "binary":
		return c.binary(n, depth)

	case "case":
		return c.branch(n, depth)

	case "map", "filter", "any", "all":
		return c.collection(n, depth)

	case "index":
		return c.index(n, depth)

	case "call":
		return c.call(n, depth)

	case "array":
		return c.array(n, depth)

	case "object":
		return c.object(n, depth)

	default:
		return c.operation(n, depth)
	}
	return nil
}

func (c *compiler) operation(n *node, depth int) error {
	for _, child := range n.children {
		if err := c.compile(child, depth+1); err != nil {
			return err
		}
	}
	switch n.kind {
	case "unary":
		op := isa.Pos
		switch n.text {
		case "!":
			op = isa.Not

		case "-":
			op = isa.Neg
		}
		c.emit(op, 0, 0)

	case "access":
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

// array builds an array literal. A leading run of plain elements uses one
// Array op; each later element folds in with Push, each spread with Concat
func (c *compiler) array(n *node, depth int) error {
	fixed := 0
	for fixed < len(n.children) && n.children[fixed].kind != "spread" {
		if err := c.compile(n.children[fixed], depth+1); err != nil {
			return err
		}
		fixed++
	}
	c.emit(isa.Array, isa.Operand(fixed), stackChange(1-fixed))
	for _, child := range n.children[fixed:] {
		if child.kind == "spread" {
			if err := c.compile(child.children[0], depth+1); err != nil {
				return err
			}
			c.emit(isa.Concat, 0, -1)
			continue
		}
		if err := c.compile(child, depth+1); err != nil {
			return err
		}
		c.emit(isa.Push, 0, -1)
	}
	return nil
}

// object builds an object literal. A leading run of plain fields uses one
// Object op; each later field folds in with Put, each spread with Merge
func (c *compiler) object(n *node, depth int) error {
	var keys []string
	fixed := 0
	for fixed < len(n.children) && n.children[fixed].kind == "field" {
		field := n.children[fixed]
		keys = append(keys, field.text)
		if err := c.compile(field.children[0], depth+1); err != nil {
			return err
		}
		fixed++
	}
	c.emit(isa.Object, c.constant(keys), stackChange(1-len(keys)))
	for _, child := range n.children[fixed:] {
		if child.kind == "spread" {
			if err := c.compile(child.children[0], depth+1); err != nil {
				return err
			}
			c.emit(isa.Merge, 0, -1)
			continue
		}
		if err := c.compile(child.children[0], depth+1); err != nil {
			return err
		}
		c.emit(isa.Put, c.constant(child.text), -1)
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

func (c *compiler) name(name string) {
	for i, v := range slices.Backward(c.bindings) {
		if v == name {
			c.emit(isa.Load, isa.Operand(i), 1)
			return
		}
	}
	c.emit(isa.Global, c.constant(name), 1)
}

func (c *compiler) binary(n *node, depth int) error {
	if err := c.compile(n.children[0], depth+1); err != nil {
		return err
	}
	op := binaryOpcode(n.text)
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

// collection compiles map/filter/any/all into an Iterate/Fetch loop over an
// iterator value, with generic CondJump/Jump driving the per-kind outcome
func (c *compiler) collection(n *node, depth int) error {
	if err := c.compile(n.children[0], depth+1); err != nil {
		return err
	}
	valueSlot := len(c.bindings)
	c.bindings = append(c.bindings, n.valueBinding)
	keyed := n.keyBinding != ""
	keySlot := 0
	if keyed {
		keySlot = len(c.bindings)
		c.bindings = append(c.bindings, n.keyBinding)
	}
	c.program.localCount = max(c.program.localCount, len(c.bindings))

	index := len(c.program.iterations)
	c.program.iterations = append(c.program.iterations, iterationSpec{
		valueSlot: valueSlot,
		keySlot:   keySlot,
		keyed:     keyed,
	})
	c.emit(isa.Iterate, isa.Operand(index), 0)

	loopTop := len(c.program.code)
	c.emit(isa.Fetch, isa.Operand(index), 0)
	if err := c.compile(n.children[1], depth+1); err != nil {
		return err
	}
	switch n.kind {
	case "map":
		c.emit(isa.Append, 0, -1)
		c.emit(isa.Jump, isa.Operand(loopTop), 0)
		c.program.iterations[index].end = len(c.program.code)
		c.emit(isa.Result, 0, 0)

	case "filter":
		skip := c.emit(isa.CondJump, 0, -1)
		c.emit(isa.Load, isa.Operand(valueSlot), 1)
		c.emit(isa.Append, 0, -1)
		c.patch(skip)
		c.emit(isa.Jump, isa.Operand(loopTop), 0)
		c.program.iterations[index].end = len(c.program.code)
		c.emit(isa.Result, 0, 0)

	case "any":
		c.emit(isa.CondJump, isa.Operand(loopTop), -1)
		c.emit(isa.Pop, 0, -1)
		c.emit(isa.Const, c.constant(true), 1)
		done := c.emit(isa.Jump, 0, 0)
		c.program.iterations[index].end = len(c.program.code)
		c.emit(isa.Pop, 0, -1)
		c.emit(isa.Const, c.constant(false), 1)
		c.patch(done)

	case "all":
		fail := c.emit(isa.CondJump, 0, -1)
		c.emit(isa.Jump, isa.Operand(loopTop), 0)
		c.patch(fail)
		c.emit(isa.Pop, 0, -1)
		c.emit(isa.Const, c.constant(false), 1)
		done := c.emit(isa.Jump, 0, 0)
		c.program.iterations[index].end = len(c.program.code)
		c.emit(isa.Pop, 0, -1)
		c.emit(isa.Const, c.constant(true), 1)
		c.patch(done)
	}
	c.bindings = c.bindings[:valueSlot]
	return nil
}

func (c *compiler) emit(
	op isa.Opcode, operand isa.Operand, change stackChange,
) int {
	if op >= isa.Pos && op <= isa.Gte {
		c.program.numeric = true
	}
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

func binaryOpcode(op string) isa.Opcode {
	switch op {
	case "+":
		return isa.Add

	case "-":
		return isa.Sub

	case "*":
		return isa.Mul

	case "/":
		return isa.Div

	case "%":
		return isa.Mod

	case "==":
		return isa.Eq

	case "!=":
		return isa.Neq

	case "in":
		return isa.In

	case "<":
		return isa.Lt

	case "<=":
		return isa.Lte

	case ">":
		return isa.Gt

	case ">=":
		return isa.Gte

	case "&&":
		return isa.And

	case "||":
		return isa.Or

	default:
		return isa.Coalesce
	}
}
