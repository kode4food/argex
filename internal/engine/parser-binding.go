package engine

// bindingPattern is the optional third child of a let or collection stage
func (n *node) bindingPattern() *node {
	if len(n.children) > 2 {
		return n.children[2]
	}
	return nil
}

func (n *node) bindingNames(names []string) []string {
	if pattern := n.bindingPattern(); pattern != nil {
		return patternNames(pattern, names)
	}
	if n.valueBinding != "" {
		names = append(names, n.valueBinding)
	}
	return names
}

func (n *node) patternType() TypeTag {
	if n.kind == nodeArray {
		return TypeArray
	}
	return TypeObject
}

func (p *parser) bindingTarget(n *node) (*node, error) {
	if p.current().kind != kindName {
		return p.pattern(map[string]bool{})
	}
	name, err := p.bindingName()
	if err != nil {
		return nil, err
	}
	n.valueBinding = name
	return nil, nil
}

func (p *parser) pattern(names map[string]bool) (*node, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > maxDepth {
		return nil, ErrLimit
	}
	switch p.current().kind {
	case kindName:
		start := p.current()
		name, err := p.bindingName()
		if err != nil {
			return nil, err
		}
		if name != "" && names[name] {
			return nil, syntax(start.pos, "duplicate binding name")
		}
		names[name] = true
		return &node{kind: nodeName, text: name}, nil

	case kindLBrack:
		return p.arrayPattern(names)

	case kindLBrace:
		return p.objectPattern(names)

	default:
		return nil, p.error("expected a binding pattern")
	}
}

func (p *parser) patternElement(names map[string]bool) (*node, error) {
	target, err := p.pattern(names)
	if err != nil {
		return nil, err
	}
	if !p.take(kindAssign) {
		return target, nil
	}
	fallback, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	return &node{kind: nodeDefault, children: []*node{target, fallback}}, nil
}

func (p *parser) arrayPattern(names map[string]bool) (*node, error) {
	p.pos++
	result := &node{kind: nodeArray}
	for !p.take(kindRBrack) {
		if p.take(kindComma) {
			result.children = append(result.children, &node{kind: nodeName})
			continue
		}
		child, err := p.patternElement(names)
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, child)
		if p.take(kindRBrack) {
			return result, nil
		}
		if err := p.expect(kindComma); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *parser) objectPattern(names map[string]bool) (*node, error) {
	p.pos++
	result := &node{kind: nodeObject}
	keys := map[string]bool{}
	for !p.take(kindRBrace) {
		key := p.current()
		name, err := p.objectKey()
		if err != nil {
			return nil, err
		}
		if keys[name] {
			return nil, syntax(key.pos, "duplicate object key")
		}
		keys[name] = true
		var child *node
		if p.peek(1).kind == kindColon {
			p.pos += 2
			child, err = p.patternElement(names)
		} else if key.kind == kindName {
			child, err = p.patternElement(names)
		} else {
			return nil, syntax(key.pos, "expected a binding name")
		}
		if err != nil {
			return nil, err
		}
		field := &node{kind: nodeField, text: name, children: []*node{child}}
		result.children = append(result.children, field)
		if p.take(kindRBrace) {
			return result, nil
		}
		if err := p.expect(kindComma); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *parser) bindingColon(pos int) int {
	if pos >= len(p.tokens) {
		return -1
	}
	end := p.bindingEnd(pos)
	if p.tokens[pos].kind == kindName && p.tokens[end].kind == kindComma {
		end = p.bindingEnd(end + 1)
	}
	if p.tokens[end].kind != kindColon {
		return -1
	}
	return end
}

func (p *parser) bindingEnd(pos int) int {
	if p.tokens[pos].kind == kindName {
		return pos + 1
	}
	if kind := p.tokens[pos].kind; kind != kindLBrack && kind != kindLBrace {
		return pos
	}
	depth := 0
	for i := pos; i < len(p.tokens); i++ {
		switch p.tokens[i].kind {
		case kindLBrack, kindLBrace:
			depth++

		case kindRBrack, kindRBrace:
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return pos
}

func patternNames(pattern *node, names []string) []string {
	if pattern.kind == nodeDefault {
		return patternNames(pattern.children[0], names)
	}
	if pattern.kind == nodeName && pattern.text != "" {
		names = append(names, pattern.text)
	}
	for _, child := range pattern.children {
		names = patternNames(child, names)
	}
	return names
}

func collectPatternNames(
	pattern *node, bound []string, seen map[string]bool, env *Env,
) []string {
	if pattern.kind == nodeDefault {
		collectNames(pattern.children[1], bound, seen, env)
		return collectPatternNames(pattern.children[0], bound, seen, env)
	}
	if pattern.kind == nodeName && pattern.text != "" {
		bound = append(bound, pattern.text)
	}
	for _, child := range pattern.children {
		bound = collectPatternNames(child, bound, seen, env)
	}
	return bound
}
