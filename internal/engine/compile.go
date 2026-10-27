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
		switch child.kind {
		case "spread":
			if err := c.compile(child.children[0], depth+1); err != nil {
				return err
			}
			c.emit(isa.Merge, 0, -1)
			continue

		case "field":
			c.emit(isa.Const, c.constant(child.text), 1)
		}
		for _, part := range child.children {
			if err := c.compile(part, depth+1); err != nil {
				return err
			}
		}
		c.emit(isa.Put, 0, -2)
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
	for _, b := range slices.Backward(c.scope) {
		if b.name == name {
			c.emit(isa.Load, isa.Operand(b.slot), 1)
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

// collection compiles a pipeline: the outermost stage plus every map/filter
// stage chained through its receiver run as one Iterate/Fetch loop
func (c *compiler) collection(n *node, depth int) error {
	stages := []*node{n}
	for chainable(stages[len(stages)-1].children[0]) {
		stages = append(stages, stages[len(stages)-1].children[0])
	}
	slices.Reverse(stages)
	return c.pipeline(stages, depth)
}

// pipeline passes each stage's output to the next through a slot; later stages
// only receive arrays, so they count their own index
func (c *compiler) pipeline(stages []*node, depth int) error {
	if err := c.compile(stages[0].children[0], depth+1); err != nil {
		return err
	}
	mark := c.nextSlot
	scopes := make([][]binding, len(stages))
	for i, stage := range stages {
		scopes[i] = []binding{{name: stage.valueBinding, slot: c.allocSlot()}}
		if stage.keyBinding != "" {
			key := binding{name: stage.keyBinding, slot: c.allocSlot()}
			scopes[i] = append(scopes[i], key)
		}
	}

	spec := iterationSpec{valueSlot: scopes[0][0].slot}
	if len(scopes[0]) > 1 {
		spec.keySlot = scopes[0][1].slot
		spec.keyed = true
	}
	index := len(c.program.iterations)
	c.program.iterations = append(c.program.iterations, spec)
	c.emit(isa.Iterate, isa.Operand(index), 0)
	for _, names := range scopes[1:] {
		if len(names) > 1 {
			c.emit(isa.Zero, isa.Operand(names[1].slot), 0)
		}
	}

	loopTop := len(c.program.code)
	c.emit(isa.Fetch, isa.Operand(index), 0)
	for i, stage := range stages {
		names := scopes[i]
		if err := c.scoped(names, stage.children[1], depth); err != nil {
			return err
		}
		if i > 0 && len(names) > 1 {
			c.emit(isa.Inc, isa.Operand(names[1].slot), 0)
		}
		if i == len(stages)-1 {
			break
		}
		if stage.kind == "filter" {
			c.emit(isa.CondJump, isa.Operand(loopTop), -1)
			c.emit(isa.Load, isa.Operand(names[0].slot), 1)
		}
		c.emit(isa.Store, isa.Operand(scopes[i+1][0].slot), -1)
	}

	switch stages[len(stages)-1].kind {
	case "map":
		c.emit(isa.Append, 0, -1)
		c.emit(isa.Jump, isa.Operand(loopTop), 0)
		c.program.iterations[index].end = len(c.program.code)
		c.emit(isa.Result, 0, 0)

	case "filter":
		skip := c.emit(isa.CondJump, 0, -1)
		c.emit(isa.Load, isa.Operand(scopes[len(scopes)-1][0].slot), 1)
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
	c.nextSlot = mark
	return nil
}

// scoped compiles body seeing the enclosing scope plus names, and nothing bound
// by sibling stages
func (c *compiler) scoped(names []binding, body *node, depth int) error {
	outer := c.scope
	c.scope = append(outer[:len(outer):len(outer)], names...)
	err := c.compile(body, depth+1)
	c.scope = outer
	return err
}

func (c *compiler) allocSlot() int {
	slot := c.nextSlot
	c.nextSlot++
	c.program.localCount = max(c.program.localCount, c.nextSlot)
	return slot
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

// chainable reports whether n can feed a later stage of the same pipeline;
// any and all produce booleans, so they can only end one
func chainable(n *node) bool {
	return n.kind == "map" || n.kind == "filter"
}
