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

	node struct {
		kind         nodeKind
		text         string
		valueBinding string
		keyBinding   string
		value        any
		children     []*node
		optional     bool
		chained      bool
		inferred     TypeSet
	}

	// nodeKind identifies an AST node's grammar production
	nodeKind int

	parser struct {
		tokens []token
		pos    int
		depth  int
	}
)

const (
	nodeLiteral nodeKind = iota
	nodeName
	nodeRoot
	nodeAccess
	nodeIndex
	nodeCall
	nodeUnary
	nodeBinary
	nodeArray
	nodeObject
	nodeField
	nodeSpread
	nodeComputed
	nodeCase
	nodeMatch
	nodeLet
	nodeMap
	nodeFilter
	nodeAny
	nodeAll
)

const maxDepth = 256

// Parse parses a complete expression. Whitespace never determines grouping
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
	if err := p.expect(kindEOF); err != nil {
		return nil, err
	}
	return &Expression{root: root}, nil
}

// Names reports the input properties expr reads by bare name, sorted. Local
// bindings, globals, and reads through this are excluded
func (e *Env) Names(expr *Expression) []string {
	if expr == nil || expr.root == nil || e == nil {
		return nil
	}
	seen := map[string]bool{}
	collectNames(expr.root, nil, seen, e)
	return slices.Sorted(maps.Keys(seen))
}

func (n *node) clone() *node {
	result := *n
	result.children = make([]*node, len(n.children))
	for i, child := range n.children {
		result.children[i] = child.clone()
	}
	return &result
}

func (p *parser) expect(kind tokenKind) error {
	if !p.take(kind) {
		return p.error("expected " + kind.String())
	}
	return nil
}

func (p *parser) take(kind tokenKind) bool {
	if p.current().kind != kind {
		return false
	}
	p.pos++
	return true
}

func (p *parser) current() token {
	return p.peek(0)
}

func (p *parser) peek(offset int) token {
	if p.pos+offset >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos+offset]
}

func (p *parser) error(message string) error {
	return syntax(p.current().pos, message)
}

// collectNames scopes bindings as the compiler does: a collection's body sees
// its own and its pipeline's bindings, its receiver only the enclosing ones
func collectNames(
	n *node, bound []string, seen map[string]bool, env *Env,
) {
	switch n.kind {
	case nodeName:
		if _, ok := env.resolve(n.text); !ok &&
			!slices.Contains(bound, n.text) {
			seen[n.text] = true
		}
		return

	case nodeMap, nodeFilter, nodeAny, nodeAll:
		collectNames(n.children[0], bound, seen, env)
		inner := append(bound[:len(bound):len(bound)], stageNames(n)...)
		collectNames(n.children[1], inner, seen, env)
		return

	case nodeLet:
		collectNames(n.children[0], bound, seen, env)
		inner := append(bound[:len(bound):len(bound)], n.valueBinding)
		collectNames(n.children[1], inner, seen, env)
		return
	}
	for _, child := range n.children {
		collectNames(child, bound, seen, env)
	}
}

func stageNames(n *node) []string {
	var names []string
	if n.chained {
		names = stageNames(n.children[0])
	}
	for _, name := range []string{n.valueBinding, n.keyBinding} {
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
