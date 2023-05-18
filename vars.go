package easycel

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/interpreter"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type varStore struct {
	vars         map[string]interface{}
	declarations []*exprpb.Decl
}

func newVarStore() *varStore {
	return &varStore{
		vars: map[string]interface{}{},
	}
}

// Register registers a variable with the given name and type declaration.
func (v *varStore) Register(name string, decl *exprpb.Type, val interface{}) {
	_, ok := v.vars[name]
	if ok {
		return
	}
	v.vars[name] = val
	v.declarations = append(v.declarations, decls.NewVar(name, decl))
}

// CompileOptions returns the options for the CEL environment.
func (v *varStore) CompileOptions() []cel.EnvOption {
	out := []cel.EnvOption{}
	if len(v.declarations) != 0 {
		out = append(out, cel.Declarations(v.declarations...))
	}
	return out
}

// ProgramOptions returns the options for the CEL program.
func (v *varStore) ProgramOptions() []cel.ProgramOption {
	out := []cel.ProgramOption{}
	if len(v.vars) != 0 {
		out = append(out, cel.Globals(v))
	}
	return out
}

// ResolveName returns the value of the variable with the given name, if it exists.
func (v *varStore) ResolveName(name string) (interface{}, bool) {
	obj, ok := v.vars[name]
	if !ok {
		return nil, false
	}
	return obj, ok
}

// Parent returns the parent of the current activation, may be nil.
// If non-nil, the parent will be searched during resolve calls.
func (v *varStore) Parent() interpreter.Activation {
	return nil
}
