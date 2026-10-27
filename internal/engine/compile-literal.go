package engine

import "github.com/kode4food/argex/internal/isa"

// array builds an array literal. A leading run of plain elements uses one Array
// op; each later element folds in with Push, each spread with Concat
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
