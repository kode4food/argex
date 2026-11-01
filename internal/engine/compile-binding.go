package engine

import "github.com/kode4food/argex/internal/isa"

func (c *compiler) bindPattern(
	pattern *node, slot int, bound []binding, depth int,
) ([]binding, error) {
	want := TypeSet(pattern.patternType())
	if !pattern.inferred.Known() || pattern.inferred&^want != 0 {
		c.emit(isa.Load, isa.Operand(slot))
		c.emit(isa.AssertType, isa.Operand(want))
		c.emit(isa.Pop, 0)
	}
	for i, child := range pattern.children {
		target := child
		if pattern.kind == nodeObject {
			target = child.children[0]
		}
		if target.kind == nodeName && target.text == "" {
			continue
		}
		c.emit(isa.Load, isa.Operand(slot))
		switch pattern.kind {
		case nodeObject:
			op := isa.Access
			if target.kind == nodeDefault {
				op = isa.OptionalAccess
			}
			c.emit(op, c.constant(child.text))

		default:
			c.emit(isa.Const, c.constant(i))
			op := isa.Index
			if target.kind == nodeDefault {
				op = isa.OptionalIndex
			}
			c.emit(op, 0)
		}
		if target.kind == nodeName &&
			(!target.inferred.Known() || target.inferred.Has(TypeMissing)) {
			c.emit(isa.AssertType, isa.Operand(^TypeSet(TypeMissing)))
		}
		target, err := c.patternDefault(target, bound, depth)
		if err != nil {
			return nil, err
		}
		if target.kind == nodeName && target.text == "" {
			c.emit(isa.Pop, 0)
			continue
		}
		next := c.allocSlot()
		c.emit(isa.Store, isa.Operand(next))
		if target.kind == nodeName {
			bound = append(bound, binding{name: target.text, slot: next})
			continue
		}
		bound, err = c.bindPattern(target, next, bound, depth+1)
		if err != nil {
			return nil, err
		}
	}
	return bound, nil
}

func (c *compiler) patternDefault(
	target *node, bound []binding, depth int,
) (*node, error) {
	if target.kind != nodeDefault {
		return target, nil
	}
	end := c.newLabel()
	c.emit(isa.Coalesce, end)
	if err := c.scoped(bound, target.children[1], depth); err != nil {
		return nil, err
	}
	c.emit(isa.Label, end)
	return target.children[0], nil
}
