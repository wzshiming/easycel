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
	adapter   *adapter
	uniq      map[string]struct{}
}

func newFunctionStore(adapter *adapter) *functionStore {
	return &functionStore{
		decls:   map[string][]*exprpb.Decl_FunctionDecl_Overload{},
		adapter: adapter,
		uniq:    map[string]struct{}{},
	}
}

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

func (f *functionStore) RegisterOverload(name string, operandTrait int, fun reflect.Value) error {
	operator := getUniqTypeName("overload_"+name, fun.Type())
	if !f.uniqCheck(operator) {
		return nil
	}

	ok, err := f.checkFunction(fun)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	argTypes, resultType, err := f.getDeclFunc(fun)
	if err != nil {
		return err
	}
	err = f.registerOverload(name, operator, argTypes, resultType)
	if err != nil {
		return err
	}
	return f.registerOperator(operator, operandTrait, fun)
}

func (f *functionStore) RegisterInstanceOverload(name string, operandTrait int, fun reflect.Value) error {
	operator := getUniqTypeName("instance_overload_"+name, fun.Type())
	if !f.uniqCheck(operator) {
		return nil
	}

	ok, err := f.checkFunction(fun)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	argTypes, resultType, err := f.getDeclFunc(fun)
	if err != nil {
		return err
	}
	err = f.registerInstanceOverload(name, operator, argTypes, resultType)
	if err != nil {
		return err
	}
	return f.registerOperator(operator, operandTrait, fun)
}

func (f *functionStore) registerOperator(operator string, operandTrait int, fun reflect.Value) error {
	out, err := f.reflectWrapFunc(fun)
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

func (f *functionStore) checkFunction(funVal reflect.Value) (bool, error) {
	if funVal.Kind() != reflect.Func {
		return false, fmt.Errorf("must be a function, not a %s", funVal.Kind())
	}
	typ := funVal.Type()

	numOut := typ.NumOut()
	switch numOut {
	default:
		return false, fmt.Errorf("too many result")
	case 0:
		return false, fmt.Errorf("result is required")
	case 2:
		if !typ.Out(1).AssignableTo(errType) {
			return false, nil
		}
		fallthrough
	case 1:
		if !typ.Out(0).AssignableTo(celVal) {
			return false, nil
		}
	}

	numIn := typ.NumIn()
	for i := 0; i != numIn; i++ {
		if !typ.In(i).AssignableTo(celVal) {
			return false, nil
		}
	}
	return true, nil
}

func (f *functionStore) getDeclFunc(funVal reflect.Value) (argTypes []*exprpb.Type, resultType *exprpb.Type, err error) {
	if funVal.Kind() != reflect.Func {
		return nil, nil, fmt.Errorf("must be a function, not a %s", funVal.Kind())
	}
	typ := funVal.Type()

	numOut := typ.NumOut()
	switch numOut {
	default:
		return nil, nil, fmt.Errorf("too many result")
	case 0:
		return nil, nil, fmt.Errorf("result is required")
	case 1, 2:
		resultType, err = f.adapter.TypeToPbType(typ.Out(0))
		if err != nil {
			return nil, nil, err
		}
		if resultType == decls.Null {
			return nil, nil, fmt.Errorf("the result of function %s is unspecified", typ.String())
		}
	}

	numIn := typ.NumIn()
	argTypes = make([]*exprpb.Type, 0, numIn)
	for i := 0; i != numIn; i++ {
		param, err := f.adapter.TypeToPbType(typ.In(i))
		if err != nil {
			return nil, nil, err
		}
		if param == decls.Null {
			return nil, nil, fmt.Errorf("the %d parameter of function %s is unspecified", i, typ.String())
		}
		argTypes = append(argTypes, param)
	}
	return argTypes, resultType, nil
}

func (f *functionStore) reflectWrapFunc(funVal reflect.Value) (out interface{}, err error) {
	if funVal.Kind() != reflect.Func {
		return nil, fmt.Errorf("must be a function, not a %s", funVal.Kind())
	}

	typ := funVal.Type()

	var needErr bool
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
		result, err = f.adapter.TypeToPbType(typ.Out(0))
		if err != nil {
			return nil, err
		}
		if result == decls.Null {
			return nil, fmt.Errorf("the result of function %s is unspecified", typ.String())
		}
	}

	numIn := typ.NumIn()
	for i := 0; i != numIn; i++ {
		param, err := f.adapter.TypeToPbType(typ.In(i))
		if err != nil {
			return nil, err
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
		out, err := f.adapter.ValueToCelValue(results[0])
		if err != nil {
			return types.NewErr(err.Error())
		}
		return out
	}

	switch numIn {
	case 1:
		return functions.UnaryOp(func(value ref.Val) ref.Val {
			return funCall([]reflect.Value{reflect.ValueOf(value)})
		}), nil
	case 2:
		return functions.BinaryOp(func(lhs ref.Val, rhs ref.Val) ref.Val {
			return funCall([]reflect.Value{reflect.ValueOf(lhs), reflect.ValueOf(rhs)})
		}), nil
	case 0:
		return functions.FunctionOp(func(values ...ref.Val) ref.Val {
			return funCall([]reflect.Value{})
		}), nil
	default:
		return functions.FunctionOp(func(values ...ref.Val) ref.Val {
			vals := make([]reflect.Value, 0, len(values))
			for _, value := range values {
				vals = append(vals, reflect.ValueOf(value))
			}
			return funCall(vals)
		}), nil
	}
}
