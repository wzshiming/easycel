package easycel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// Provider resolves typing information by modeling attributes
// as fields in nested proto messages with presence.
type Provider struct {
	ref.TypeRegistry

	functions *functionStore
	adapter   *adapter
	vars      *varStore
}

func NewProvider() (*Provider, error) {
	p := &Provider{}

	registry, err := types.NewRegistry()
	if err != nil {
		return nil, err
	}
	adapter := newAdapter(p)
	functions := newFunctionStore(adapter)
	vars := newVarStore()

	p.TypeRegistry = registry
	p.adapter = adapter
	p.functions = functions
	p.vars = vars

	return p, nil
}

func (p *Provider) Registry(name string, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Func {
		err := p.functions.RegisterOverload(name, 0, val)
		if err != nil {
			return err
		}
	} else {
		if _, ok := v.(ref.Val); !ok {
			vv := p.NativeToValue(v)
			if vv.Type() == types.ErrType {
				return fmt.Errorf("%s", vv.Value())
			}
			v = vv
		}

		var pbType *exprpb.Type
		if t, ok := v.(PbType); ok {
			pbType = t.PbType()
		} else {
			t, err := p.adapter.TypeToPbType(reflect.TypeOf(v.(ref.Val).Value()))
			if err != nil {
				return err
			}
			pbType = t
		}

		p.vars.Registry(name, pbType, v)
		if met, ok := v.(CelMethods); ok {
			typ := reflect.TypeOf(v)
			for _, name := range met.CelMethods() {
				method, ok := typ.MethodByName(name)
				if !ok {
					continue
				}
				err := p.functions.RegisterInstanceOverload(method.Name, 0, method.Func)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (p *Provider) CompileOptions() []cel.EnvOption {
	opts := []cel.EnvOption{}
	opts = append(opts,
		cel.CustomTypeProvider(p),
		cel.CustomTypeAdapter(p),
		cel.Lib(p.functions),
		cel.Lib(p.vars),
	)
	return opts
}

func (p *Provider) NativeToValue(value interface{}) ref.Val {
	switch c := value.(type) {
	case CelVal:
		return c.CelVal()
	case ref.Val:
		return c
	}
	return p.TypeRegistry.NativeToValue(value)
}
