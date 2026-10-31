package engine

import (
	"fmt"
	"maps"
)

// Env holds immutable global bindings for compiling expressions. Locals shadow
// globals, and globals shadow input properties
type Env struct {
	entries map[string]any
}

// NewEnv returns an empty environment
func NewEnv() *Env {
	return &Env{}
}

// Bind returns an environment binding name to value, leaving e unchanged.
// Values are shared with compiled programs and must not be mutated while in use
func (e *Env) Bind(name string, value any) (*Env, error) {
	if e == nil {
		return nil, fmt.Errorf("%w: no environment", ErrFunction)
	}
	if fn, ok := value.(*Function); ok && !fn.valid() {
		return nil, fmt.Errorf("%w: %s", ErrFunction, name)
	}
	entries := maps.Clone(e.entries)
	if entries == nil {
		entries = map[string]any{}
	}
	entries[name] = value
	return &Env{entries: entries}, nil
}

func (e *Env) resolve(name string) (any, bool) {
	value, ok := e.entries[name]
	return value, ok
}
