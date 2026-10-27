package app

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kode4food/argex"
)

// runner evaluates one compiled program against each input value and writes the
// results
type runner struct {
	program argex.Program
	encoder *json.Encoder
	out     io.Writer
	raw     bool
}

const usage = "usage: argex [-c] [-r] [-n] expression [file ...]"

var (
	ErrUsage = errors.New(usage)
	ErrInput = errors.New("invalid JSON input")
)

// Run evaluates the expression in args against each JSON value read from the
// named files, or from stdin when none are given, writing results to stdout
func Run(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("argex", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	compact := flags.Bool("c", false, "compact output")
	raw := flags.Bool("r", false, "print string results without quotes")
	null := flags.Bool("n", false, "evaluate once with null, reading no input")
	err := flags.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		flags.SetOutput(stdout)
		_, _ = fmt.Fprintln(stdout, usage)
		flags.PrintDefaults()
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %s", ErrUsage, err)
	}
	if flags.NArg() == 0 {
		return ErrUsage
	}
	program, err := compile(flags.Arg(0))
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if !*compact {
		encoder.SetIndent("", "  ")
	}
	r := &runner{program: program, encoder: encoder, out: stdout, raw: *raw}
	if *null {
		return r.evaluate(nil)
	}
	return r.inputs(flags.Args()[1:], stdin)
}

func (r *runner) inputs(paths []string, stdin io.Reader) error {
	if len(paths) == 0 {
		return r.stream(stdin)
	}
	for _, path := range paths {
		if err := r.file(path); err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) file(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return r.stream(f)
}

// stream decodes each JSON value in input. Numbers stay json.Number, so values
// that pass through unchanged keep their exact text
func (r *runner) stream(input io.Reader) error {
	decoder := json.NewDecoder(input)
	decoder.UseNumber()
	for {
		var this any
		err := decoder.Decode(&this)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInput, err)
		}
		if err := r.evaluate(this); err != nil {
			return err
		}
	}
}

func (r *runner) evaluate(this any) error {
	result, err := r.program(this)
	if err != nil {
		return err
	}
	if argex.IsMissing(result) {
		return nil
	}
	if text, ok := result.(string); ok && r.raw {
		_, err := fmt.Fprintln(r.out, text)
		return err
	}
	return r.encoder.Encode(result)
}

func compile(source string) (argex.Program, error) {
	expr, err := argex.Parse(source)
	if err != nil {
		return nil, err
	}
	return argex.Compile(expr)
}
