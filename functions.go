package easycel

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type functionStore struct {
	decls     map[string][]*exprpb.Decl_FunctionDecl_Overload
	overloads []*functions.Overload
	adapter   ref.TypeAdapter
	uniq      map[string]struct{}
	tagName   string
}

func newFunctionStore(adapter ref.TypeAdapter, tagName string) *functionStore {
	return &functionStore{
		decls:   map[string][]*exprpb.Decl_FunctionDecl_Overload{},
		adapter: adapter,
		uniq:    map[string]struct{}{},
		tagName: tagName,
	}
}

// CompileOptions returns the CEL environment options for the function store.
func (f *functionStore) CompileOptions() []cel.EnvOption {
	out := []cel.EnvOption{}
	if len(f.decls) != 0 {
		decl := make([]*exprpb.Decl, 0, len(f.decls))
		for name, funcs := range f.decls {
			decl = append(decl, decls.NewFunction(name, funcs...))
		}
		sort.Slice(decl, func(i, j int) bool {
			return decl[i].Name < decl[j].Name
		})
		out = append(out, cel.Declarations(decl...))
	}
	return out
}

// ProgramOptions returns the CEL program options for the function store.
func (f *functionStore) ProgramOptions() []cel.ProgramOption {
	out := []cel.ProgramOption{}
	if len(f.overloads) != 0 {
		out = append(out, cel.Functions(f.overloads...))
	}
	return out
}

func (f *functionStore) registerFunction(operator string, operandTrait int, fun interface{}) error {
	switch fun := fun.(type) {
	case functions.UnaryOp:
		f.overloads = append(f.overloads, &functions.Overload{
			Operator:     operator,
			OperandTrait: operandTrait,
			Unary:        fun,
		})
	case functions.BinaryOp:
		f.overloads = append(f.overloads, &functions.Overload{
			Operator:     operator,
			OperandTrait: operandTrait,
			Binary:       fun,
		})
	case functions.FunctionOp:
		f.overloads = append(f.overloads, &functions.Overload{
			Operator:     operator,
			OperandTrait: operandTrait,
			Function:     fun,
		})
	}
	return nil
}

func (f *functionStore) uniqCheck(name string) bool {
	_, ok := f.uniq[name]
	if ok {
		return false
	}
	f.uniq[name] = struct{}{}
	return true
}

// RegisterOverload registers a function overload.
func (f *functionStore) RegisterOverload(name string, operandTrait int, typ reflect.Type, funVal reflect.Value) error {
	operator := getUniqTypeName("overload_"+name, typ)
	if !f.uniqCheck(operator) {
		return nil
	}

	ok, err := f.checkFunction(typ)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("function %s is not a valid", typ)
	}
	argTypes, resultType, err := f.getDeclFunc(typ)
	if err != nil {
		return err
	}
	err = f.registerOverload(name, operator, argTypes, resultType)
	if err != nil {
		return err
	}
	return f.registerOperator(operator, operandTrait, typ, funVal)
}

// RegisterInstanceOverload registers a function instance overload.
func (f *functionStore) RegisterInstanceOverload(name string, operandTrait int, typ reflect.Type, funVal reflect.Value) error {
	operator := getUniqTypeName("instance_overload_"+name, typ)
	if !f.uniqCheck(operator) {
		return nil
	}

	ok, err := f.checkFunction(typ)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	argTypes, resultType, err := f.getDeclFunc(typ)
	if err != nil {
		return err
	}
	err = f.registerInstanceOverload(name, operator, argTypes, resultType)
	if err != nil {
		return err
	}
	return f.registerOperator(operator, operandTrait, typ, funVal)
}

func (f *functionStore) registerOperator(operator string, operandTrait int, typ reflect.Type, funVal reflect.Value) error {
	out, err := f.reflectWrapFunc(typ, funVal)
	if err != nil {
		return err
	}
	err = f.registerFunction(operator, operandTrait, out)
	if err != nil {
		return err
	}
	return nil
}

func (f *functionStore) registerOverload(name string, operator string, argTypes []*exprpb.Type, resultType *exprpb.Type) error {
	f.decls[name] = append(f.decls[name], decls.NewOverload(operator, argTypes, resultType))
	return nil
}

func (f *functionStore) registerInstanceOverload(name string, operator string, argTypes []*exprpb.Type, resultType *exprpb.Type) error {
	f.decls[name] = append(f.decls[name], decls.NewInstanceOverload(operator, argTypes, resultType))
	return nil
}

func (f *functionStore) checkFunction(typ reflect.Type) (bool, error) {
	numOut := typ.NumOut()
	switch numOut {
	default:
		return false, fmt.Errorf("too many result")
	case 2:
		if !typ.Out(1).AssignableTo(errType) {
			return false, nil
		}
		fallthrough
	case 0, 1:
		// Skip
	}

	return true, nil
}

func (f *functionStore) getDeclFunc(typ reflect.Type) (argTypes []*exprpb.Type, resultType *exprpb.Type, err error) {
	var ok bool
	numOut := typ.NumOut()
	switch numOut {
	default:
		return nil, nil, fmt.Errorf("too many result")
	case 0:
		return nil, nil, fmt.Errorf("result is required")
	case 1, 2:
		resultType, ok = convertToExprType(typ.Out(0))
		if !ok {
			return nil, nil, fmt.Errorf("the result of function %s is not supported", typ.String())
		}
		if resultType == decls.Null {
			return nil, nil, fmt.Errorf("the result of function %s is unspecified", typ.String())
		}
	}

	numIn := typ.NumIn()
	argTypes = make([]*exprpb.Type, 0, numIn)
	for i := 0; i != numIn; i++ {
		param, ok := convertToExprType(typ.In(i))
		if !ok {
			return nil, nil, fmt.Errorf("the %d parameter of function %s is not supported", i, typ.String())
		}
		if param == decls.Null {
			return nil, nil, fmt.Errorf("the %d parameter of function %s is unspecified", i, typ.String())
		}
		argTypes = append(argTypes, param)
	}
	return argTypes, resultType, nil
}

func (f *functionStore) reflectWrapFunc(typ reflect.Type, funVal reflect.Value) (out interface{}, err error) {
	var needErr bool
	var ok bool
	var result *exprpb.Type
	numOut := typ.NumOut()
	switch numOut {
	default:
		return nil, fmt.Errorf("too many result")
	case 0:
		return nil, fmt.Errorf("result is required")
	case 2:
		if !typ.Out(1).AssignableTo(errType) {
			return nil, fmt.Errorf("last result must be error %s", typ.String())
		}
		needErr = true
		fallthrough
	case 1:
		result, ok = convertToExprType(typ.Out(0))
		if !ok {
			return nil, fmt.Errorf("the result of function %s is not supported", typ.String())
		}
		if result == decls.Null {
			return nil, fmt.Errorf("the result of function %s is unspecified", typ.String())
		}
	}

	numIn := typ.NumIn()
	for i := 0; i != numIn; i++ {
		param, ok := convertToExprType(typ.In(i))
		if !ok {
			return nil, fmt.Errorf("the %d parameter of function %s is not supported", i, typ.String())
		}
		if param == decls.Null {
			return nil, fmt.Errorf("the %d parameter of function %s is unspecified", i, typ.String())
		}
	}

	funCall := func(values []reflect.Value) ref.Val {
		results := funVal.Call(values)
		if needErr && len(results) == 2 {
			err, _ := results[1].Interface().(error)
			if err != nil {
				return types.NewErr(err.Error())
			}
		}
		out, err := convertToRefVal(f.adapter, results[0], f.tagName)
		if err != nil {
			return types.NewErr(err.Error())
		}
		return out
	}

	switch numIn {
	case 1:
		return functions.UnaryOp(func(value ref.Val) ref.Val {
			return funCall([]reflect.Value{reflect.ValueOf(value.Value())})
		}), nil
	case 2:
		return functions.BinaryOp(func(lhs ref.Val, rhs ref.Val) ref.Val {
			return funCall([]reflect.Value{reflect.ValueOf(lhs.Value()), reflect.ValueOf(rhs.Value())})
		}), nil
	case 0:
		return functions.FunctionOp(func(values ...ref.Val) ref.Val {
			return funCall([]reflect.Value{})
		}), nil
	default:
		return functions.FunctionOp(func(values ...ref.Val) ref.Val {
			vals := make([]reflect.Value, 0, len(values))
			for _, value := range values {
				vals = append(vals, reflect.ValueOf(value.Value()))
			}
			return funCall(vals)
		}), nil
	}
}
