package engine

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type (
	// TypeTag is a single possible runtime type for an expression's result
	TypeTag uint16

	// TypeSet is the union of every TypeTag an expression might evaluate to
	TypeSet uint16

	// typedName is a let or stage binding visible to the node being annotated
	typedName struct {
		name     string
		inferred TypeSet
	}
)

const (
	TypeBool TypeTag = 1 << iota
	TypeNumber
	TypeString
	TypeArray
	TypeObject
	TypeNull
	TypeMissing
	TypeFunction
	TypeAny
)

var typeTagNames = []struct {
	tag  TypeTag
	name string
}{
	{TypeBool, "bool"},
	{TypeNumber, "number"},
	{TypeString, "string"},
	{TypeArray, "array"},
	{TypeObject, "object"},
	{TypeNull, "null"},
	{TypeMissing, "missing"},
	{TypeFunction, "function"},
	{TypeAny, "any"},
}

// Has reports whether t is one of the types in the set
func (s TypeSet) Has(t TypeTag) bool {
	return TypeSet(t)&s != 0
}

// Is reports whether the set is exactly t, with no other possibilities
func (s TypeSet) Is(t TypeTag) bool {
	return s == TypeSet(t)
}

// Known reports whether every possibility in the set is statically known
func (s TypeSet) Known() bool {
	return !s.Has(TypeAny)
}

// String renders the set as its member type names joined with "|"
func (s TypeSet) String() string {
	var names []string
	for _, entry := range typeTagNames {
		if s.Has(entry.tag) {
			names = append(names, entry.name)
		}
	}
	return strings.Join(names, "|")
}

// StaticType returns the result types inferred during compilation
func (p *Program) StaticType() TypeSet {
	return p.resultType
}

// resolve annotates a copy, keeping the expression reusable across environments
func (e *Expression) resolve(env *Env) (*node, error) {
	if env == nil {
		return nil, fmt.Errorf("%w: no environment", ErrFunction)
	}
	root := e.root.clone()
	lowerNode(root)
	if err := annotateType(root, nil, env); err != nil {
		return nil, err
	}
	return root, nil
}

// annotateType resolves globals, infers each node's type bottom-up, and rejects
// any operand whose known type can never succeed
func annotateType(n *node, scope []typedName, env *Env) error {
	if err := annotateChildren(n, scope, env); err != nil {
		return err
	}
	if value, ok := resolveGlobal(n, scope, env); ok {
		n.kind = nodeLiteral
		n.value = value
	}
	n.inferred = computeType(n, scope)
	return checkNode(n)
}

// annotateChildren scopes bindings as collectNames does
func annotateChildren(n *node, scope []typedName, env *Env) error {
	switch n.kind {
	case nodeMap, nodeFilter, nodeAny, nodeAll:
		if err := annotateType(n.children[0], scope, env); err != nil {
			return err
		}
		inner := scope[:len(scope):len(scope)]
		for _, name := range stageNames(n) {
			unknown := typedName{name: name, inferred: TypeSet(TypeAny)}
			inner = append(inner, unknown)
		}
		return annotateType(n.children[1], inner, env)

	case nodeLet:
		value := n.children[0]
		if err := annotateType(value, scope, env); err != nil {
			return err
		}
		bound := typedName{name: n.valueBinding, inferred: value.inferred}
		inner := append(scope[:len(scope):len(scope)], bound)
		return annotateType(n.children[1], inner, env)
	}
	for _, child := range n.children {
		if err := annotateType(child, scope, env); err != nil {
			return err
		}
	}
	return nil
}

func computeType(n *node, scope []typedName) TypeSet {
	switch n.kind {
	case nodeLiteral:
		return literalType(n.value)

	case nodeName:
		return nameType(n.text, scope)

	case nodeRoot:
		return TypeSet(TypeAny)

	case nodeLet:
		return n.children[1].inferred

	case nodeAccess, nodeIndex:
		if n.optional {
			return TypeSet(TypeAny) | TypeSet(TypeMissing)
		}
		return TypeSet(TypeAny)

	case nodeCall:
		callee := n.children[0]
		if fn, ok := callee.value.(*Function); ok && fn.Result != 0 {
			return TypeSet(fn.Result)
		}
		return TypeSet(TypeAny)

	case nodeUnary:
		if n.text == opNot {
			return TypeSet(TypeBool)
		}
		return TypeSet(TypeNumber)

	case nodeBinary:
		return inferBinaryType(n)

	case nodeArray, nodeMap, nodeFilter:
		return TypeSet(TypeArray)

	case nodeObject:
		return TypeSet(TypeObject)

	case nodeAny, nodeAll:
		return TypeSet(TypeBool)

	case nodeCase:
		return inferCaseType(n)

	default:
		return TypeSet(TypeAny)
	}
}

func literalType(value any) TypeSet {
	switch value.(type) {
	case nil:
		return TypeSet(TypeNull)

	case Missing:
		return TypeSet(TypeMissing)

	case bool:
		return TypeSet(TypeBool)

	case float64, int, json.Number:
		return TypeSet(TypeNumber)

	case string:
		return TypeSet(TypeString)

	case []any:
		return TypeSet(TypeArray)

	case map[string]any:
		return TypeSet(TypeObject)

	case *Function:
		return TypeSet(TypeFunction)

	default:
		return TypeSet(TypeAny)
	}
}

func nameType(name string, scope []typedName) TypeSet {
	for _, bound := range slices.Backward(scope) {
		if bound.name == name {
			return bound.inferred
		}
	}
	return TypeSet(TypeAny)
}

func resolveGlobal(n *node, scope []typedName, env *Env) (any, bool) {
	if n.kind != nodeName {
		return nil, false
	}
	for _, bound := range scope {
		if bound.name == n.text {
			return nil, false
		}
	}
	return env.resolve(n.text)
}

func inferBinaryType(n *node) TypeSet {
	switch n.text {
	case opEq, opNeq, opLt, opLte, opGt, opGte, opIn, opAnd, opOr:
		return TypeSet(TypeBool)

	case opCoalesce:
		left := n.children[0].inferred
		left &^= TypeSet(TypeNull) | TypeSet(TypeMissing)
		return left | n.children[1].inferred

	case opAdd:
		left := n.children[0].inferred
		switch {
		case left.Is(TypeString):
			return TypeSet(TypeString)

		case left.Is(TypeNumber):
			return TypeSet(TypeNumber)

		default:
			return TypeSet(TypeNumber) | TypeSet(TypeString)
		}

	default: // opSub, opMul, opDiv, opMod
		return TypeSet(TypeNumber)
	}
}

func inferCaseType(n *node) TypeSet {
	var result TypeSet
	last := len(n.children) - 1
	for i := 1; i < last; i += 2 {
		result |= n.children[i].inferred
	}
	return result | n.children[last].inferred
}
