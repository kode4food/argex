package engine

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
