package isa

type (
	// Word is the native-sized unit of the instruction stream
	Word uintptr

	// Instruction packs an opcode and its operand into one word
	Instruction Word

	// Instructions is an executable instruction stream
	Instructions []Instruction
)

// Opcode returns the instruction's operation
func (i Instruction) Opcode() Opcode {
	return Opcode(i) & OpcodeMask
}

// Operand returns the instruction's parameter
func (i Instruction) Operand() Operand {
	return Operand(i) >> OpcodeSize
}
