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
	pattern, err := p.stageBinding(stage)
	if err != nil {
		return nil, err
	}
	body, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	stage.children = []*node{receiver, body}
	if pattern != nil {
		stage.children = append(stage.children, pattern)
	}
	return stage, nil
}

func (p *parser) stageBinding(stage *node) (*node, error) {
	if !p.hasBinding() {
		return nil, nil
	}
	pattern, err := p.bindingTarget(stage)
	if err != nil {
		return nil, err
	}
	if !p.take(kindComma) {
		return pattern, p.expect(kindColon)
	}
	stage.keyBinding = stage.valueBinding
	stage.valueBinding = ""
	pattern, err = p.bindingTarget(stage)
	if err != nil {
		return nil, err
	}
	return pattern, p.expect(kindColon)
}

// hasBinding checks the delimiter after a complete target, so array and object
// expressions without a following colon remain ordinary stage bodies
func (p *parser) hasBinding() bool {
	end := p.bindingColon(p.pos)
	return end >= 0 && end != p.conditionEnd
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
