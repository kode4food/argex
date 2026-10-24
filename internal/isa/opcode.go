package isa

type (
	// Opcode identifies an instruction's operation
	Opcode Word

	// Operand parameterizes an instruction
	Operand Word
)

const (
	// OpcodeSize is the number of opcode bits in an instruction
	OpcodeSize = 7

	// OpcodeMask selects an instruction's opcode bits
	OpcodeMask = Opcode(1<<OpcodeSize - 1)

	// OperandMask is the largest operand an instruction can hold
	OperandMask = ^Operand(0) >> OpcodeSize
)

const (
	// Constants, globals, and locals
	Const  Opcode = iota // Push constant (operand = index)
	Global               // Push input variable (operand = name index)
	Load                 // Push local binding (operand = slot)

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
	And      // If false, jump; otherwise pop left operand
	Or       // If true, jump; otherwise pop left operand
	Coalesce // If present and non-null, jump; otherwise pop left operand
	Boolean  // Check Boolean operand (operand = error context index)

	// Numeric operations
	Pos    // Convert operand to a number
	Neg    // Negate number
	Add    // Pop two values, push numeric sum or string concatenation
	Sub    // Pop two numbers, push difference
	Mul    // Pop two numbers, push product
	Div    // Pop two numbers, push quotient
	Mod    // Pop two numbers, push remainder
	Lt     // Pop two numbers or strings, push left < right
	Lte    // Pop two numbers or strings, push left <= right
	Gt     // Pop two numbers or strings, push left > right
	Gte    // Pop two numbers or strings, push left >= right
	Length // Replace string, array, or object with its length
	Abs    // Replace number with its absolute value
	Min    // Pop numbers, push minimum (operand = count)
	Max    // Pop numbers, push maximum (operand = count)

	// Control flow
	CondJump // Pop condition, jump if false (operand = destination)
	Jump     // Jump unconditionally (operand = destination)

	// Construction and iteration
	Array  // Pop values, push array (operand = count)
	Object // Pop values, push object (operand = property names index)
	Map    // Start mapping an array (operand = iteration index)
	Filter // Start filtering an array (operand = iteration index)
	Next   // Collect body result and advance or finish iteration
)

// New packs the opcode and operand into an instruction
func (o Opcode) New(operand Operand) Instruction {
	return Instruction(o) | Instruction(operand)<<OpcodeSize
}
