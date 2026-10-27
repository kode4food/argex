package app_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argex"
	app "github.com/kode4food/argex/cmd/argex/internal"
)

type (
	runCase struct {
		name  string
		args  []string
		input string
		want  string
	}

	errorCase struct {
		name  string
		args  []string
		input string
		want  error
	}
)

func TestRun(t *testing.T) {
	cases := []runCase{
		{
			name:  "pretty object",
			args:  []string{"{id: this.id}"},
			input: `{"id":1}`,
			want:  "{\n  \"id\": 1\n}\n",
		},
		{
			name:  "compact",
			args:  []string{"-c", "this | map x: x * 2"},
			input: `[1,2]`,
			want:  "[2,4]\n",
		},
		{
			name:  "raw string",
			args:  []string{"-r", "name"},
			input: `{"name":"Ada"}`,
			want:  "Ada\n",
		},
		{
			name:  "raw leaves non-strings as JSON",
			args:  []string{"-r", "-c", "this"},
			input: `[1]`,
			want:  "[1]\n",
		},
		{
			name: "null input",
			args: []string{"-n", "this ?? 3"},
			want: "3\n",
		},
		{
			name:  "value stream",
			args:  []string{"-c", "this.a"},
			input: `{"a":1} {"a":2}`,
			want:  "1\n2\n",
		},
		{
			name:  "missing prints nothing",
			args:  []string{"absent"},
			input: `{}`,
			want:  "",
		},
		{
			name:  "number text preserved",
			args:  []string{"-c", "this"},
			input: `{"p":1.10,"big":12345678901234567890}`,
			want:  "{\"big\":12345678901234567890,\"p\":1.10}\n",
		},
		{
			name:  "numbers compute",
			args:  []string{"p * 2"},
			input: `{"p":1.5}`,
			want:  "3\n",
		},
		{
			name:  "html not escaped",
			args:  []string{"a"},
			input: `{"a":"<b>"}`,
			want:  "\"<b>\"\n",
		},
		{
			name:  "empty input",
			args:  []string{"this"},
			input: "",
			want:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := app.Run(tc.args, strings.NewReader(tc.input), &out)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, out.String())
		})
	}
}

func TestRunFiles(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")
	assert.NoError(t, os.WriteFile(first, []byte("1"), 0o600))
	assert.NoError(t, os.WriteFile(second, []byte("2"), 0o600))
	var out bytes.Buffer
	err := app.Run(
		[]string{"this * 10", first, second}, strings.NewReader("99"), &out,
	)
	assert.NoError(t, err)
	assert.Equal(t, "10\n20\n", out.String())
}

func TestRunHelp(t *testing.T) {
	var out bytes.Buffer
	err := app.Run([]string{"-h"}, strings.NewReader(""), &out)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "usage: argex")
	assert.Contains(t, out.String(), "-c")
}

func TestRunErrors(t *testing.T) {
	cases := []errorCase{
		{name: "no expression", want: app.ErrUsage},
		{
			name: "unknown flag",
			args: []string{"-x", "this"},
			want: app.ErrUsage,
		},
		{name: "syntax", args: []string{"1 +"}, want: argex.ErrSyntax},
		{
			name: "unknown func",
			args: []string{"nope()"},
			want: argex.ErrFunction,
		},
		{
			name:  "runtime",
			args:  []string{"a + 1"},
			input: `{"a":"x"}`,
			want:  argex.ErrType,
		},
		{
			name:  "invalid json",
			args:  []string{"this"},
			input: "{",
			want:  app.ErrInput,
		},
		{
			name:  "missing in json",
			args:  []string{"[absent]"},
			input: "{}",
			want:  argex.ErrMissing,
		},
		{
			name: "missing file",
			args: []string{"this", "/nonexistent/argex.json"},
			want: os.ErrNotExist,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := app.Run(tc.args, strings.NewReader(tc.input), &out)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
