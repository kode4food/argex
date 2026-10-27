package engine

import (
	"maps"
	"slices"
)

type (
	// Expression is an immutable parsed expression
	Expression struct {
		root *node
	}

	parser struct {
		tokens []token
		pos    int
		depth  int
	}

	node struct {
		kind         string
		text         string
		valueBinding string
		keyBinding   string
		value        any
		children     []*node
		optional     bool
	}
)

const maxDepth = 256

// Parse parses a complete expression; whitespace never determines grouping
func Parse(source string) (*Expression, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	root, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	if err := p.expect("eof"); err != nil {
		return nil, err
	}
	return &Expression{root: root}, nil
}

// Names reports the input properties the expression reads by bare name, sorted.
// Names bound by collection operations and reads through this are excluded
func (e *Expression) Names() []string {
	if e == nil || e.root == nil {
		return nil
	}
	seen := map[string]bool{}
	collectNames(e.root, nil, seen)
	return slices.Sorted(maps.Keys(seen))
}

func (p *parser) expect(kind string) error {
	if !p.take(kind) {
		return p.error("expected " + kind)
	}
	return nil
}

func (p *parser) take(kind string) bool {
	if p.current().kind != kind {
		return false
	}
	p.pos++
	return true
}

func (p *parser) current() token {
	if p.pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos]
}

func (p *parser) error(message string) error {
	return syntax(p.current().pos, message)
}

// collectNames scopes bindings as the compiler does: a collection's body sees
// its own bindings, its receiver only the enclosing ones
func collectNames(n *node, bound []string, seen map[string]bool) {
	switch n.kind {
	case "name":
		if !slices.Contains(bound, n.text) {
			seen[n.text] = true
		}
		return

	case "map", "filter", "any", "all":
		collectNames(n.children[0], bound, seen)
		inner := append(bound[:len(bound):len(bound)], n.valueBinding)
		if n.keyBinding != "" {
			inner = append(inner, n.keyBinding)
		}
		collectNames(n.children[1], inner, seen)
		return
	}
	for _, child := range n.children {
		collectNames(child, bound, seen)
	}
}
