package isa_test

import (
	"testing"

	"github.com/kode4food/argex/internal/isa"
	"github.com/stretchr/testify/assert"
)

func TestInstruction(t *testing.T) {
	operands := []isa.Operand{
		0, 1, isa.Operand(isa.OpcodeMask),
		isa.Operand(isa.OpcodeMask) + 1, isa.OperandMask,
	}
	for opcode := isa.Const; opcode <= isa.Label; opcode++ {
		for _, operand := range operands {
			instruction := opcode.New(operand)
			assert.Equal(t, opcode, instruction.Opcode())
			assert.Equal(t, operand, instruction.Operand())
		}
	}
}

func TestEffects(t *testing.T) {
	assert.Len(t, isa.Effects, int(isa.Label)+1)
}
