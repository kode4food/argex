package isa

type (
	// Effect describes how an instruction changes the operand stack and what
	// its operand refers to
	Effect struct {
		Operand ActOn // What the operand refers to
		Pop     int   // Values popped
		Push    int   // Values pushed
		DPop    bool  // Also pops a count the operand determines
		Keep    bool  // Jumping leaves the popped value on the stack
	}

	// ActOn identifies what an instruction's operand refers to
	ActOn int
)

const (
	Nothing   ActOn = iota
	Constants       // A constant, which sets the count for a DPop
	Count           // A count of values
	Labels          // A label until compiling finishes, then a destination
)

// Effects describes every opcode, indexed by opcode
var Effects = [...]Effect{
	Const:          {Push: 1},
	Global:         {Push: 1},
	Load:           {Push: 1},
	Store:          {Pop: 1},
	Zero:           {},
	Inc:            {},
	Pop:            {Pop: 1},
	Access:         {Pop: 1, Push: 1},
	OptionalAccess: {Pop: 1, Push: 1},
	Index:          {Pop: 2, Push: 1},
	OptionalIndex:  {Pop: 2, Push: 1},
	Optional:       {Operand: Labels},
	Not:            {Pop: 1, Push: 1},
	Eq:             {Pop: 2, Push: 1},
	Neq:            {Pop: 2, Push: 1},
	In:             {Pop: 2, Push: 1},
	And:            {Pop: 1, Operand: Labels, Keep: true},
	Or:             {Pop: 1, Operand: Labels, Keep: true},
	Coalesce:       {Pop: 1, Operand: Labels, Keep: true},
	Boolean:        {Pop: 1, Push: 1},
	Pos:            {Pop: 1, Push: 1},
	Neg:            {Pop: 1, Push: 1},
	Add:            {Pop: 2, Push: 1},
	Sub:            {Pop: 2, Push: 1},
	Mul:            {Pop: 2, Push: 1},
	Div:            {Pop: 2, Push: 1},
	Mod:            {Pop: 2, Push: 1},
	Lt:             {Pop: 2, Push: 1},
	Lte:            {Pop: 2, Push: 1},
	Gt:             {Pop: 2, Push: 1},
	Gte:            {Pop: 2, Push: 1},
	Call:           {DPop: true, Push: 1, Operand: Constants},
	CallDynamic:    {Pop: 1, DPop: true, Push: 1, Operand: Count},
	CallWith:       {Pop: 2, Push: 1},
	CondJump:       {Pop: 1, Operand: Labels},
	Jump:           {Operand: Labels},
	JumpIf:         {Pop: 1, Operand: Labels},
	JumpIfLt:       {Pop: 2, Operand: Labels},
	JumpIfLte:      {Pop: 2, Operand: Labels},
	JumpIfGt:       {Pop: 2, Operand: Labels},
	JumpIfGte:      {Pop: 2, Operand: Labels},
	Array:          {DPop: true, Push: 1, Operand: Count},
	Object:         {DPop: true, Push: 1, Operand: Constants},
	Push:           {Pop: 1},
	Put:            {Pop: 2},
	Concat:         {Pop: 1},
	Merge:          {Pop: 1},
	Iterate:        {Pop: 1, Push: 1},
	Fetch:          {},
	Append:         {Pop: 1},
	Result:         {Pop: 1, Push: 1},
	Label:          {Operand: Labels},
}
