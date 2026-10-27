package engine

var reservedNames = map[string]bool{
	"true":  true,
	"false": true,
	"null":  true,
	"case":  true,
	"this":  true,
}

// pipeline chains collection operations left to right; the pipe is the
// lowest-precedence operator, so a binding body stops before the next pipe
func (p *parser) pipeline() (*node, error) {
	left, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	for p.current().kind == "|" {
		left, err = p.collection(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *parser) collection(receiver *node) (*node, error) {
	p.pos++
	operation := p.current()
	if err := p.expect("name"); err != nil {
		return nil, err
	}
	switch operation.text {
	case "map", "filter", "any", "all":
	default:
		return nil, syntax(operation.pos, "unknown collection operation")
	}
	first, err := p.bindingName()
	if err != nil {
		return nil, err
	}
	keyBinding, valueBinding := "", first
	if p.take(",") {
		keyBinding = first
		valueBinding, err = p.bindingName()
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(":"); err != nil {
		return nil, err
	}
	body, err := p.expression(precedenceOr)
	return &node{
		kind:         operation.text,
		valueBinding: valueBinding,
		keyBinding:   keyBinding,
		children:     []*node{receiver, body},
	}, err
}

func (p *parser) bindingName() (string, error) {
	name := p.current()
	if err := p.expect("name"); err != nil {
		return "", err
	}
	if reservedNames[name.text] {
		return "", syntax(name.pos, "reserved binding name")
	}
	return name.text, nil
}
