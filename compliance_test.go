package argex_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"
)

type (
	complianceSuite struct {
		Tests []complianceCase `json:"tests"`
	}

	complianceCase struct {
		Name       string          `json:"name"`
		Expression string          `json:"expression"`
		Variables  map[string]any  `json:"variables"`
		This       json.RawMessage `json:"this"`
		Result     json.RawMessage `json:"result"`
		Missing    bool            `json:"missing"`
		Error      string          `json:"error"`
		Phase      string          `json:"phase"`
		this       any
		want       any
		wantErr    error
	}
)

func TestComplianceSuite(t *testing.T) {
	files, err := filepath.Glob("testdata/compliance/*.json")
	if !assert.NoError(t, err) || !assert.NotEmpty(t, files) {
		return
	}
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if !assert.NoError(t, err) {
				return
			}
			var suite complianceSuite
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if !assert.NoError(t, decoder.Decode(&suite)) {
				return
			}
			var extra any
			if !assert.ErrorIs(t, decoder.Decode(&extra), io.EOF) {
				return
			}
			if !assert.NotEmpty(t, suite.Tests) {
				return
			}
			seen := map[string]bool{}
			for _, tc := range suite.Tests {
				if !assert.NotEmpty(t, tc.Name) ||
					!assert.False(t, seen[tc.Name]) {
					return
				}
				seen[tc.Name] = true
				t.Run(tc.Name, func(t *testing.T) {
					t.Parallel()
					runCompliance(t, &tc)
				})
			}
		})
	}
}

func runCompliance(t *testing.T, tc *complianceCase) {
	t.Helper()
	expectations := 0
	if len(tc.Result) > 0 {
		expectations++
		if !assert.NoError(t, json.Unmarshal(tc.Result, &tc.want)) {
			return
		}
	}
	if tc.Missing {
		expectations++
	}
	if tc.Error != "" {
		expectations++
		tc.wantErr = complianceError(t, tc.Error)
	}
	if !assert.Equal(t, 1, expectations) {
		return
	}
	if tc.wantErr == nil && !assert.Empty(t, tc.Phase) {
		return
	}
	if tc.wantErr != nil &&
		!assert.Contains(t, []string{"parse", "compile", "eval"}, tc.Phase) {
		return
	}
	if !complianceThis(t, tc) {
		return
	}
	before, err := json.Marshal(tc.this)
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		after, err := json.Marshal(tc.this)
		assert.NoError(t, err)
		assert.Equal(t, before, after)
	}()
	value, err := argex.Eval(tc.Expression, tc.this)
	assertCompliance(t, tc, value, err)
	expr, err := argex.Parse(tc.Expression)
	if tc.Phase == "parse" {
		assert.ErrorIs(t, err, tc.wantErr)
		return
	}
	if !assert.NoError(t, err) {
		return
	}
	program, err := argex.DefaultEnv().Compile(expr)
	if tc.Phase == "compile" {
		assert.ErrorIs(t, err, tc.wantErr)
		return
	}
	if !assert.NoError(t, err) {
		return
	}
	for range 2 {
		value, err := program.Eval(tc.this)
		assertCompliance(t, tc, value, err)
	}
	second, err := argex.DefaultEnv().Compile(expr)
	if !assert.NoError(t, err) {
		return
	}
	value, err = second.Eval(tc.this)
	assertCompliance(t, tc, value, err)
}

// complianceThis uses a case's this when given, otherwise its variables
// object. A case may not specify both
func complianceThis(t *testing.T, tc *complianceCase) bool {
	t.Helper()
	if len(tc.This) == 0 {
		tc.this = tc.Variables
		return true
	}
	if !assert.Nil(t, tc.Variables) {
		return false
	}
	return assert.NoError(t, json.Unmarshal(tc.This, &tc.this))
}

func assertCompliance(t *testing.T, tc *complianceCase, value any, err error) {
	t.Helper()
	if tc.wantErr != nil {
		assert.ErrorIs(t, err, tc.wantErr)
		return
	}
	if !assert.NoError(t, err) {
		return
	}
	if tc.Missing {
		assert.True(t, argex.IsMissing(value))
		return
	}
	assert.Equal(t, tc.want, value)
}

func complianceError(t *testing.T, name string) error {
	t.Helper()
	switch name {
	case "syntax":
		return argex.ErrSyntax

	case "type":
		return argex.ErrType

	case "missing":
		return argex.ErrMissing

	case "arithmetic":
		return argex.ErrArithmetic

	case "function":
		return argex.ErrFunction

	case "limit":
		return argex.ErrLimit

	default:
		t.Fatalf("unknown compliance error: %q", name)
		return nil
	}
}
