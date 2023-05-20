package easycel

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common"
	"github.com/google/cel-go/ext"
)

var (
	errNoSourceCode = fmt.Errorf("no source code")
)

// Environment is a wrapper around CEL environment
type Environment struct {
	env *cel.Env
}

// NewEnvironment creates a new CEL environment
func NewEnvironment(opts ...cel.EnvOption) (*Environment, error) {
	env, err := cel.NewCustomEnv(opts...)
	if err != nil {
		return nil, err
	}
	return &Environment{
		env: env,
	}, nil
}

// NewEnvironmentWithExtensions creates a new CEL environment with extensions
func NewEnvironmentWithExtensions(opts ...cel.EnvOption) (*Environment, error) {
	opts = append(opts,
		cel.StdLib(),
		cel.HomogeneousAggregateLiterals(),
		ext.Strings(),
		ext.Encoders(),
	)
	return NewEnvironment(opts...)
}

// Program creates a new CEL program
func (e *Environment) Program(src string, opts ...cel.ProgramOption) (cel.Program, error) {
	if src == "" {
		return nil, errNoSourceCode
	}
	source := common.NewStringSource(src, "")
	ast, issue := e.env.ParseSource(source)
	if issue != nil && issue.Err() != nil {
		return nil, issue.Err()
	}
	return e.env.Program(ast, opts...)
}
