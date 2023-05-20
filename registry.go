package easycel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type Registry struct {
	conversions map[reflect.Type]func(adapter ref.TypeAdapter, value reflect.Value) ref.Val
	types       map[string]ref.Type
	funcs       map[string][]cel.FunctionOpt
	adapter     ref.TypeAdapter
	provider    ref.TypeProvider
	libraryName string
}

type RegistryOption func(*Registry)

// WithTypeAdapter sets the type adapter used to convert types to CEL types.
func WithTypeAdapter(adapter ref.TypeAdapter) RegistryOption {
	return func(r *Registry) {
		r.adapter = adapter
	}
}

// WithTypeProvider sets the type provider used to convert types to CEL types.
func WithTypeProvider(provider ref.TypeProvider) RegistryOption {
	return func(r *Registry) {
		r.provider = provider
	}
}

// NewRegistry creates adapter new Registry.
func NewRegistry(libraryName string, opts ...RegistryOption) *Registry {
	r := &Registry{
		conversions: make(map[reflect.Type]func(adapter ref.TypeAdapter, value reflect.Value) ref.Val),
		types:       make(map[string]ref.Type),
		funcs:       make(map[string][]cel.FunctionOpt),
		libraryName: libraryName,
	}
	for _, opt := range opts {
		opt(r)
	}
	registry, _ := types.NewRegistry()
	if r.adapter == nil {
		r.adapter = registry
	}
	if r.provider == nil {
		r.provider = registry
	}
	return r
}

// LibraryName implements the Library interface method.
func (r *Registry) LibraryName() string {
	return r.libraryName
}

// CompileOptions implements the Library interface method.
func (r *Registry) CompileOptions() []cel.EnvOption {
	opts := make([]cel.EnvOption, 0, len(r.types)+len(r.funcs)+1)
	for _, typ := range r.types {
		opts = append(opts, cel.Types(typ))
	}
	for name, fn := range r.funcs {
		opts = append(opts, cel.Function(name, fn...))
	}
	opts = append(opts, cel.CustomTypeAdapter(r))
	return opts
}

// NativeToValue converts the input `value` to adapter CEL `ref.Val`.
func (r Registry) NativeToValue(value any) ref.Val {
	refVal := reflect.ValueOf(value)
	if convert := r.conversions[refVal.Type()]; convert != nil {
		return convert(r, refVal)
	}

	val := generalAdapter{r.adapter}.NativeToValue(value)
	if !types.IsError(val) {
		return val
	}
	return r.adapter.NativeToValue(value)
}

// EnumValue returns the numeric value of the given enum value name.
func (r Registry) EnumValue(enumName string) ref.Val {
	return r.provider.EnumValue(enumName)
}

// FindIdent takes adapter qualified identifier name and returns adapter Value if one exists.
func (r Registry) FindIdent(identName string) (ref.Val, bool) {
	return r.provider.FindIdent(identName)
}

// FindType looks up the Type given adapter qualified typeName.
func (r Registry) FindType(typeName string) (*exprpb.Type, bool) {
	return r.provider.FindType(typeName)
}

// FindFieldType returns the field type for adapter checked type value.
func (r Registry) FindFieldType(messageType string, fieldName string) (*ref.FieldType, bool) {
	return r.provider.FindFieldType(messageType, fieldName)
}

// NewValue creates adapter new type value from adapter qualified name and map of field name to value.
func (r Registry) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	return r.provider.NewValue(typeName, fields)
}

// ProgramOptions implements the Library interface method.
func (Registry) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{}
}

// RegisterType registers adapter type with the registry.
func (r *Registry) RegisterType(name string, val interface{}) error {
	if _, ok := r.types[name]; ok {
		return fmt.Errorf("type %s already registered", name)
	}
	typ := reflect.TypeOf(val)
	typVal := GetTypeValue(typ)
	r.types[name] = typVal
	return nil
}

// RegisterFunction registers adapter function with the registry.
func (r *Registry) RegisterFunction(name string, fun interface{}) error {
	return r.registerFunction(name, fun, false)
}

// RegisterMethod registers adapter method with the registry.
func (r *Registry) RegisterMethod(name string, fun interface{}) error {
	return r.registerFunction(name, fun, true)
}

// RegisterConversion registers adapter conversion function with the registry.
func (r *Registry) RegisterConversion(typ reflect.Type, fun func(adapter ref.TypeAdapter, value reflect.Value) ref.Val) error {
	if _, ok := r.conversions[typ]; ok {
		return fmt.Errorf("conversion for type %s already registered", typ.String())
	}

	r.conversions[typ] = fun
	if typ.Kind() == reflect.Struct {
		ptyp := reflect.PtrTo(typ)
		if _, ok := r.conversions[ptyp]; !ok {
			r.conversions[ptyp] = func(adapter ref.TypeAdapter, value reflect.Value) ref.Val {
				return fun(adapter, value.Elem())
			}
		}
	}
	return nil
}

func (r *Registry) registerFunction(name string, fun interface{}, member bool) error {
	funVal := reflect.ValueOf(fun)
	if funVal.Kind() != reflect.Func {
		return fmt.Errorf("func must be func")
	}
	typ := funVal.Type()
	overloadOpt, err := r.getOverloadOpt(typ, funVal)
	if err != nil {
		return err
	}

	numIn := typ.NumIn()
	if member {
		if numIn == 0 {
			return fmt.Errorf("method must have at least one argument")
		}
	}
	argsCelType := make([]*cel.Type, 0, numIn)
	argsReflectType := make([]reflect.Type, 0, numIn)
	for i := 0; i < numIn; i++ {
		in := typ.In(i)
		celType, ok := convertToCelType(in)
		if !ok {
			return fmt.Errorf("invalid input type %s", in.String())
		}
		argsCelType = append(argsCelType, celType)
		argsReflectType = append(argsReflectType, in)
	}

	out := typ.Out(0)
	resultType, ok := convertToCelType(out)
	if !ok {
		return fmt.Errorf("invalid output type %s", out.String())
	}

	overloadID := formatFunction(name, argsReflectType, out)
	var funcOpt cel.FunctionOpt
	if member {
		funcOpt = cel.MemberOverload(overloadID, argsCelType, resultType, overloadOpt)
	} else {
		funcOpt = cel.Overload(overloadID, argsCelType, resultType, overloadOpt)
	}
	r.funcs[name] = append(r.funcs[name], funcOpt)
	return nil
}

func formatFunction(name string, args []reflect.Type, resultType reflect.Type) string {
	return fmt.Sprintf("%s_@%s_%s", name, formatTypes(args), typeName(resultType))
}

func formatTypes(types []reflect.Type) string {
	if len(types) == 0 {
		return ""
	}
	out := typeName(types[0])
	for _, typ := range types[1:] {
		out += "_" + typeName(typ)
	}
	return out
}

func (r *Registry) getOverloadOpt(typ reflect.Type, funVal reflect.Value) (out cel.OverloadOpt, err error) {
	numOut := typ.NumOut()
	switch numOut {
	default:
		return nil, fmt.Errorf("too many result")
	case 0:
		return nil, fmt.Errorf("result is required")
	case 2:
		if !typ.Out(1).AssignableTo(errorType) {
			return nil, fmt.Errorf("last result must be error %s", typ.String())
		}
	case 1:
	}

	switch typ.NumIn() {
	case 1:
		return cel.UnaryBinding(func(value ref.Val) ref.Val {
			val, err := reflectFuncCall(funVal, []reflect.Value{reflect.ValueOf(value)})
			if err != nil {
				return types.WrapErr(err)
			}
			return r.NativeToValue(val.Interface())
		}), nil
	case 2:
		return cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
			val, err := reflectFuncCall(funVal, []reflect.Value{reflect.ValueOf(lhs), reflect.ValueOf(rhs)})
			if err != nil {
				return types.WrapErr(err)
			}
			return r.NativeToValue(val.Interface())
		}), nil
	case 0:
		return cel.FunctionBinding(func(values ...ref.Val) ref.Val {
			val, err := reflectFuncCall(funVal, []reflect.Value{})
			if err != nil {
				return types.WrapErr(err)
			}
			return r.NativeToValue(val.Interface())
		}), nil
	default:
		return cel.FunctionBinding(func(values ...ref.Val) ref.Val {
			vals := make([]reflect.Value, 0, len(values))
			for _, value := range values {
				vals = append(vals, reflect.ValueOf(value))
			}
			val, err := reflectFuncCall(funVal, vals)
			if err != nil {
				return types.WrapErr(err)
			}
			return r.NativeToValue(val.Interface())
		}), nil
	}
}

func reflectFuncCall(funVal reflect.Value, values []reflect.Value) (reflect.Value, error) {
	results := funVal.Call(values)
	if len(results) == 2 {
		err, _ := results[1].Interface().(error)
		if err != nil {
			return reflect.Value{}, err
		}
	}
	return results[0], nil
}
