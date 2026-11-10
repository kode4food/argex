package engine

import "errors"

type caseBranch struct {
	condition *node
	value     *node
}

const maxCaseRetries = 256

func (p *parser) caseExpression() (*node, error) {
	if p.current().kind == kindLBrace {
		return p.caseArms()
	}
	subject, err := p.expression(precedenceOr)
	if err != nil {
		return nil, err
	}
	arms, err := p.caseArms()
	if err != nil {
		return nil, err
	}
	return &node{
		kind:     nodeMatch,
		children: []*node{subject, arms},
	}, nil
}

func (p *parser) caseArms() (*node, error) {
	if err := p.expect(kindLBrace); err != nil {
		return nil, err
	}
	result := &node{kind: nodeCase}
	var fallback *node
	for {
		branch, err := p.caseBranch()
		if err != nil {
			return nil, err
		}
		if branch.condition == nil {
			fallback = branch.value
			break
		}
		result.children = append(result.children, branch.condition, branch.value)
	}
	if len(result.children) == 0 {
		return nil, p.error("case requires a conditional arm")
	}
	result.children = append(result.children, fallback)
	p.take(kindComma)
	return result, p.expect(kindRBrace)
}

func (p *parser) caseBranch() (caseBranch, error) {
	start := *p
	branch, err := p.parseCaseBranch()
	if err == nil || errors.Is(err, ErrLimit) {
		return branch, err
	}
	// A stage literal can resemble a pattern. Retry with the arm's colon reserved, leaving binding
	// colons inside the condition available
	depth := 0
	for i := start.pos; i < len(p.tokens); i++ {
		switch p.tokens[i].kind {
		case kindLParen, kindLBrack, kindLBrace:
			depth++

		case kindRParen, kindRBrack, kindRBrace, kindEOF:
			if depth == 0 {
				return caseBranch{}, err
			}
			depth--

		case kindPipe:
			if depth != 0 {
				continue
			}
			end := p.bindingColon(i + 2)
			if end < 0 {
				continue
			}
			kind := p.tokens[end-1].kind
			if kind != kindRBrace && kind != kindRBrack {
				continue
			}
			if p.caseRetries == maxCaseRetries {
				return caseBranch{}, ErrLimit
			}
			p.caseRetries++
			trial := start
			trial.conditionEnd = end
			trial.caseRetries = p.caseRetries
			branch, trialErr := trial.parseCaseBranch()
			p.caseRetries = trial.caseRetries
			if trialErr == nil {
				p.pos = trial.pos
				return branch, nil
			}
			if errors.Is(trialErr, ErrLimit) {
				return caseBranch{}, trialErr
			}
		}
	}
	return caseBranch{}, err
}

func (p *parser) parseCaseBranch() (caseBranch, error) {
	condition, err := p.pipeline()
	if err != nil {
		return caseBranch{}, err
	}
	if !p.take(kindColon) {
		if p.current().kind != kindRBrace &&
			!(p.current().kind == kindComma && p.peek(1).kind == kindRBrace) {
			return caseBranch{}, p.error("expected a case arm or fallback")
		}
		return caseBranch{value: condition}, nil
	}
	value, err := p.pipeline()
	if err != nil {
		return caseBranch{}, err
	}
	if err := p.expect(kindComma); err != nil {
		return caseBranch{}, err
	}
	return caseBranch{
		condition: condition,
		value:     value,
	}, nil
}
