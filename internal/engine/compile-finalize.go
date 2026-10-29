package engine

import "github.com/kode4food/argex/internal/isa"

// labelSite is where a label landed and the stack depth its jumps carry there
type labelSite struct {
	offset  isa.Operand
	depth   int
	reached bool
}

// fusedJumps replaces a comparison and the conditional jump right after it.
// Jumping unless a comparison holds jumps if its inverse does
var fusedJumps = map[[2]isa.Opcode]isa.Opcode{
	{isa.Lt, isa.JumpIf}:    isa.JumpIfLt,
	{isa.Lte, isa.JumpIf}:   isa.JumpIfLte,
	{isa.Gt, isa.JumpIf}:    isa.JumpIfGt,
	{isa.Gte, isa.JumpIf}:   isa.JumpIfGte,
	{isa.Lt, isa.CondJump}:  isa.JumpIfGte,
	{isa.Lte, isa.CondJump}: isa.JumpIfGt,
	{isa.Gt, isa.CondJump}:  isa.JumpIfLte,
	{isa.Gte, isa.CondJump}: isa.JumpIfLt,
}

// finalize makes the compiled code runnable, like Ale's Encoded.Runnable. It
// places the loop slots, fuses jumps, sizes the stack, and resolves labels
func (c *compiler) finalize() {
	p := c.program
	for _, index := range c.loopRefs {
		inst := p.code[index]
		slot := inst.Operand() + isa.Operand(p.localCount)
		p.code[index] = inst.Opcode().New(slot)
	}
	p.localCount += c.loopBase
	p.code = fuseJumps(p.code)
	labels := make([]labelSite, c.nextLabel)
	p.stackSize = p.layout(labels)
	p.resolveLabels(labels)
}

// layout finds where each label lands and the deepest the stack gets. Jumps
// are structured, so a label sits at the depth its jumps carry, or the current
func (p *program) layout(labels []labelSite) int {
	depth := 0
	deepest := 0
	count := 0
	for _, inst := range p.code {
		op := inst.Opcode()
		if op == isa.Label {
			site := &labels[inst.Operand()]
			site.offset = isa.Operand(count)
			if site.reached {
				depth = site.depth
			}
			continue
		}
		count++
		effect := isa.Effects[op]
		pops := p.pops(inst, effect)
		if effect.Operand == isa.Labels {
			site := &labels[inst.Operand()]
			site.depth = depth - pops
			if effect.Keep {
				site.depth = depth
			}
			site.reached = true
		}
		depth += effect.Push - pops
		deepest = max(deepest, depth)
	}
	return deepest
}

func (p *program) pops(inst isa.Instruction, effect isa.Effect) int {
	if !effect.DPop {
		return effect.Pop
	}
	if effect.Operand == isa.Count {
		return int(inst.Operand())
	}
	switch constant := p.constants[inst.Operand()].(type) {
	case []string:
		return len(constant)

	case callSpec:
		return constant.arity

	default:
		return 0
	}
}

// resolveLabels drops the labels and points every jump, and every loop's Fetch,
// at the instruction its label marks
func (p *program) resolveLabels(labels []labelSite) {
	code := p.code[:0]
	for _, inst := range p.code {
		op := inst.Opcode()
		if op == isa.Label {
			continue
		}
		if isa.Effects[op].Operand == isa.Labels {
			inst = op.New(labels[inst.Operand()].offset)
		}
		code = append(code, inst)
	}
	for i := range p.iterations {
		p.iterations[i].top = int(labels[p.iterations[i].top].offset)
	}
	p.code = code
}

func fuseJumps(code isa.Instructions) isa.Instructions {
	res := code[:0]
	for i := 0; i < len(code); i++ {
		if fused, ok := fusedJump(code, i); ok {
			res = append(res, fused)
			i++
			continue
		}
		res = append(res, code[i])
	}
	return res
}

func fusedJump(code isa.Instructions, i int) (isa.Instruction, bool) {
	if i+1 == len(code) {
		return 0, false
	}
	next := code[i+1].Opcode()
	if next != isa.CondJump && next != isa.JumpIf {
		return 0, false
	}
	pair := [2]isa.Opcode{code[i].Opcode(), next}
	if jump, ok := fusedJumps[pair]; ok {
		return jump.New(code[i+1].Operand()), true
	}
	return 0, false
}
