package engine

import (
	"slices"

	"github.com/kode4food/argex/internal/isa"
)

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

// chainable reports whether n can feed a later stage of the same pipeline;
// any and all produce booleans, so they can only end one
func chainable(n *node) bool {
	return n.kind == "map" || n.kind == "filter"
}
