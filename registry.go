package easycel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// Registry resolves typing information by modeling attributes
// as fields in nested proto messages with presence.
type Registry struct {
	ref.TypeRegistry

	functions *functionStore

	vars *varStore

	tagName string
}

type registryOption func(*Registry)

// WithTagName sets the tag name for the registry
func WithTagName(tagName string) registryOption {
	return func(p *Registry) {
		p.tagName = tagName
	}
}

// NewRegistry creates a new Registry
func NewRegistry(opts ...registryOption) (*Registry, error) {
	p := &Registry{}

	types.NewEmptyRegistry()
	registry, err := types.NewRegistry()
	if err != nil {
		return nil, err
	}
	p.TypeRegistry = registry

	p.vars = newVarStore()
	p.tagName = "easycel"

	for _, opt := range opts {
		opt(p)
	}

	p.functions = newFunctionStore(p, p.tagName)

	return p, nil
}

// Register registers a value in the registry
func (r *Registry) Register(name string, v interface{}) error {
	val := reflect.ValueOf(v)
	typ := val.Type()
	if val.Kind() == reflect.Func {
		err := r.functions.RegisterOverload(name, 0, typ, val)
		if err != nil {
			return err
		}
		err = r.functions.RegisterInstanceOverload(name, 0, typ, val)
		if err != nil {
			return err
		}
	} else {
		err := r.registryType(name, typ, val)
		if err != nil {
			return err
		}
	}
	return nil
}

// RegisterFunction registers a function in the registry
func (r *Registry) RegisterFunction(name string, operandTrait int, v interface{}) error {
	val := reflect.ValueOf(v)
	typ := val.Type()
	err := r.functions.RegisterOverload(name, operandTrait, typ, val)
	if err != nil {
		return err
	}
	return nil
}

// RegisterMethod registers a method in the registry
func (r *Registry) RegisterMethod(name string, operandTrait int, v interface{}) error {
	val := reflect.ValueOf(v)
	typ := val.Type()
	err := r.functions.RegisterInstanceOverload(name, operandTrait, typ, val)
	if err != nil {
		return err
	}
	return nil
}

// RegisterType registers a type in the registry
func (r *Registry) RegisterType(name string, v interface{}) error {
	val := reflect.ValueOf(v)
	typ := val.Type()

	err := r.registryType(name, typ, val)
	if err != nil {
		return err
	}

	return nil
}

func (r *Registry) register(name string, typ reflect.Type, val reflect.Value) error {
	if typ.Kind() == reflect.Func {
		err := r.functions.RegisterOverload(name, 0, typ, val)
		if err != nil {
			return err
		}
	} else {
		err := r.registryType(name, typ, val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) registryType(name string, typ reflect.Type, val reflect.Value) error {
	refVal, err := convertToRefVal(r.TypeRegistry, val, r.tagName)
	if err != nil {
		return err
	}

	pbType, ok := convertToExprType(typ)
	if !ok {
		return fmt.Errorf("type %s is not supported", typ.String())
	}

	r.vars.Register(name, pbType, refVal)
	return nil
}

// CompileOptions returns the options for the cel.Env
func (r *Registry) CompileOptions() []cel.EnvOption {
	opts := []cel.EnvOption{}
	opts = append(opts,
		cel.CustomTypeProvider(r),
		cel.CustomTypeAdapter(r),
		cel.Lib(r.functions),
		cel.Lib(r.vars),
	)
	return opts
}

// NativeToValue converts a native Go value to a ref.Val.
func (r *Registry) NativeToValue(value interface{}) ref.Val {
	v, err := convertToRefVal(r.TypeRegistry, reflect.ValueOf(value), r.tagName)
	if err != nil {
		return types.NewErr(err.Error())
	}
	return v
}

// EnumValue returns the ref.Val value of the enum name.
func (r *Registry) EnumValue(enumName string) ref.Val {
	return r.TypeRegistry.EnumValue(enumName)
}

// FindIdent returns the ref.Val value of the ident name.
func (r *Registry) FindIdent(identName string) (ref.Val, bool) {
	return r.TypeRegistry.FindIdent(identName)
}

// FindType returns the exprpb.Type value of the type name.
func (r *Registry) FindType(typeName string) (*exprpb.Type, bool) {
	return r.TypeRegistry.FindType(typeName)
}

// FindFieldType returns the ref.FieldType value of the field name.
func (r *Registry) FindFieldType(messageType string, fieldName string) (*ref.FieldType, bool) {
	return r.TypeRegistry.FindFieldType(messageType, fieldName)
}

// NewValue returns a new ref.Val instance of the given type name.
func (r *Registry) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	return r.TypeRegistry.NewValue(typeName, fields)
}
