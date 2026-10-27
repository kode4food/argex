package engine

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
