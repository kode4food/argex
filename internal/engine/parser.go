package engine

type parser struct {
	tokens []token
	pos    int
	depth  int
}

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

func (p *parser) expression(minimum int) (*node, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > maxDepth {
		return nil, LimitError
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
		level := precedence(op)
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
		child, err := p.expression(precedenceUnary)
		return &node{
			kind:     "unary",
			text:     t.kind,
			children: []*node{child},
		}, err

	case "(":
		expr, err := p.expression(precedenceOr)
		if err != nil {
			return nil, err
		}
		return expr, p.expect(")")

	case "[":
		children, err := p.list("]")
		return &node{kind: "array", children: children}, err

	case "{":
		return p.object()

	default:
		return nil, syntax(t.pos, "expected an expression")
	}
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

		case "|":
			left, err = p.collection(left)

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
	key, err := p.expression(precedenceOr)
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
	if operation.text != "map" && operation.text != "filter" {
		return nil, syntax(operation.pos, "unknown collection operation")
	}
	name := p.current()
	if err := p.expect("name"); err != nil {
		return nil, err
	}
	if name.text == "true" || name.text == "false" ||
		name.text == "null" || name.text == "case" {
		return nil, syntax(name.pos, "reserved binding name")
	}
	if err := p.expect(":"); err != nil {
		return nil, err
	}
	body, err := p.expression(precedenceOr)
	return &node{
		kind:     operation.text,
		text:     name.text,
		children: []*node{receiver, body},
	}, err
}

func (p *parser) list(end string) ([]*node, error) {
	var children []*node
	for !p.take(end) {
		child, err := p.expression(precedenceOr)
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

func (p *parser) object() (*node, error) {
	result := &node{kind: "object"}
	seen := map[string]bool{}
	for !p.take("}") {
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
		value, err := p.expression(precedenceOr)
		if err != nil {
			return nil, err
		}
		result.keys = append(result.keys, name)
		result.children = append(result.children, value)
		if p.take("}") {
			return result, nil
		}
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	return result, nil
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
		condition, err := p.expression(precedenceOr)
		if err != nil {
			return nil, err
		}
		if !p.take(":") {
			fallback = condition
			break
		}
		value, err := p.expression(precedenceOr)
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

// Parse parses a complete expression; whitespace never determines grouping
func Parse(source string) (*Expression, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	root, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	if err := p.expect("eof"); err != nil {
		return nil, err
	}
	return &Expression{root: root}, nil
}

func precedence(op string) int {
	switch op {
	case "||":
		return precedenceOr

	case "&&":
		return precedenceAnd

	case "??":
		return precedenceCoalesce

	case "==", "!=":
		return precedenceEquality

	case "<", "<=", ">", ">=", "in":
		return precedenceComparison

	case "+", "-":
		return precedenceAdd

	case "*", "/", "%":
		return precedenceMultiply

	default:
		return 0
	}
}
