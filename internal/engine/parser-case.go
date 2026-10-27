package engine

func (p *parser) caseExpression() (*node, error) {
	if err := p.expect(kindLBrace); err != nil {
		return nil, err
	}
	result := &node{kind: nodeCase}
	var fallback *node
	for {
		condition, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		if !p.take(kindColon) {
			fallback = condition
			break
		}
		value, err := p.pipeline()
		if err != nil {
			return nil, err
		}
		result.children = append(result.children, condition, value)
		if err := p.expect(kindComma); err != nil {
			return nil, err
		}
	}
	if len(result.children) == 0 {
		return nil, p.error("case requires a conditional arm")
	}
	result.children = append(result.children, fallback)
	p.take(kindComma)
	return result, p.expect(kindRBrace)
}
