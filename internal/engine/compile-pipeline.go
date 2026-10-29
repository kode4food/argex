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

// pipeline passes each stage's output to the next through a slot. Later stages
// count their own index and see earlier stages' bindings for the same element
func (c *compiler) pipeline(stages []*node, depth int) error {
	if err := c.compile(stages[0].children[0], depth+1); err != nil {
		return err
	}
	looping := c.looping
	c.looping = true
	mark := c.nextSlot
	scopes := make([][]binding, len(stages))
	visible := make([][]binding, len(stages))
	counters := make([]int, len(stages))
	for i, stage := range stages {
		scopes[i] = []binding{{name: stage.valueBinding, slot: c.allocSlot()}}
		if stage.keyBinding != "" {
			key := binding{name: stage.keyBinding, slot: c.allocSlot()}
			scopes[i] = append(scopes[i], key)
			counters[i] = key.slot
		}
		if i > 0 && i < len(stages)-1 && stage.keyBinding != "" {
			counters[i] = c.allocSlot()
		}
		visible[i] = scopes[i]
		if stage.chained {
			visible[i] = slices.Concat(visible[i-1], scopes[i])
		}
	}

	// the loop is entered at its Fetch, which sits at the bottom and jumps back
	// to the top while elements remain. Skipped elements jump to it too
	top := c.newLabel()
	next := c.newLabel()
	entry := c.newLabel()
	done := c.newLabel()
	spec := iterationSpec{valueSlot: scopes[0][0].slot, top: int(top)}
	if len(scopes[0]) > 1 {
		spec.keySlot = scopes[0][1].slot
		spec.keyed = true
	}
	index := isa.Operand(len(c.program.iterations))
	c.program.iterations = append(c.program.iterations, spec)
	c.emit(isa.Iterate, index)
	for i, names := range scopes[1:] {
		if len(names) > 1 {
			c.emit(isa.Zero, isa.Operand(counters[i+1]))
		}
	}
	c.emit(isa.Jump, entry)
	c.emit(isa.Label, top)
	for i, stage := range stages {
		names := scopes[i]
		// later stages read the key, so it can't be the counter Inc advances
		if i > 0 && len(names) > 1 && counters[i] != names[1].slot {
			c.emit(isa.Load, isa.Operand(counters[i]))
			c.emit(isa.Store, isa.Operand(names[1].slot))
		}
		if err := c.scoped(visible[i], stage.children[1], depth); err != nil {
			return err
		}
		if i > 0 && len(names) > 1 {
			c.emit(isa.Inc, isa.Operand(counters[i]))
		}
		if i == len(stages)-1 {
			break
		}
		if stage.kind == nodeFilter {
			c.emit(isa.CondJump, next)
			c.emit(isa.Load, isa.Operand(names[0].slot))
		}
		c.emit(isa.Store, isa.Operand(scopes[i+1][0].slot))
	}

	final := stages[len(stages)-1].kind
	switch final {
	case nodeMap:
		c.emit(isa.Append, 0)

	case nodeFilter:
		c.emit(isa.CondJump, next)
		c.emit(isa.Load, isa.Operand(scopes[len(scopes)-1][0].slot))
		c.emit(isa.Append, 0)

	case nodeAny:
		c.emit(isa.CondJump, next)
		c.emit(isa.Pop, 0)
		c.emit(isa.Const, c.constant(true))
		c.emit(isa.Jump, done)

	case nodeAll:
		c.emit(isa.JumpIf, next)
		c.emit(isa.Pop, 0)
		c.emit(isa.Const, c.constant(false))
		c.emit(isa.Jump, done)
	}

	preload := !looping && len(c.hoisted) > 0
	if !preload {
		c.emit(isa.Label, entry)
	}
	c.emit(isa.Label, next)
	c.emit(isa.Fetch, index)
	switch final {
	case nodeAny:
		c.emit(isa.Pop, 0)
		c.emit(isa.Const, c.constant(false))

	case nodeAll:
		c.emit(isa.Pop, 0)
		c.emit(isa.Const, c.constant(true))

	default:
		c.emit(isa.Result, 0)
	}
	if preload {
		c.preload(entry, next, done)
	}
	c.emit(isa.Label, done)
	c.nextSlot = mark
	c.looping = looping
	return nil
}

// preload loads the names an outermost loop read into their slots, in a block
// after the loop that its entry jump passes through on the way to Fetch
func (c *compiler) preload(entry, next, done isa.Operand) {
	c.emit(isa.Jump, done)
	c.emit(isa.Label, entry)
	for i, name := range c.hoisted {
		c.emit(isa.Global, c.constant(name))
		slot := isa.Operand(c.loopBase + i)
		c.loopRefs = append(c.loopRefs, c.emit(isa.Store, slot))
	}
	c.loopBase += len(c.hoisted)
	c.hoisted = nil
	c.emit(isa.Jump, next)
}

// scoped compiles body seeing the enclosing scope plus names, and nothing bound
// outside its pipeline
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

// chainable reports whether n can feed a later stage of the same pipeline. Any
// and all produce booleans, so they can only end one
func chainable(n *node) bool {
	return n.kind == nodeMap || n.kind == nodeFilter
}
