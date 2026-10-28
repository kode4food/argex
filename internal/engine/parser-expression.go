package engine

const (
	kwTrue  = "true"
	kwFalse = "false"
	kwNull  = "null"
	kwCase  = "case"
	kwLet   = "let"
	kwThis  = "this"
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

var precedences = map[tokenKind]int{
	kindOr:       precedenceOr,
	kindAnd:      precedenceAnd,
	kindCoalesce: precedenceCoalesce,
	kindEq:       precedenceEquality,
	kindNeq:      precedenceEquality,
	kindLt:       precedenceComparison,
	kindLte:      precedenceComparison,
	kindGt:       precedenceComparison,
	kindGte:      precedenceComparison,
	kindIn:       precedenceComparison,
	kindAdd:      precedenceAdd,
	kindSub:      precedenceAdd,
	kindMul:      precedenceMultiply,
	kindDiv:      precedenceMultiply,
	kindMod:      precedenceMultiply,
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
		if op == kindName && p.current().text == opIn {
			op = kindIn
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
		if op == kindCoalesce {
			next = level
		}
		right, err := p.expression(next)
		if err != nil {
			return nil, err
		}
		left = &node{
			kind:     nodeBinary,
			text:     operatorText[op],
			children: []*node{left, right},
		}
	}
}

func (p *parser) primary() (*node, error) {
	t := p.current()
	p.pos++
	switch t.kind {
	case kindLiteral:
		return &node{kind: nodeLiteral, value: t.value}, nil

	case kindName:
		return p.named(t.text)

	case kindNot, kindSub, kindAdd:
		return p.unary(operatorText[t.kind])

	case kindLParen:
		return p.group()

	case kindLBrack:
		return p.array()

	case kindLBrace:
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
	return &node{kind: nodeUnary, text: op, children: []*node{child}}, nil
}

func (p *parser) group() (*node, error) {
	expr, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	if err := p.expect(kindRParen); err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *parser) named(name string) (*node, error) {
	switch name {
	case kwTrue, kwFalse:
		return &node{kind: nodeLiteral, value: name == kwTrue}, nil

	case kwNull:
		return &node{kind: nodeLiteral}, nil

	case kwCase:
		return p.caseExpression()

	case kwLet:
		return nil, syntax(p.peek(-1).pos, "let requires parentheses")

	case kwThis:
		return &node{kind: nodeRoot}, nil

	default:
		return &node{kind: nodeName, text: name}, nil
	}
}
