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
