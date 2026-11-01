package engine_test

import (
	"strings"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

const caseRetryDepth = 12

func TestPatternSyntax(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{name: "duplicate array name", source: "let [x, x] = []: 1"},
		{name: "duplicate alias", source: "let {x, y: x} = {}: 1"},
		{name: "duplicate key", source: "let {x, x: y} = {}: 1"},
		{name: "reserved array name", source: "let [true] = []: 1"},
		{name: "reserved shorthand", source: "let {null} = {}: 1"},
		{name: "string shorthand", source: `let {"x"} = {}: 1`},
		{name: "missing target", source: "let {x:} = {}: 1"},
		{name: "missing array default", source: "let [x = ] = []: 1"},
		{name: "array rest", source: "let [...x] = []: 1"},
		{name: "computed key", source: "let {[x]: y} = {}: 1"},
		{name: "unclosed array", source: "let [x = []: x"},
		{name: "unclosed object", source: "let {x = {}: x"},
		{name: "literal target", source: "let 1 = []: 1"},
		{name: "duplicate stage name", source: "[] | map i, {x, x}: x"},
		{name: "reserved stage name", source: "[] | map [x, true]: x"},
		{name: "mismatched brackets", source: "[] | map [x, {y]]: x"},
		{name: "missing stage target", source: "[] | map {x:}: x"},
		{name: "missing object default", source: "let {x = } = {}: x"},
		{name: "duplicate default name", source: "let {x = 1, x} = {}: x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := argex.Parse(tc.source)
			assert.ErrorIs(t, err, argex.ErrSyntax)
		})
	}
	_, err := argex.Parse(
		"let " + strings.Repeat("[", expressionDepth) + "x" +
			strings.Repeat("]", expressionDepth) + " = []: x",
	)
	assert.ErrorIs(t, err, argex.ErrLimit)
	_, err = argex.Parse(
		strings.Repeat("case { [0] | map {x}: ", caseRetryDepth) +
			"case {}" + strings.Repeat(", false }", caseRetryDepth),
	)
	assert.ErrorIs(t, err, argex.ErrLimit)
}

func TestPatternColonContexts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   any
	}{
		{
			name:   "object literal",
			source: "case [{x: 1}] { [0] | map {x: 1}: true, false }",
			want:   true,
		},
		{
			name: "pattern shaped object literal",
			source: "let y = 1: " +
				"case [{x: 1}] { [0] | map {x: y}: true, false }",
			want: true,
		},
		{
			name:   "array literal",
			source: "case [[1]] { [0] | map [1]: true, false }",
			want:   true,
		},
		{
			name: "pattern shaped array literal",
			source: "let x = 1: " +
				"case [[1]] { [0] | map [x]: true, false }",
			want: true,
		},
		{
			name: "several arms",
			source: "case [{x: 1}] { [0] | map {x: 0}: false, " +
				"[0] | map {x: 1}: true, false }",
			want: true,
		},
		{
			name: "pipeline arm value",
			source: "case [{x: 1}] { " +
				"[0] | map {x: 1}: [2] | map x: x, [] }",
			want: []any{float64(2)},
		},
		{
			name: "condition pattern",
			source: "case { " +
				"[{x: 1}] | any {x}: x == 1: true, false }",
			want: true,
		},
		{
			name: "fallback pattern",
			source: "case { false: [], " +
				"[{x: 1}] | map {x}: x }",
			want: []any{float64(1)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := argex.Eval(tc.source, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestPatternNames(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "nested sequential bindings",
			source: "let {x, y: [z]} = source, [n] = z: x + n + extra",
			want:   []string{"extra", "source"},
		},
		{
			name:   "source scope",
			source: "let {x} = x: x",
			want:   []string{"x"},
		},
		{
			name:   "discard",
			source: "let {x: _} = source: _",
			want:   []string{"_", "source"},
		},
		{
			name:   "stage scope",
			source: "items | map {x}: x | map x",
			want:   []string{"items"},
		},
		{
			name:   "key scope",
			source: "items | map key, [x]: key + x + extra",
			want:   []string{"extra", "items"},
		},
		{
			name:   "grouped stage",
			source: "(items | map {x}: x) | map x",
			want:   []string{"items", "x"},
		},
		{
			name:   "shadow builtin",
			source: "items | map {length}: length",
			want:   []string{"items"},
		},
		{
			name:   "default scope",
			source: "let {x = fallback, y = x} = source: y",
			want:   []string{"fallback", "source"},
		},
		{
			name:   "future name",
			source: "let {x = y, y = 1} = source: x",
			want:   []string{"source", "y"},
		},
		{
			name:   "default key scope",
			source: "items | map i, {x = i, y = x + offset}: y",
			want:   []string{"items", "offset"},
		},
		{
			name:   "default stage scope",
			source: "items | map {x}: {} | map {y = x}: y",
			want:   []string{"items"},
		},
		{
			name:   "nested default scope",
			source: "let {x = (let y = fallback: y)} = source: x",
			want:   []string{"fallback", "source"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := argex.Parse(tc.source)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, tc.want, argex.DefaultEnv().Names(expr))
		})
	}
}

func TestPatternTypes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   error
	}{
		{
			name:   "array source",
			source: "let [] = 1: 0",
			want:   argex.ErrType,
		},
		{
			name:   "object source",
			source: "let {} = []: 0",
			want:   argex.ErrType,
		},
		{
			name:   "numeric object key",
			source: `let [x] = {"0": 1}: x`,
			want:   argex.ErrType,
		},
		{
			name:   "null array source",
			source: "let [] = null: 0",
			want:   argex.ErrMissing,
		},
		{
			name:   "null object source",
			source: "let {} = null: 0",
			want:   argex.ErrMissing,
		},
		{
			name:   "null source with default",
			source: "let {x = 1} = null: x",
			want:   argex.ErrMissing,
		},
		{
			name:   "null source with fallback",
			source: "let {x} = null ?? {x: 7}: x",
			want:   argex.ErrMissing,
		},
		{
			name:   "default type",
			source: "let {x = true + 1} = {x: 3}: x",
			want:   argex.ErrType,
		},
		{
			name:   "unknown function",
			source: "let {x = unknown()} = {x: 3}: x",
			want:   argex.ErrFunction,
		},
		{
			name:   "nested unknown function",
			source: "let {x: [y = unknown()]} = {x: [1]}: y",
			want:   argex.ErrFunction,
		},
		{
			name:   "stage unknown function",
			source: "[] | map {x = unknown()}: x",
			want:   argex.ErrFunction,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := argex.Parse(tc.source)
			if !assert.NoError(t, err) {
				return
			}
			_, err = argex.DefaultEnv().Compile(expr)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
