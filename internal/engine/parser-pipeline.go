package engine

// discardName binds nothing in its position
const discardName = "_"

var reservedNames = map[string]bool{
	kwTrue:  true,
	kwFalse: true,
	kwNull:  true,
	kwCase:  true,
	kwLet:   true,
	kwThis:  true,
}

var collectionKinds = map[string]nodeKind{
	"map":    nodeMap,
	"filter": nodeFilter,
	"any":    nodeAny,
	"all":    nodeAll,
}

var collectionNames = map[nodeKind]string{
	nodeMap:    "map",
	nodeFilter: "filter",
	nodeAny:    "any",
	nodeAll:    "all",
}

// pipeline chains collection operations left to right, and a stage body stops
// before the next pipe. A let shares this level, so its body is a pipeline
func (p *parser) pipeline() (*node, error) {
	if p.current().kind == kindName && p.current().text == kwLet {
		p.pos++
		return p.letBinding()
	}
	left, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	for stage := 0; p.current().kind == kindPipe; stage++ {
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
	if err := p.expect(kindName); err != nil {
		return nil, err
	}
	kind, ok := collectionKinds[operation.text]
	if !ok {
		return nil, syntax(operation.pos, "unknown collection operation")
	}
	stage := &node{kind: kind, chained: chained}
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
	if !p.take(kindComma) {
		stage.valueBinding = first
		return p.expect(kindColon)
	}
	stage.keyBinding = first
	stage.valueBinding, err = p.bindingName()
	if err != nil {
		return err
	}
	return p.expect(kindColon)
}

// hasBinding looks for "name:" or "name, name:". Neither can begin an
// expression, so anything else is a body without bindings
func (p *parser) hasBinding() bool {
	if p.peek(0).kind != kindName {
		return false
	}
	switch p.peek(1).kind {
	case kindColon:
		return true

	case kindComma:
		return p.peek(2).kind == kindName && p.peek(3).kind == kindColon

	default:
		return false
	}
}

// bindingName returns "" for a discard, which binds no name
func (p *parser) bindingName() (string, error) {
	name := p.current()
	if err := p.expect(kindName); err != nil {
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
