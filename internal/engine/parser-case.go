package engine

func (p *parser) caseExpression() (*node, error) {
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	result := &node{kind: "case"}
	var fallback *node
	for {
		condition, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		if !p.take(":") {
			fallback = condition
			break
		}
		value, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, condition, value)
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	if len(result.children) == 0 {
		return nil, p.error("case requires a conditional arm")
	}
	result.children = append(result.children, fallback)
	p.take(",")
	return result, p.expect("}")
}
