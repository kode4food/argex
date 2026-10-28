package engine

// letBinding nests one node per name, so later values see earlier bindings. Its
// depth counts toward the limit that expression enforces on the value
func (p *parser) letBinding() (*node, error) {
	p.depth++
	defer func() { p.depth-- }()
	name, err := p.bindingName()
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
	return &node{
		kind:         nodeLet,
		valueBinding: name,
		children:     []*node{value, body},
	}, nil
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
