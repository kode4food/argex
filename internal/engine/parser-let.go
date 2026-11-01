package engine

// letBinding nests one node per assignment, so later values see earlier names.
// Its depth counts toward the limit that expression enforces on the value
func (p *parser) letBinding() (*node, error) {
	p.depth++
	defer func() { p.depth-- }()
	result := &node{kind: nodeLet}
	pattern, err := p.bindingTarget(result)
	if err != nil {
		return nil, err
	}
	if err := p.expect(kindAssign); err != nil {
		return nil, err
	}
	value, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	body, err := p.letBody()
	if err != nil {
		return nil, err
	}
	result.children = []*node{value, body}
	if pattern != nil {
		result.children = append(result.children, pattern)
	}
	return result, nil
}

func (p *parser) letBody() (*node, error) {
	if p.take(kindComma) {
		return p.letBinding()
	}
	if err := p.expect(kindColon); err != nil {
		return nil, err
	}
	return p.pipeline()
}
