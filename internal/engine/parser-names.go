package engine

// bindingPattern is the optional child after a let or collection stage's body,
// and after a reduce stage's accumulator
func (n *node) bindingPattern() *node {
	index := 2
	if n.kind == nodeReduce {
		index = 3
	}
	if len(n.children) > index {
		return n.children[index]
	}
	return nil
}

// accumulator is a reduce stage's target and initial value, held as a default
func (n *node) accumulator() *node {
	return n.children[2]
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

func (c nameCollector) collectPattern(pattern *node, bound []string) []string {
	if pattern.kind == nodeDefault {
		c.collect(pattern.children[1], bound)
		return c.collectPattern(pattern.children[0], bound)
	}
	if pattern.kind == nodeName && pattern.text != "" {
		bound = append(bound, pattern.text)
	}
	for _, child := range pattern.children {
		bound = c.collectPattern(child, bound)
	}
	return bound
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
