package engine

func (n *node) annotatePattern(source *node, scope []typedName, env *Env) error {
	n.inferred = TypeSet(TypeAny)
	if source != nil {
		n.inferred = source.inferred
	}
	switch n.kind {
	case nodeDefault:
		fallback := n.children[1]
		if err := annotateType(fallback, scope, env); err != nil {
			return err
		}
		if n.inferred.Is(TypeMissing) {
			source = fallback
		}
		target := n.children[0]
		if err := target.annotatePattern(source, scope, env); err != nil {
			return err
		}
		if !n.inferred.Known() || n.inferred.Has(TypeMissing) {
			target.inferred = n.inferred&^TypeSet(TypeMissing) |
				fallback.inferred
		}
		return nil

	case nodeField:
		return n.children[0].annotatePattern(source, scope, env)
	}
	if source != nil && countSpreads(source.children) > 0 {
		source = nil
	}
	scope = scope[:len(scope):len(scope)]
	for i, child := range n.children {
		member := source.patternMember(child, i)
		if err := child.annotatePattern(member, scope, env); err != nil {
			return err
		}
		if i+1 < len(n.children) {
			scope = patternScope(child, scope)
		}
	}
	return nil
}

func (n *node) patternMember(field *node, index int) *node {
	if n == nil {
		return nil
	}
	if n.kind == nodeArray && index < len(n.children) {
		return n.children[index]
	}
	if n.kind != nodeObject {
		return nil
	}
	var result *node
	for _, child := range n.children {
		if child.kind != nodeField {
			return nil
		}
		if child.text != field.text {
			continue
		}
		result = child.children[0]
	}
	return result
}

// annotateLetPattern expects the let value to be annotated already
func annotateLetPattern(n *node, scope []typedName, env *Env) error {
	pattern := n.bindingPattern()
	if err := checkReceiver(n, TypeSet(pattern.patternType())); err != nil {
		return err
	}
	inner := scope[:len(scope):len(scope)]
	if err := pattern.annotatePattern(n.children[0], inner, env); err != nil {
		return err
	}
	return annotateType(n.children[1], patternScope(pattern, inner), env)
}

func annotateStagePattern(n *node, scope []typedName, env *Env) error {
	inner := scope[:len(scope):len(scope)]
	for _, name := range stageOuterNames(n) {
		bound := typedName{name: name, inferred: TypeSet(TypeAny)}
		inner = append(inner, bound)
	}
	pattern := n.bindingPattern()
	if err := pattern.annotatePattern(nil, inner, env); err != nil {
		return err
	}
	return annotateType(n.children[1], patternScope(pattern, inner), env)
}

// annotateAccumulator types the initial value in the enclosing scope. The accumulator changes per
// element, so its bindings are untyped
func annotateAccumulator(n *node, scope []typedName, env *Env) error {
	if n.kind != nodeReduce {
		return nil
	}
	accumulator := n.accumulator()
	if err := annotateType(accumulator.children[1], scope, env); err != nil {
		return err
	}
	inner := scope[:len(scope):len(scope)]
	if n.chained {
		for _, name := range stageNames(n.children[0]) {
			bound := typedName{name: name, inferred: TypeSet(TypeAny)}
			inner = append(inner, bound)
		}
	}
	return accumulator.children[0].annotatePattern(nil, inner, env)
}

// patternScope builds scope outside recursive inference so ordinary let and pipeline scopes can
// stay on the stack
func patternScope(pattern *node, scope []typedName) []typedName {
	if pattern.kind == nodeDefault {
		return patternScope(pattern.children[0], scope)
	}
	if pattern.kind == nodeName && pattern.text != "" {
		bound := typedName{name: pattern.text, inferred: TypeSet(TypeAny)}
		scope = append(scope, bound)
	}
	for _, child := range pattern.children {
		scope = patternScope(child, scope)
	}
	return scope
}
