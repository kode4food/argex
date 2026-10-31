package engine

func (p *parser) postfix(left *node) (*node, error) {
	for {
		var err error
		switch p.current().kind {
		case kindDot, kindOptDot:
			left, err = p.member(left)

		case kindLBrack:
			p.pos++
			left, err = p.index(left, false)

		case kindLParen:
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
	optional := p.current().kind == kindOptDot
	p.pos++
	if optional && p.take(kindLBrack) {
		return p.index(receiver, optional)
	}
	name := p.current()
	if err := p.expect(kindName); err != nil {
		return nil, err
	}
	return &node{
		kind:     nodeAccess,
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
		kind:     nodeIndex,
		optional: optional,
		children: []*node{receiver, key},
	}, p.expect(kindRBrack)
}

func (p *parser) call(receiver *node) (*node, error) {
	p.pos++
	args, err := p.list(kindRParen)
	if err != nil {
		return nil, err
	}
	children := append([]*node{receiver}, args...)
	return &node{kind: nodeCall, children: children}, nil
}

func (p *parser) list(end tokenKind) ([]*node, error) {
	var children []*node
	for !p.take(end) {
		child, err := p.spreadable()
		if err != nil {
			return nil, err
		}
		children = append(children, child)
		if p.take(end) {
			return children, nil
		}
		if err := p.expect(kindComma); err != nil {
			return nil, err
		}
	}
	return children, nil
}
