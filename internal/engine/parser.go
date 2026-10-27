package engine

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

const (
	precedenceOr = iota + 1
	precedenceAnd
	precedenceCoalesce
	precedenceEquality
	precedenceComparison
	precedenceAdd
	precedenceMultiply
	precedenceUnary
)

const maxDepth = 256

var precedences = map[string]int{
	"||": precedenceOr,
	"&&": precedenceAnd,
	"??": precedenceCoalesce,
	"==": precedenceEquality,
	"!=": precedenceEquality,
	"<":  precedenceComparison,
	"<=": precedenceComparison,
	">":  precedenceComparison,
	">=": precedenceComparison,
	"in": precedenceComparison,
	"+":  precedenceAdd,
	"-":  precedenceAdd,
	"*":  precedenceMultiply,
	"/":  precedenceMultiply,
	"%":  precedenceMultiply,
}

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

// pipeline chains collection operations left to right; the pipe is the
// lowest-precedence operator, so a binding body stops before the next pipe
func (p *parser) pipeline() (*node, error) {
	left, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	for p.current().kind == "|" {
		left, err = p.collection(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *parser) expression(minimum int) (*node, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > maxDepth {
		return nil, ErrLimit
	}
	left, err := p.primary()
	if err != nil {
		return nil, err
	}
	left, err = p.postfix(left)
	if err != nil {
		return nil, err
	}
	compared := false
	for {
		op := p.current().kind
		if op == "name" && p.current().text == "in" {
			op = "in"
		}
		level := precedences[op]
		if level < minimum {
			return left, nil
		}
		if level == precedenceComparison && compared {
			return nil, p.error("comparison requires parentheses")
		}
		compared = compared || level == precedenceComparison
		p.pos++
		next := level + 1
		if op == "??" {
			next = level
		}
		right, err := p.expression(next)
		if err != nil {
			return nil, err
		}
		left = &node{
			kind:     "binary",
			text:     op,
			children: []*node{left, right},
		}
	}
}

func (p *parser) primary() (*node, error) {
	t := p.current()
	p.pos++
	switch t.kind {
	case "literal":
		return &node{kind: "literal", value: t.value}, nil

	case "name":
		return p.named(t.text)

	case "!", "-", "+":
		return p.unary(t.kind)

	case "(":
		return p.group()

	case "[":
		return p.array()

	case "{":
		return p.object()

	default:
		return nil, syntax(t.pos, "expected an expression")
	}
}

func (p *parser) unary(op string) (*node, error) {
	child, err := p.expression(precedenceUnary)
	if err != nil {
		return nil, err
	}
	return &node{kind: "unary", text: op, children: []*node{child}}, nil
}

func (p *parser) group() (*node, error) {
	expr, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	if err := p.expect(")"); err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *parser) named(name string) (*node, error) {
	switch name {
	case "true", "false":
		return &node{kind: "literal", value: name == "true"}, nil

	case "null":
		return &node{kind: "literal"}, nil

	case "case":
		return p.caseExpression()

	default:
		return &node{kind: "name", text: name}, nil
	}
}

func (p *parser) postfix(left *node) (*node, error) {
	for {
		var err error
		switch p.current().kind {
		case ".", "?.":
			left, err = p.member(left)

		case "[":
			p.pos++
			left, err = p.index(left, false)

		case "(":
			left, err = p.call(left)

		default:
			return left, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func (p *parser) member(receiver *node) (*node, error) {
	optional := p.current().kind == "?."
	p.pos++
	if optional && p.take("[") {
		return p.index(receiver, optional)
	}
	name := p.current()
	if err := p.expect("name"); err != nil {
		return nil, err
	}
	return &node{
		kind:     "access",
		text:     name.text,
		optional: optional,
		children: []*node{receiver},
	}, nil
}

func (p *parser) index(receiver *node, optional bool) (*node, error) {
	key, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	return &node{
		kind:     "index",
		optional: optional,
		children: []*node{receiver, key},
	}, p.expect("]")
}

func (p *parser) call(receiver *node) (*node, error) {
	if receiver.kind != "name" {
		return nil, p.error("only named functions can be called")
	}
	p.pos++
	children, err := p.list(")")
	if err != nil {
		return nil, err
	}
	return &node{
		kind:     "call",
		text:     receiver.text,
		children: children,
	}, nil
}

func (p *parser) collection(receiver *node) (*node, error) {
	p.pos++
	operation := p.current()
	if err := p.expect("name"); err != nil {
		return nil, err
	}
	switch operation.text {
	case "map", "filter", "any", "all":
	default:
		return nil, syntax(operation.pos, "unknown collection operation")
	}
	first, err := p.bindingName()
	if err != nil {
		return nil, err
	}
	keyBinding, valueBinding := "", first
	if p.take(",") {
		keyBinding = first
		valueBinding, err = p.bindingName()
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(":"); err != nil {
		return nil, err
	}
	body, err := p.expression(precedenceOr)
	return &node{
		kind:         operation.text,
		valueBinding: valueBinding,
		keyBinding:   keyBinding,
		children:     []*node{receiver, body},
	}, err
}

func (p *parser) bindingName() (string, error) {
	name := p.current()
	if err := p.expect("name"); err != nil {
		return "", err
	}
	if name.text == "true" || name.text == "false" ||
		name.text == "null" || name.text == "case" {
		return "", syntax(name.pos, "reserved binding name")
	}
	return name.text, nil
}

func (p *parser) list(end string) ([]*node, error) {
	var children []*node
	for !p.take(end) {
		child, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		children = append(children, child)
		if p.take(end) {
			return children, nil
		}
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	return children, nil
}

func (p *parser) array() (*node, error) {
	result := &node{kind: "array"}
	for !p.take("]") {
		element, err := p.spreadable()
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, element)
		if p.take("]") {
			return result, nil
		}
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *parser) object() (*node, error) {
	result := &node{kind: "object"}
	seen := map[string]bool{}
	for !p.take("}") {
		entry, err := p.objectMember(seen)
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, entry)
		if p.take("}") {
			return result, nil
		}
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *parser) objectMember(seen map[string]bool) (*node, error) {
	if p.current().kind == "..." {
		return p.spreadable()
	}
	if p.current().kind == "[" {
		return p.computedMember(seen)
	}
	key := p.current()
	name, err := p.objectKey()
	if err != nil {
		return nil, err
	}
	p.pos++
	if seen[name] {
		return nil, syntax(key.pos, "duplicate object key")
	}
	seen[name] = true
	if err := p.expect(":"); err != nil {
		return nil, err
	}
	value, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	return &node{kind: "field", text: name, children: []*node{value}}, nil
}

// computedMember parses a [key]: value member. A string literal key is a plain
// field, so it keeps the constant-key path and duplicate check
func (p *parser) computedMember(seen map[string]bool) (*node, error) {
	start := p.current()
	p.pos++
	key, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	if err := p.expect("]"); err != nil {
		return nil, err
	}
	if err := p.expect(":"); err != nil {
		return nil, err
	}
	value, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	name, ok := key.value.(string)
	if key.kind != "literal" || !ok {
		children := []*node{key, value}
		return &node{kind: "computed", children: children}, nil
	}
	if seen[name] {
		return nil, syntax(start.pos, "duplicate object key")
	}
	seen[name] = true
	return &node{kind: "field", text: name, children: []*node{value}}, nil
}

func (p *parser) spreadable() (*node, error) {
	spread := p.take("...")
	expr, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	if !spread {
		return expr, nil
	}
	return &node{kind: "spread", children: []*node{expr}}, nil
}

func (p *parser) objectKey() (string, error) {
	key := p.current()
	if key.kind == "name" {
		return key.text, nil
	}
	if name, ok := key.value.(string); key.kind == "literal" && ok {
		return name, nil
	}
	return "", p.error("expected an object key")
}

func (p *parser) caseExpression() (*node, error) {
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	result := &node{kind: "case"}
	var fallback *node
	for {
		condition, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		if !p.take(":") {
			fallback = condition
			break
		}
		value, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, condition, value)
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	if len(result.children) == 0 {
		return nil, p.error("case requires a conditional arm")
	}
	result.children = append(result.children, fallback)
	p.take(",")
	return result, p.expect("}")
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
