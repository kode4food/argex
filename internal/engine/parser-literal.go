package engine

func (p *parser) array() (*node, error) {
	result := &node{kind: nodeArray}
	for !p.take(kindRBrack) {
		element, err := p.spreadable()
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, element)
		if p.take(kindRBrack) {
			return result, nil
		}
		if err := p.expect(kindComma); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *parser) object() (*node, error) {
	result := &node{kind: nodeObject}
	seen := map[string]bool{}
	for !p.take(kindRBrace) {
		entry, err := p.objectMember(seen)
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, entry)
		if p.take(kindRBrace) {
			return result, nil
		}
		if err := p.expect(kindComma); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *parser) objectMember(seen map[string]bool) (*node, error) {
	if p.current().kind == kindSpread {
		return p.spreadable()
	}
	if p.current().kind == kindLBrack {
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
	if key.kind == kindName && p.current().kind != kindColon {
		return shorthandField(key)
	}
	if err := p.expect(kindColon); err != nil {
		return nil, err
	}
	value, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	return &node{kind: nodeField, text: name, children: []*node{value}}, nil
}

func (p *parser) computedMember(seen map[string]bool) (*node, error) {
	start := p.current()
	p.pos++
	key, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	if err := p.expect(kindRBrack); err != nil {
		return nil, err
	}
	if err := p.expect(kindColon); err != nil {
		return nil, err
	}
	value, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	children := []*node{key, value}
	result := &node{kind: nodeComputed, children: children}
	name, ok := key.value.(string)
	if key.kind != nodeLiteral || !ok {
		return result, nil
	}
	if seen[name] {
		return nil, syntax(start.pos, "duplicate object key")
	}
	seen[name] = true
	return result, nil
}

func (p *parser) spreadable() (*node, error) {
	spread := p.take(kindSpread)
	expr, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	if !spread {
		return expr, nil
	}
	return &node{kind: nodeSpread, children: []*node{expr}}, nil
}

func (p *parser) objectKey() (string, error) {
	key := p.current()
	if key.kind == kindName {
		return key.text, nil
	}
	if name, ok := key.value.(string); key.kind == kindLiteral && ok {
		return name, nil
	}
	return "", p.error("expected an object key")
}

func shorthandField(key token) (*node, error) {
	if reservedNames[key.text] {
		return nil, syntax(key.pos, "reserved binding name")
	}
	value := &node{kind: nodeName, text: key.text}
	return &node{kind: nodeField, text: key.text, children: []*node{value}}, nil
}
