package easycel

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/overloads"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

type Registry struct {
	nativeTypeProvider *nativeTypeProvider
	funcs              map[string][]cel.FunctionOpt
	variables          map[string]*cel.Type
	registry           *types.Registry
	adapter            types.Adapter
	provider           types.Provider
	tagName            string
	libraryName        string
}

type RegistryOption func(*Registry)

// WithTypeAdapter sets the type adapter used to convert types to CEL types.
func WithTypeAdapter(adapter types.Adapter) RegistryOption {
	return func(r *Registry) {
		r.adapter = adapter
	}
}

// WithTypeProvider sets the type provider used to convert types to CEL types.
func WithTypeProvider(provider types.Provider) RegistryOption {
	return func(r *Registry) {
		r.provider = provider
	}
}

// WithTagName sets the tag name used to convert types to CEL types.
func WithTagName(tagName string) RegistryOption {
	return func(r *Registry) {
		r.tagName = tagName
	}
}

// NewRegistry creates adapter new Registry.
func NewRegistry(libraryName string, opts ...RegistryOption) *Registry {
	r := &Registry{
		funcs:       make(map[string][]cel.FunctionOpt),
		variables:   make(map[string]*cel.Type),
		tagName:     "easycel",
		libraryName: libraryName,
	}
	for _, opt := range opts {
		opt(r)
	}
	registry, _ := types.NewRegistry()
	tp := newNativeTypeProvider(r.tagName, registry, registry)
	if r.adapter == nil {
		r.adapter = tp
	}
	if r.provider == nil {
		r.provider = tp
	}
	r.registry = registry
	r.nativeTypeProvider = tp

	return r
}

// LibraryName implements the Library interface method.
func (r *Registry) LibraryName() string {
	return r.libraryName
}

// CompileOptions implements the Library interface method.
func (r *Registry) CompileOptions() []cel.EnvOption {
	opts := []cel.EnvOption{}
	for name, fn := range r.funcs {
		opts = append(opts, cel.Function(name, fn...))
	}
	for name, typ := range r.variables {
		opts = append(opts, cel.Variable(name, typ))
	}
	opts = append(opts,
		cel.CustomTypeAdapter(r),
		cel.CustomTypeProvider(r),
	)
	return opts
}

// NativeToValue converts the input `value` to adapter CEL `ref.Val`.
func (r *Registry) NativeToValue(value any) ref.Val {
	return r.adapter.NativeToValue(value)
}

// EnumValue returns the numeric value of the given enum value name.
func (r *Registry) EnumValue(enumName string) ref.Val {
	return r.provider.EnumValue(enumName)
}

// FindIdent takes adapter qualified identifier name and returns adapter Value if one exists.
func (r *Registry) FindIdent(identName string) (ref.Val, bool) {
	return r.provider.FindIdent(identName)
}

// FindStructType returns the Type give a qualified type name.
func (r *Registry) FindStructType(structType string) (*types.Type, bool) {
	return r.provider.FindStructType(structType)
}

// FindStructFieldType returns the field type for a checked type value. Returns
// false if the field could not be found.
func (r *Registry) FindStructFieldType(structType, fieldName string) (*types.FieldType, bool) {
	return r.provider.FindStructFieldType(structType, fieldName)
}

// NewValue creates adapter new type value from adapter qualified name and map of field name to value.
func (r *Registry) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	return r.provider.NewValue(typeName, fields)
}

// ProgramOptions implements the Library interface method.
func (r *Registry) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{}
}

// RegisterType registers adapter type with the registry.
func (r *Registry) RegisterType(refTypes any) error {
	switch v := refTypes.(type) {
	case ref.Val:
		err := r.registerTraits(v)
		if err != nil {
			return err
		}
		return r.registry.RegisterType(v.Type())
	case ref.Type:
		return r.registry.RegisterType(v)
	}
	return r.nativeTypeProvider.registerType(reflect.TypeOf(refTypes))
}

// RegisterVariable registers adapter value with the registry.
func (r *Registry) RegisterVariable(name string, val interface{}) error {
	if _, ok := r.variables[name]; ok {
		return fmt.Errorf("variable %s already registered", name)
	}
	typ := reflect.TypeOf(val)
	celType, ok := convertToCelType(typ)
	if !ok {
		return fmt.Errorf("variable %s type %s not supported", name, typ.String())
	}
	r.variables[name] = celType
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
func (r *Registry) RegisterConversion(fun any) error {
	return r.nativeTypeProvider.registerConversionsFunc(fun)
}

func (r *Registry) registerTraits(v ref.Val) error {
	typ := v.Type()

	tv := getTypeValue(reflect.TypeOf(v))

	if _, ok := v.(traits.Adder); ok && typ.HasTrait(traits.AdderType) {
		argsCelType := []*cel.Type{
			tv,
			types.DynType,
		}
		funcName := operators.Add
		resultType := tv

		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.AdderType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
	}

	if _, ok := v.(traits.Subtractor); ok && typ.HasTrait(traits.SubtractorType) {
		argsCelType := []*cel.Type{
			tv,
			types.DynType,
		}
		funcName := operators.Subtract
		resultType := tv

		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.SubtractorType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
	}

	if _, ok := v.(traits.Negater); ok && typ.HasTrait(traits.NegatorType) {
		argsCelType := []*cel.Type{
			tv,
		}
		funcName := operators.Negate
		resultType := tv

		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.NegatorType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
	}

	if _, ok := v.(traits.Multiplier); ok && typ.HasTrait(traits.MultiplierType) {
		argsCelType := []*cel.Type{
			tv,
			types.DynType,
		}
		funcName := operators.Multiply
		resultType := tv

		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.MultiplierType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
	}

	if _, ok := v.(traits.Divider); ok && typ.HasTrait(traits.DividerType) {
		argsCelType := []*cel.Type{
			tv,
			types.DynType,
		}
		funcName := operators.Divide
		resultType := tv

		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.DividerType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
	}

	if _, ok := v.(traits.Modder); ok && typ.HasTrait(traits.ModderType) {
		argsCelType := []*cel.Type{
			tv,
			types.DynType,
		}
		funcName := operators.Modulo
		resultType := tv

		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.ModderType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
	}

	if _, ok := v.(traits.Comparer); ok && typ.HasTrait(traits.ComparerType) {
		argsCelType := []*cel.Type{
			tv,
			types.DynType,
		}
		resultType := types.IntType

		comparers := []string{
			operators.Greater,
			operators.GreaterEquals,
			operators.Less,
			operators.LessEquals,
		}
		for _, comparer := range comparers {
			funcName := comparer
			overloadID := getOverloadID(funcName, argsCelType, resultType, false)
			funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.ComparerType))
			r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
		}
	}

	if _, ok := v.(traits.Indexer); ok && typ.HasTrait(traits.IndexerType) {
		argsCelType := []*cel.Type{
			tv,
			types.DynType,
		}
		resultType := types.DynType

		funcName := operators.Index
		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.IndexerType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
	}

	if _, ok := v.(traits.Sizer); ok && typ.HasTrait(traits.SizerType) {
		argsCelType := []*cel.Type{
			tv,
		}
		resultType := types.IntType

		funcName := overloads.Size
		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.SizerType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
	}

	if _, ok := v.(traits.Container); ok && typ.HasTrait(traits.ContainerType) {
		argsCelType := []*cel.Type{
			types.DynType,
			tv,
		}
		resultType := types.BoolType

		funcName := operators.In
		overloadID := getOverloadID(funcName, argsCelType, resultType, false)
		funcOpt := cel.Overload(overloadID, argsCelType, resultType, cel.OverloadOperandTrait(traits.ContainerType))
		r.funcs[funcName] = append(r.funcs[funcName], funcOpt)
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

	overloadID := getOverloadID(name, argsCelType, resultType, member)
	opts := []cel.OverloadOpt{overloadOpt}
	var funcOpt cel.FunctionOpt
	if member {
		funcOpt = cel.MemberOverload(overloadID, argsCelType, resultType, opts...)
	} else {
		funcOpt = cel.Overload(overloadID, argsCelType, resultType, opts...)
	}
	r.funcs[name] = append(r.funcs[name], funcOpt)
	return nil
}

func getOverloadID(name string, args []*cel.Type, resultType *cel.Type, member bool) string {
	if member {
		return fmt.Sprintf("%s|member@|%s|%s", name, getTypesID(args), resultType.String())
	}
	return fmt.Sprintf("%s|@|%s|%s", name, getTypesID(args), resultType.String())
}

func getTypesID(types []*cel.Type) string {
	if len(types) == 0 {
		return ""
	}
	out := types[0].String()
	for _, typ := range types[1:] {
		out += "," + typ.String()
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

	numIn := typ.NumIn()
	isRefVal := make([]bool, numIn)
	isPtr := make([]bool, numIn)
	if numIn > 0 {
		for i := 0; i < numIn; i++ {
			in := typ.In(i)
			isRefVal[i] = in.Implements(refValType)
			isPtr[i] = in.Kind() == reflect.Ptr
		}
	}

	switch numIn {
	case 1:
		return cel.UnaryBinding(func(value ref.Val) ref.Val {
			val, err := reflectFuncCall(funVal,
				[]reflect.Value{
					convertToReflectValue(value, isRefVal[0], isPtr[0]),
				},
			)
			if err != nil {
				return types.WrapErr(err)
			}
			return r.NativeToValue(val.Interface())
		}), nil
	case 2:
		return cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
			val, err := reflectFuncCall(funVal,
				[]reflect.Value{
					convertToReflectValue(lhs, isRefVal[0], isPtr[0]),
					convertToReflectValue(rhs, isRefVal[1], isPtr[1]),
				},
			)
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
			for i, value := range values {
				vals = append(vals,
					convertToReflectValue(value, isRefVal[i], isPtr[i]),
				)
			}
			val, err := reflectFuncCall(funVal, vals)
			if err != nil {
				return types.WrapErr(err)
			}
			return r.NativeToValue(val.Interface())
		}), nil
	}
}

func convertToReflectValue(val ref.Val, isRefVal, isPtr bool) reflect.Value {
	var value reflect.Value
	if isRefVal {
		value = reflect.ValueOf(val)
	} else {
		value = reflect.ValueOf(val.Value())
		if isPtr {
			if value.Kind() != reflect.Ptr {
				value = value.Addr()
			}
		} else {
			if value.Kind() == reflect.Ptr {
				value = value.Elem()
			}
		}
	}
	return value
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

func fieldNameWithTag(field reflect.StructField, tagName string) (name string, exported bool) {
	value, ok := field.Tag.Lookup(tagName)
	if !ok {
		return field.Name, true
	}

	name = strings.Split(value, ",")[0]
	if name == "-" {
		return "", false
	}

	if name == "" {
		name = field.Name
	}
	return name, true
}
