package engine_test

import (
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	cases := map[string]error{
		"missing.field":               argex.ErrMissing,
		"{}.field":                    argex.ErrMissing,
		"[1][5]":                      argex.ErrMissing,
		"null.field":                  argex.ErrMissing,
		"1 + true":                    argex.ErrType,
		"true + 1":                    argex.ErrType,
		`"1" + 2`:                     argex.ErrType,
		`"1" - "2"`:                   argex.ErrType,
		"!1":                          argex.ErrType,
		"-true":                       argex.ErrType,
		"+true":                       argex.ErrType,
		"1 - true":                    argex.ErrType,
		"1 * true":                    argex.ErrType,
		"1 / true":                    argex.ErrType,
		"1 % true":                    argex.ErrType,
		"1 < true":                    argex.ErrType,
		"1 <= true":                   argex.ErrType,
		"1 > true":                    argex.ErrType,
		"1 >= true":                   argex.ErrType,
		`"1" < 2`:                     argex.ErrType,
		"1 && true":                   argex.ErrType,
		"false || 1":                  argex.ErrType,
		"true && 1":                   argex.ErrType,
		"case { 1: 2, 3 }":            argex.ErrType,
		"[1] | filter x: x":           argex.ErrType,
		"1 | map x: x":                argex.ErrType,
		"[1] | map x: x.field":        argex.ErrType,
		"1 / 0":                       argex.ErrArithmetic,
		"0 / 0":                       argex.ErrArithmetic,
		"1 % 0":                       argex.ErrArithmetic,
		"1e308 * 1e308":               argex.ErrArithmetic,
		"-1e308 - 1e308":              argex.ErrArithmetic,
		"[1][0.5]":                    argex.ErrType,
		`[1]["x"]`:                    argex.ErrType,
		"{}[0]":                       argex.ErrType,
		"true?.field":                 argex.ErrType,
		"customer?.absent.field":      argex.ErrMissing,
		"missing.field ?? 1":          argex.ErrMissing,
		"[1] | map x: other.field":    argex.ErrMissing,
		"[1] | filter x: other.field": argex.ErrMissing,
		"length(1)":                   argex.ErrType,
		"abs(true)":                   argex.ErrType,
		"min(1,true)":                 argex.ErrType,
		"min(true,1)":                 argex.ErrType,
		"max(1,true)":                 argex.ErrType,
		"max(true,1)":                 argex.ErrType,
		"abs()":                       argex.ErrFunction,
		`length("a","b")`:             argex.ErrFunction,
		"min()":                       argex.ErrFunction,
		"max()":                       argex.ErrFunction,
		"unknown(1)":                  argex.ErrFunction,
		"[1] | map x: unknown()":      argex.ErrFunction,
		`startsWith(1, "a")`:          argex.ErrType,
		`startsWith("a", 1)`:          argex.ErrType,
		`endsWith(1, "a")`:            argex.ErrType,
		`endsWith("a", 1)`:            argex.ErrType,
		`includes(1, "a")`:            argex.ErrType,
		`includes("a", 1)`:            argex.ErrType,
		"lower(1)":                    argex.ErrType,
		"upper(1)":                    argex.ErrType,
		"trim(1)":                     argex.ErrType,
	}
	for source, want := range cases {
		t.Run(source, func(t *testing.T) {
			_, err := argex.Eval(source, map[string]any{
				"customer": map[string]any{},
			})
			assert.ErrorIs(t, err, want)
		})
	}
	_, err := argex.Compile(nil)
	assert.ErrorIs(t, err, argex.ErrSyntax)
	_, err = argex.Compile(&argex.Expression{})
	assert.ErrorIs(t, err, argex.ErrSyntax)
}
