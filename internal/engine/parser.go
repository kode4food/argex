package engine

type (
	// Expression is an immutable parsed expression
	Expression struct {
		root *node
	}

	parser struct {
		tokens []token
		pos    int
		depth  int
	}

	node struct {
		kind         string
		text         string
		valueBinding string
		keyBinding   string
		value        any
		children     []*node
		optional     bool
	}
)

const maxDepth = 256

// Parse parses a complete expression; whitespace never determines grouping
func Parse(source string) (*Expression, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	root, err := p.pipeline()
	if err != nil {
		return nil, err
	}
	if err := p.expect("eof"); err != nil {
		return nil, err
	}
	return &Expression{root: root}, nil
}

func (p *parser) expect(kind string) error {
	if !p.take(kind) {
		return p.error("expected " + kind)
	}
	return nil
}

func (p *parser) take(kind string) bool {
	if p.current().kind != kind {
		return false
	}
	p.pos++
	return true
}

func (p *parser) current() token {
	if p.pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos]
}

func (p *parser) error(message string) error {
	return syntax(p.current().pos, message)
}
