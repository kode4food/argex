package engine

import "github.com/kode4food/argex/internal/isa"

// array builds an array of elements. A leading run of plain elements uses one
// Array op. Each later element folds in with Push, each spread with Concat
func (c *compiler) array(elements []*node, depth int) error {
	fixed := 0
	for fixed < len(elements) && elements[fixed].kind != nodeSpread {
		if err := c.compile(elements[fixed], depth+1); err != nil {
			return err
		}
		fixed++
	}
	c.emit(isa.Array, isa.Operand(fixed))
	for _, child := range elements[fixed:] {
		if child.kind == nodeSpread {
			if err := c.compile(child.children[0], depth+1); err != nil {
				return err
			}
			c.emit(isa.Concat, 0)
			continue
		}
		if err := c.compile(child, depth+1); err != nil {
			return err
		}
		c.emit(isa.Push, 0)
	}
	return nil
}

// object builds an object literal. A leading run of plain fields uses one
// Object op. Each later field folds in with Put, each spread with Merge
func (c *compiler) object(n *node, depth int) error {
	var keys []string
	fixed := 0
	for fixed < len(n.children) && n.children[fixed].kind == nodeField {
		field := n.children[fixed]
		keys = append(keys, field.text)
		if err := c.compile(field.children[0], depth+1); err != nil {
			return err
		}
		fixed++
	}
	c.emit(isa.Object, c.constant(keys))
	for _, child := range n.children[fixed:] {
		switch child.kind {
		case nodeSpread:
			if err := c.compile(child.children[0], depth+1); err != nil {
				return err
			}
			c.emit(isa.Merge, 0)
			continue

		case nodeField:
			c.emit(isa.Const, c.constant(child.text))
		}
		for _, part := range child.children {
			if err := c.compile(part, depth+1); err != nil {
				return err
			}
		}
		c.emit(isa.Put, 0)
	}
	return nil
}
