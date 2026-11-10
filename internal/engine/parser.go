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
		tokens       []token
		pos          int
		depth        int
		conditionEnd int
		caseRetries  int
	}

	// nameCollector gathers the input names an expression reads unbound
	nameCollector struct {
		seen map[string]bool
		env  *Env
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
	nodeReduce
	nodeDefault
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

// Names reports the input properties expr reads by bare name, sorted. Local bindings, globals, and
// reads through this are excluded
func (e *Env) Names(expr *Expression) []string {
	if expr == nil || expr.root == nil || e == nil {
		return nil
	}
	c := nameCollector{seen: map[string]bool{}, env: e}
	c.collect(expr.root, nil)
	return slices.Sorted(maps.Keys(c.seen))
}

func (n *node) clone() *node {
	// Allocate the tree together instead of allocating each node separately
	count := n.count()
	nodes := make([]node, count)
	children := make([]*node, count-1)
	n.cloneInto(nodes, children)
	return &nodes[0]
}

func (n *node) count() int {
	count := 1
	for _, child := range n.children {
		count += child.count()
	}
	return count
}

func (n *node) cloneInto(nodes []node, children []*node) int {
	result := &nodes[0]
	*result = *n
	count := len(n.children)
	result.children = children[:count:count]
	children = children[count:]
	used := 1
	for i, child := range n.children {
		result.children[i] = &nodes[used]
		copied := child.cloneInto(nodes[used:], children)
		used += copied
		children = children[copied-1:]
	}
	return used
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

// collect scopes bindings as the compiler does: a collection's body sees its own and its pipeline's
// bindings, its receiver only the enclosing ones
func (c nameCollector) collect(n *node, bound []string) {
	switch n.kind {
	case nodeName:
		if _, ok := c.env.resolve(n.text); !ok &&
			!slices.Contains(bound, n.text) {
			c.seen[n.text] = true
		}
		return

	case nodeMap, nodeFilter, nodeAny, nodeAll, nodeReduce:
		c.collect(n.children[0], bound)
		if n.kind == nodeReduce {
			c.collectAccumulator(n, bound)
		}
		if pattern := n.bindingPattern(); pattern != nil {
			inner := append(bound[:len(bound):len(bound)], stageOuterNames(n)...)
			inner = c.collectPattern(pattern, inner)
			c.collect(n.children[1], inner)
			return
		}
		inner := append(bound[:len(bound):len(bound)], stageNames(n)...)
		c.collect(n.children[1], inner)
		return

	case nodeLet:
		c.collect(n.children[0], bound)
		if pattern := n.bindingPattern(); pattern != nil {
			inner := c.collectPattern(pattern, bound[:len(bound):len(bound)])
			c.collect(n.children[1], inner)
			return
		}
		inner := n.bindingNames(bound[:len(bound):len(bound)])
		c.collect(n.children[1], inner)
		return
	}
	for _, child := range n.children {
		c.collect(child, bound)
	}
}

// collectAccumulator reads the initial value outside the loop. Pattern defaults run per element, so
// they see the earlier stages' bindings
func (c nameCollector) collectAccumulator(n *node, bound []string) {
	accumulator := n.accumulator()
	c.collect(accumulator.children[1], bound)
	inner := bound[:len(bound):len(bound)]
	if n.chained {
		inner = append(inner, stageNames(n.children[0])...)
	}
	c.collectPattern(accumulator.children[0], inner)
}

func stageNames(n *node) []string {
	var names []string
	if n.chained {
		names = stageNames(n.children[0])
	}
	names = accumulatorNames(n, names)
	names = n.bindingNames(names)
	if n.keyBinding != "" {
		names = append(names, n.keyBinding)
	}
	return names
}

func stageOuterNames(n *node) []string {
	var names []string
	if n.chained {
		names = stageNames(n.children[0])
	}
	names = accumulatorNames(n, names)
	if n.keyBinding != "" {
		names = append(names, n.keyBinding)
	}
	return names
}

// accumulatorNames precedes the element bindings, which shadow it
func accumulatorNames(n *node, names []string) []string {
	if n.kind != nodeReduce {
		return names
	}
	return patternNames(n.accumulator().children[0], names)
}
