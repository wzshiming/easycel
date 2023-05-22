package easycel

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common"
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
	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, err
	}
	return &Environment{
		env: env,
	}, nil
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
	ast, issue = e.env.Check(ast)
	if issue != nil && issue.Err() != nil {
		return nil, issue.Err()
	}
	return e.env.Program(ast, opts...)
}
