package engine

import (
	"fmt"
	"strings"
)

type (
	// TypeTag is a single possible runtime type for an expression's result
	TypeTag uint16

	// TypeSet is the union of every TypeTag an expression might evaluate to
	TypeSet uint16
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
// evaluating it; reading runtime input widens the result to include TypeAny
func (e *Expression) StaticType() TypeSet {
	if e == nil || e.root == nil {
		return TypeSet(TypeAny)
	}
	return inferType(e.root)
}

// Check statically validates every operator and builtin call operand whose type
// is fully known without `this`, catching a mismatch that can never succeed
// (e.g. length(-1)) without evaluating anything
func (e *Expression) Check() error {
	if e == nil || e.root == nil {
		return nil
	}
	return checkNode(e.root)
}

func inferType(n *node) TypeSet {
	switch n.kind {
	case nodeLiteral:
		return literalType(n.value)

	case nodeName, nodeRoot:
		return TypeSet(TypeAny)

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

func inferBinaryType(n *node) TypeSet {
	switch n.text {
	case opEq, opNeq, opLt, opLte, opGt, opGte, opIn, opAnd, opOr:
		return TypeSet(TypeBool)

	case opCoalesce:
		left := inferType(n.children[0])
		left &^= TypeSet(TypeNull) | TypeSet(TypeMissing)
		return left | inferType(n.children[1])

	case opAdd:
		left := inferType(n.children[0])
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
		result |= inferType(n.children[i])
	}
	return result | inferType(n.children[last])
}

func checkNode(n *node) error {
	if err := checkNodeSelf(n); err != nil {
		return err
	}
	for _, child := range n.children {
		if err := checkNode(child); err != nil {
			return err
		}
	}
	return nil
}

func checkNodeSelf(n *node) error {
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
		return checkKeyType(n.children[0], "computed member")

	case nodeCase:
		return checkCase(n)

	case nodeAny, nodeAll, nodeFilter:
		return checkBoolBody(n)
	}
	return nil
}

// paramType returns the declared type for argument i of fn; a variadic
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
		what := fmt.Sprintf("%s argument %d", n.text, i+1)
		if err := requireType(child, expected, what); err != nil {
			return err
		}
	}
	return nil
}

// requireType fails when n's known type has no overlap with allowed
func requireType(n *node, allowed TypeSet, what string) error {
	actual := inferType(n)
	if !actual.Known() || actual&allowed != 0 {
		return nil
	}
	return fmt.Errorf(
		"%w: %s must be %s, not %s", ErrType, what, allowed, actual,
	)
}

func checkUnary(n *node) error {
	if n.text == opNot {
		return requireType(n.children[0], TypeSet(TypeBool), opNot+" operand")
	}
	return requireType(n.children[0], numericSet, n.text+" operand")
}

func checkBinary(n *node) error {
	left := n.children[0]
	right := n.children[1]
	switch n.text {
	case opAnd, opOr:
		what := n.text + " left operand"
		if err := requireType(left, TypeSet(TypeBool), what); err != nil {
			return err
		}
		what = n.text + " right operand"
		return requireType(right, TypeSet(TypeBool), what)

	case opIn:
		return requireType(right, TypeSet(TypeArray), opIn+" right operand")

	case opEq, opNeq, opCoalesce:
		return nil // structural equality and coalesce accept any type

	case opAdd:
		return checkAddOperands(left, right)

	case opSub, opMul, opDiv, opMod:
		what := n.text + " left operand"
		if err := requireType(left, numericSet, what); err != nil {
			return err
		}
		what = n.text + " right operand"
		return requireType(right, numericSet, what)

	default: // opLt, opLte, opGt, opGte
		return checkComparisonOperands(n.text, left, right)
	}
}

// checkAddOperands mirrors vm.go's Add: a string left operand requires a
// string right operand; anything else runs both through number()
func checkAddOperands(left, right *node) error {
	leftType := inferType(left)
	if !leftType.Known() {
		return nil
	}
	if leftType.Is(TypeString) {
		return requireType(right, TypeSet(TypeString), "+ right operand")
	}
	if leftType&numericSet == 0 {
		allowed := numericSet | TypeSet(TypeString)
		return fmt.Errorf(
			"%w: + left operand must be %s, not %s",
			ErrType, allowed, leftType,
		)
	}
	return requireType(right, numericSet, "+ right operand")
}

// checkComparisonOperands mirrors vm.go's Lt/Lte/Gt/Gte, the same dual
// string-or-number path as checkAddOperands without string concatenation
func checkComparisonOperands(op string, left, right *node) error {
	leftType := inferType(left)
	if !leftType.Known() {
		return nil
	}
	if leftType.Is(TypeString) {
		return requireType(right, TypeSet(TypeString), op+" right operand")
	}
	if leftType&numericSet == 0 {
		allowed := numericSet | TypeSet(TypeString)
		return fmt.Errorf(
			"%w: %s left operand must be %s, not %s",
			ErrType, op, allowed, leftType,
		)
	}
	return requireType(right, numericSet, op+" right operand")
}

func checkIndex(n *node) error {
	if err := checkReceiver(n, collectionSet); err != nil {
		return err
	}
	key := n.children[1]
	switch receiverType := effectiveReceiverType(n); {
	case receiverType.Is(TypeObject):
		return requireType(key, TypeSet(TypeString), "index key")

	case receiverType.Is(TypeArray):
		return requireType(key, numericSet, "index key")

	default:
		return nil
	}
}

// effectiveReceiverType removes the nullish portion for an optional
// access/index, since it short-circuits before the container check
func effectiveReceiverType(n *node) TypeSet {
	receiverType := inferType(n.children[0])
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
		return fmt.Errorf(
			"%w: access receiver must be %s, not %s",
			ErrType, allowed, receiverType,
		)
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
		if err := checkContainerSource(source, want, "spread"); err != nil {
			return err
		}
	}
	return nil
}

// checkContainerSource mirrors vm.go's Concat/Merge: a nullish source is
// skipped outright, so only its non-nullish portion is checked
func checkContainerSource(n *node, want TypeTag, what string) error {
	sourceType := inferType(n) &^ nullishSet
	if !sourceType.Known() || sourceType == 0 {
		return nil
	}
	if !sourceType.Has(want) {
		return fmt.Errorf(
			"%w: %s source must be %s, not %s",
			ErrType, what, TypeSet(want), sourceType,
		)
	}
	return nil
}

func checkKeyType(n *node, what string) error {
	return requireType(n, TypeSet(TypeString), what+" key")
}

// checkCase mirrors vm.go's CondJump: every arm's condition must be boolean
func checkCase(n *node) error {
	last := len(n.children) - 1
	for i := 0; i < last; i += 2 {
		condition := n.children[i]
		what := "case condition"
		if err := requireType(condition, TypeSet(TypeBool), what); err != nil {
			return err
		}
	}
	return nil
}

// checkBoolBody mirrors vm.go's CondJump: any/all/filter re-check their
// body as a condition on every iteration
func checkBoolBody(n *node) error {
	what := collectionNames[n.kind] + " body"
	return requireType(n.children[1], TypeSet(TypeBool), what)
}
