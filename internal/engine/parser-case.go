package engine

// caseSubject names a subject case's hidden binding. It can't be an identifier
const caseSubject = "case subject"

// caseExpression parses a boolean case. A subject case becomes a let of the
// subject around a boolean case whose conditions compare it with ==
func (p *parser) caseExpression() (*node, error) {
	if p.current().kind == kindLBrace {
		return p.caseArms()
	}
	subject, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	arms, err := p.caseArms()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(arms.children)-1; i += 2 {
		name := &node{kind: nodeName, text: caseSubject}
		arms.children[i] = &node{
			kind:     nodeBinary,
			text:     opEq,
			children: []*node{name, arms.children[i]},
		}
	}
	return &node{
		kind:         nodeLet,
		valueBinding: caseSubject,
		children:     []*node{subject, arms},
	}, nil
}

func (p *parser) caseArms() (*node, error) {
	if err := p.expect(kindLBrace); err != nil {
		return nil, err
	}
	result := &node{kind: nodeCase}
	var fallback *node
	for {
		condition, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		if !p.take(kindColon) {
			fallback = condition
			break
		}
		value, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, condition, value)
		if err := p.expect(kindComma); err != nil {
			return nil, err
		}
	}
	if len(result.children) == 0 {
		return nil, p.error("case requires a conditional arm")
	}
	result.children = append(result.children, fallback)
	p.take(kindComma)
	return result, p.expect(kindRBrace)
}
