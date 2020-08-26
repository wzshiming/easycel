package easycel

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
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
func (v *varStore) Registry(name string, decl *exprpb.Type, val interface{}) {
	_, ok := v.vars[name]
	if ok {
		return
	}
	v.vars[name] = val
	v.declarations = append(v.declarations, decls.NewVar(name, decl))
}

func (v *varStore) CompileOptions() []cel.EnvOption {
	out := []cel.EnvOption{}
	if len(v.declarations) != 0 {
		out = append(out, cel.Declarations(v.declarations...))
	}
	return out
}

func (v *varStore) ProgramOptions() []cel.ProgramOption {
	out := []cel.ProgramOption{}
	if len(v.vars) != 0 {
		out = append(out, cel.Globals(v.vars))
	}
	return out
}
