package engine

import (
	"fmt"
	"slices"
	"strings"
)

type (
	// TypeTag is a single possible runtime type for an expression's result
	TypeTag uint16

	// TypeSet is the union of every TypeTag an expression might evaluate to
	TypeSet uint16

	// operand names a checked operand, rendered only when its check fails
	operand struct {
		owner string
		role  string
		index int // 1-based argument position, or 0 for other operands
	}

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
	{TypeAny, "any"},
}

var (
	numericSet    = TypeSet(TypeNumber)
	collectionSet = TypeSet(TypeArray) | TypeSet(TypeObject)
	nullishSet    = TypeSet(TypeNull) | TypeSet(TypeMissing)
)

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

// StaticType infers the set of types this expression's result can take without
// evaluating it. Reading runtime input widens the result to include TypeAny
func (e *Expression) StaticType() TypeSet {
	if e == nil || e.root == nil {
		return TypeSet(TypeAny)
	}
	return e.root.inferred
}

func (o operand) String() string {
	if o.index > 0 {
		return fmt.Sprintf("%s %s %d", o.owner, o.role, o.index)
	}
	return o.owner + " " + o.role
}

// annotateType infers each node's type bottom-up and rejects any operand whose
// known type can never succeed. A let name carries its value's type
func annotateType(n *node, scope []typedName) error {
	if err := annotateChildren(n, scope); err != nil {
		return err
	}
	n.inferred = computeType(n, scope)
	return checkNode(n)
}

// annotateChildren scopes bindings as collectNames does
func annotateChildren(n *node, scope []typedName) error {
	switch n.kind {
	case nodeMap, nodeFilter, nodeAny, nodeAll:
		if err := annotateType(n.children[0], scope); err != nil {
			return err
		}
		inner := scope[:len(scope):len(scope)]
		for _, name := range stageNames(n) {
			unknown := typedName{name: name, inferred: TypeSet(TypeAny)}
			inner = append(inner, unknown)
		}
		return annotateType(n.children[1], inner)

	case nodeLet:
		value := n.children[0]
		if err := annotateType(value, scope); err != nil {
			return err
		}
		bound := typedName{name: n.valueBinding, inferred: value.inferred}
		inner := append(scope[:len(scope):len(scope)], bound)
		return annotateType(n.children[1], inner)
	}
	for _, child := range n.children {
		if err := annotateType(child, scope); err != nil {
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
		if fn, ok := builtins[n.text]; ok && fn.result != 0 {
			return TypeSet(fn.result)
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

	case bool:
		return TypeSet(TypeBool)

	case float64:
		return TypeSet(TypeNumber)

	case string:
		return TypeSet(TypeString)

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

func checkNode(n *node) error {
	switch n.kind {
	case nodeCall:
		if fn, ok := builtins[n.text]; ok {
			return checkCallArgs(n, fn)
		}

	case nodeUnary:
		return checkUnary(n)

	case nodeBinary:
		return checkBinary(n)

	case nodeAccess:
		return checkReceiver(n, TypeSet(TypeObject))

	case nodeIndex:
		return checkIndex(n)

	case nodeArray:
		return checkSpreads(n, TypeArray)

	case nodeObject:
		return checkSpreads(n, TypeObject)

	case nodeComputed:
		what := operand{owner: "computed member", role: "key"}
		return requireType(n.children[0], TypeSet(TypeString), what)

	case nodeCase:
		return checkCase(n)

	case nodeLet:
		if n.valueBinding == caseSubject {
			return checkMatch(n)
		}

	case nodeAny, nodeAll, nodeFilter:
		return checkBoolBody(n)
	}
	return nil
}

// paramType returns the declared type for argument i of fn. A variadic
// function's last declared entry also covers every argument past it
func paramType(fn builtin, i int) TypeSet {
	if i < len(fn.params) {
		return fn.params[i]
	}
	return fn.params[len(fn.params)-1]
}

func checkCallArgs(n *node, fn builtin) error {
	for i, child := range n.children {
		expected := paramType(fn, i)
		what := operand{owner: n.text, role: "argument", index: i + 1}
		if err := requireType(child, expected, what); err != nil {
			return err
		}
	}
	return nil
}

// requireType fails when n's known type has no overlap with allowed
func requireType(n *node, allowed TypeSet, what operand) error {
	actual := n.inferred
	if !actual.Known() || actual&allowed != 0 {
		return nil
	}
	return fmt.Errorf("%w: %s must be %s, not %s",
		ErrType, what, allowed, actual)
}

func checkUnary(n *node) error {
	what := operand{owner: n.text, role: "operand"}
	if n.text == opNot {
		return requireType(n.children[0], TypeSet(TypeBool), what)
	}
	return requireType(n.children[0], numericSet, what)
}

func checkBinary(n *node) error {
	left := n.children[0]
	right := n.children[1]
	switch n.text {
	case opAnd, opOr:
		what := operand{owner: n.text, role: "left operand"}
		if err := requireType(left, TypeSet(TypeBool), what); err != nil {
			return err
		}
		what.role = "right operand"
		return requireType(right, TypeSet(TypeBool), what)

	case opIn:
		what := operand{owner: opIn, role: "right operand"}
		return requireType(right, TypeSet(TypeArray), what)

	case opEq, opNeq, opCoalesce:
		return nil // structural equality and coalesce accept any type

	case opSub, opMul, opDiv, opMod:
		what := operand{owner: n.text, role: "left operand"}
		if err := requireType(left, numericSet, what); err != nil {
			return err
		}
		what.role = "right operand"
		return requireType(right, numericSet, what)

	default: // opAdd, opLt, opLte, opGt, opGte
		return checkStringOrNumber(n.text, left, right)
	}
}

// checkStringOrNumber mirrors vm.go's Add/Lt/Lte/Gt/Gte. A string left operand
// requires a string right operand. Anything else runs both through number()
func checkStringOrNumber(op string, left, right *node) error {
	leftType := left.inferred
	if !leftType.Known() {
		return nil
	}
	what := operand{owner: op, role: "right operand"}
	if leftType.Is(TypeString) {
		return requireType(right, TypeSet(TypeString), what)
	}
	if leftType&numericSet == 0 {
		allowed := numericSet | TypeSet(TypeString)
		return fmt.Errorf("%w: %s left operand must be %s, not %s",
			ErrType, op, allowed, leftType)
	}
	return requireType(right, numericSet, what)
}

func checkIndex(n *node) error {
	if err := checkReceiver(n, collectionSet); err != nil {
		return err
	}
	key := n.children[1]
	what := operand{owner: "index", role: "key"}
	switch receiverType := effectiveReceiverType(n); {
	case receiverType.Is(TypeObject):
		return requireType(key, TypeSet(TypeString), what)

	case receiverType.Is(TypeArray):
		return requireType(key, numericSet, what)

	default:
		return nil
	}
}

// effectiveReceiverType removes the nullish portion for an optional
// access/index, since it short-circuits before the container check
func effectiveReceiverType(n *node) TypeSet {
	receiverType := n.children[0].inferred
	if n.optional {
		receiverType &^= nullishSet
	}
	return receiverType
}

// checkReceiver mirrors vm.go/accessChild: a receiver known to be always
// null fails with ErrMissing, not ErrType
func checkReceiver(n *node, allowed TypeSet) error {
	receiverType := effectiveReceiverType(n)
	if !receiverType.Known() || receiverType == 0 {
		return nil
	}
	if receiverType.Is(TypeNull) {
		return fmt.Errorf("%w: access receiver is always null", ErrMissing)
	}
	if receiverType&allowed == 0 {
		return fmt.Errorf("%w: access receiver must be %s, not %s",
			ErrType, allowed, receiverType)
	}
	return nil
}

// checkSpreads validates every "...expr" entry directly inside an array
// (want TypeArray) or object (want TypeObject) literal
func checkSpreads(n *node, want TypeTag) error {
	for _, child := range n.children {
		if child.kind != nodeSpread {
			continue
		}
		source := child.children[0]
		if err := checkContainerSource(source, want); err != nil {
			return err
		}
	}
	return nil
}

// checkContainerSource mirrors vm.go's Concat/Merge: a nullish source is
// skipped outright, so only its non-nullish portion is checked
func checkContainerSource(n *node, want TypeTag) error {
	sourceType := n.inferred &^ nullishSet
	if !sourceType.Known() || sourceType == 0 {
		return nil
	}
	if !sourceType.Has(want) {
		return fmt.Errorf("%w: spread source must be %s, not %s",
			ErrType, TypeSet(want), sourceType,
		)
	}
	return nil
}

// checkCase mirrors vm.go's CondJump: every arm's condition must be boolean
func checkCase(n *node) error {
	last := len(n.children) - 1
	what := operand{owner: "case", role: "condition"}
	for i := 0; i < last; i += 2 {
		condition := n.children[i]
		if err := requireType(condition, TypeSet(TypeBool), what); err != nil {
			return err
		}
	}
	return nil
}

// checkMatch mirrors vm.go's Eq: values of different types are never equal, so
// an arm whose value can't share a type with the subject never matches
func checkMatch(n *node) error {
	subject := n.children[0].inferred
	if !subject.Known() {
		return nil
	}
	arms := n.children[1].children
	what := operand{owner: "case", role: "value"}
	for i := 0; i < len(arms)-1; i += 2 {
		value := arms[i].children[1]
		if err := requireType(value, subject, what); err != nil {
			return err
		}
	}
	return nil
}

// checkBoolBody mirrors vm.go's CondJump: any/all/filter re-check their
// body as a condition on every iteration
func checkBoolBody(n *node) error {
	what := operand{owner: collectionNames[n.kind], role: "body"}
	return requireType(n.children[1], TypeSet(TypeBool), what)
}
