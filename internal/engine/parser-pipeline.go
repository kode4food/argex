package engine

// discardName binds nothing in its position
const discardName = "_"

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
	for stage := 0; p.current().kind == "|"; stage++ {
		left, err = p.collection(left, stage > 0 && chainable(left))
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

// collection parses one stage. A chained stage's body also sees the bindings
// of the earlier stages in its pipeline
func (p *parser) collection(receiver *node, chained bool) (*node, error) {
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
	stage := &node{kind: operation.text, chained: chained}
	if err := p.stageBinding(stage); err != nil {
		return nil, err
	}
	body, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	stage.children = []*node{receiver, body}
	return stage, nil
}

func (p *parser) stageBinding(stage *node) error {
	if !p.hasBinding() {
		return nil
	}
	first, err := p.bindingName()
	if err != nil {
		return err
	}
	if !p.take(",") {
		stage.valueBinding = first
		return p.expect(":")
	}
	stage.keyBinding = first
	stage.valueBinding, err = p.bindingName()
	if err != nil {
		return err
	}
	return p.expect(":")
}

// hasBinding looks for "name:" or "name, name:". Neither can begin an
// expression, so anything else is a body without bindings
func (p *parser) hasBinding() bool {
	if p.peek(0).kind != "name" {
		return false
	}
	switch p.peek(1).kind {
	case ":":
		return true

	case ",":
		return p.peek(2).kind == "name" && p.peek(3).kind == ":"

	default:
		return false
	}
}

// bindingName returns "" for a discard, which binds no name
func (p *parser) bindingName() (string, error) {
	name := p.current()
	if err := p.expect("name"); err != nil {
		return "", err
	}
	if reservedNames[name.text] {
		return "", syntax(name.pos, "reserved binding name")
	}
	if name.text == discardName {
		return "", nil
	}
	return name.text, nil
}
