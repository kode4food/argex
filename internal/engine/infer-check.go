package engine

import "fmt"

// operand names a checked operand, rendered only when its check fails
type operand struct {
	owner string
	role  string
	index int // 1-based argument position, or 0 for other operands
}

var (
	numericSet    = TypeSet(TypeNumber)
	collectionSet = TypeSet(TypeArray) | TypeSet(TypeObject)
	nullishSet    = TypeSet(TypeNull) | TypeSet(TypeMissing)
)

func (o operand) String() string {
	if o.index > 0 {
		return fmt.Sprintf("%s %s %d", o.owner, o.role, o.index)
	}
	return o.owner + " " + o.role
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

// checkReceiver mirrors vm.go/accessChild: a receiver known to be always null
// fails with ErrMissing, not ErrType
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

// checkSpreads validates every "...expr" entry directly inside an array (want
// TypeArray) or object (want TypeObject) literal
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

// checkBoolBody mirrors vm.go's CondJump: any/all/filter re-check their body as
// a condition on every iteration
func checkBoolBody(n *node) error {
	what := operand{owner: collectionNames[n.kind], role: "body"}
	return requireType(n.children[1], TypeSet(TypeBool), what)
}
