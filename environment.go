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

type Environment struct {
	env *cel.Env
}

func NewEnvironment(opts ...cel.EnvOption) (*Environment, error) {
	opts = append(opts,
		cel.StdLib(),
		ext.Strings(),
		cel.HomogeneousAggregateLiterals(),
	)
	env, err := cel.NewCustomEnv(opts...)
	if err != nil {
		return nil, err
	}
	return &Environment{
		env: env,
	}, nil
}

func (e *Environment) Program(src string, opts ...cel.ProgramOption) (cel.Program, error) {
	if src == "" {
		return nil, errNoSourceCode
	}
	source := common.NewStringSource(src, "")
	past, iss := e.env.ParseSource(source)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}
	past, iss = e.env.Check(past)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}
	return e.env.Program(past, opts...)
}
