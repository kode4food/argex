package isa

type (
	// Opcode identifies an instruction's operation
	Opcode Word

	// Operand parameterizes an instruction
	Operand Word
)

const (
	// OpcodeSize is the number of opcode bits in an instruction
	OpcodeSize = 6

	// OpcodeMask selects an instruction's opcode bits
	OpcodeMask = Opcode(1<<OpcodeSize - 1)

	// OperandMask is the largest operand an instruction can hold
	OperandMask = ^Operand(0) >> OpcodeSize
)

const (
	// Stack, constants, globals, and locals
	Const  Opcode = iota // Push constant (operand = index)
	Global               // Push input variable (operand = name index)
	Load                 // Push local binding (operand = slot)
	Store                // Pop a value into a local binding (operand = slot)
	Zero                 // Set a local binding to zero (operand = slot)
	Inc                  // Add one to a numeric local binding (operand = slot)
	Pop                  // Discard the top of the stack

	// Property and index access
	Access         // Replace receiver with property (operand = name index)
	OptionalAccess // Access property, allowing absent receivers and members
	Index          // Pop index, replace receiver with indexed value
	OptionalIndex  // Index, allowing absent receivers and members
	Optional       // If receiver is absent, replace with missing and jump

	// Boolean operations and equality
	Not      // Negate Boolean value
	Eq       // Pop two values, push structural equality
	Neq      // Pop two values, push structural inequality
	In       // Pop value and array, push membership by structural equality
	And      // If false, jump. Otherwise pop left operand
	Or       // If true, jump. Otherwise pop left operand
	Coalesce // If present and non-null, jump. Otherwise pop left operand
	Boolean  // Check that the top of the stack is a Boolean

	// Numeric operations
	Pos // Convert operand to a number
	Neg // Negate number
	Add // Pop two values, push numeric sum or string concatenation
	Sub // Pop two numbers, push difference
	Mul // Pop two numbers, push product
	Div // Pop two numbers, push quotient
	Mod // Pop two numbers, push remainder
	Lt  // Pop two numbers or strings, push left < right
	Lte // Pop two numbers or strings, push left <= right
	Gt  // Pop two numbers or strings, push left > right
	Gte // Pop two numbers or strings, push left >= right

	// Function calls
	Call // Call a resolved function (operand = call specification index)

	// Control flow
	CondJump  // Pop condition, jump if false (operand = destination)
	Jump      // Jump unconditionally (operand = destination)
	JumpIf    // Pop condition, jump if true (operand = destination)
	JumpIfLt  // Pop two values, jump if left < right, comparing like Lt
	JumpIfLte // Pop two values, jump if left <= right, comparing like Lte
	JumpIfGt  // Pop two values, jump if left > right, comparing like Gt
	JumpIfGte // Pop two values, jump if left >= right, comparing like Gte

	// Construction and iteration
	Array   // Pop values, push array (operand = count)
	Object  // Pop values, push object (operand = property names index)
	Push    // Pop a value, append it to the array being built
	Put     // Pop a value and a string key, set it on the object being built
	Concat  // Pop an array, append its elements to the array being built
	Merge   // Pop an object, set its entries on the object being built
	Iterate // Replace a collection with an iterator (operand = iteration index)
	Fetch   // Bind the iterator's next element and jump to the loop top, if any
	Append  // Pop a value, append it to the iterator's result
	Result  // Replace the iterator with its accumulated result

	// Compilation only
	Label // Mark a jump destination, removed when compiling finishes
)

// New packs the opcode and operand into an instruction
func (o Opcode) New(operand Operand) Instruction {
	return Instruction(o) | Instruction(operand)<<OpcodeSize
}
