package engine

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
