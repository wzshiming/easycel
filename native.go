package easycel

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// Type provider for native types.
type Type interface {
	ref.Val
	ref.Type
	TagName() string
	GetRawType() reflect.Type
}

func newNativeTypeProvider(tagName string, adapter types.Adapter, provider types.Provider) *nativeTypeProvider {
	return &nativeTypeProvider{
		tagName:      tagName,
		conversions:  make(map[reflect.Type]*convertType),
		nativeTypes:  make(map[string]Type),
		baseAdapter:  adapter,
		baseProvider: provider,
	}
}

type nativeTypeProvider struct {
	tagName      string
	conversions  map[reflect.Type]*convertType
	nativeTypes  map[string]Type
	baseAdapter  types.Adapter
	baseProvider types.Provider
}

type convertType struct {
	targetType  reflect.Type
	convertFunc reflect.Value
}

func (tp *nativeTypeProvider) registerConversionsFunc(fun interface{}) error {
	typ := reflect.TypeOf(fun)
	if typ.Kind() != reflect.Func {
		return fmt.Errorf("conversion must be a function")
	}
	if typ.NumOut() != 1 {
		return fmt.Errorf("conversion must return a single value")
	}

	if !typ.Out(0).Implements(refValType) {
		return fmt.Errorf("conversion must return a value, must implement ref.Val")
	}

	if typ.NumIn() != 1 {
		return fmt.Errorf("conversion must accept a single value")
	}

	if typ.In(0).Implements(refValType) {
		return fmt.Errorf("conversion must accept a single value, must not implement ref.Val")
	}

	tp.conversions[typ.In(0)] = &convertType{
		targetType:  typ.Out(0),
		convertFunc: reflect.ValueOf(fun),
	}
	return nil
}

func (tp *nativeTypeProvider) registerNativeType(tagName string, rawType reflect.Type) (t Type, err error) {
	typeName := rawTypeName(rawType)
	if _, ok := tp.nativeTypes[typeName]; ok {
		return nil, fmt.Errorf("native type already registered: %v", typeName)
	}

	switch rawType.Kind() {
	case reflect.Struct:
		t, err = newStructType(tagName, rawType)
	}
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("unsupported native type: %v", rawType)
	}

	tp.nativeTypes[typeName] = t
	return t, nil
}

func (tp *nativeTypeProvider) registerType(refType any) (Type, error) {
	switch rt := refType.(type) {
	case reflect.Type:
		rawType := rt
		return tp.registerNativeType(tp.tagName, rawType)
	case reflect.Value:
		rawType := rt.Type()
		return tp.registerNativeType(tp.tagName, rawType)
	default:
		return nil, fmt.Errorf("unsupported native type: %v (%T) must be reflect.Type or reflect.Value", rt, rt)
	}
}

// EnumValue proxies to the types.Provider configured at the times the NativeTypes
// option was configured.
func (tp *nativeTypeProvider) EnumValue(enumName string) ref.Val {
	return tp.baseProvider.EnumValue(enumName)
}

// FindIdent looks up natives type instances by qualified identifier, and if not found
// proxies to the composed types.Provider.
func (tp *nativeTypeProvider) FindIdent(typeName string) (ref.Val, bool) {
	if t, found := tp.nativeTypes[typeName]; found {
		return t, true
	}
	return tp.baseProvider.FindIdent(typeName)
}

func (tp *nativeTypeProvider) findStructRawType(structType string) (reflect.Type, bool) {
	t, found := tp.nativeTypes[structType]
	if !found {
		return nil, false
	}
	rawType := t.GetRawType()

	return tp.toTargetType(rawType), true
}

func (tp *nativeTypeProvider) toTargetType(rawType reflect.Type) reflect.Type {
	convertInfo, ok := tp.conversions[rawType]
	if !ok {
		return rawType
	}
	return convertInfo.targetType
}

func (tp *nativeTypeProvider) toTargetValue(rawType reflect.Type, val reflect.Value) ([]reflect.Value, bool) {
	convertInfo, ok := tp.conversions[rawType]
	if !ok {
		return nil, false
	}

	// Convert the value using the conversion function.
	convertArgs := []reflect.Value{val}
	convertResult := convertInfo.convertFunc.Call(convertArgs)

	return convertResult, true
}

// FindStructType returns the Type give a qualified type name.
func (tp *nativeTypeProvider) FindStructType(structType string) (*types.Type, bool) {
	rawType, found := tp.findStructRawType(structType)
	if !found {
		return tp.baseProvider.FindStructType(structType)
	}
	if !isSupportedFieldType(rawType) {
		return nil, false
	}

	_, found = tp.nativeTypes[structType]
	if !found {
		return tp.baseProvider.FindStructType(structType)
	}

	return types.NewTypeTypeWithParam(getTypeValue(rawType)), true
}

// FindStructFieldType returns the field type for a checked type value. Returns
// false if the field could not be found.
func (tp *nativeTypeProvider) FindStructFieldType(structType, fieldName string) (*types.FieldType, bool) {
	rawType, found := tp.findStructRawType(structType)
	if !found {
		return tp.baseProvider.FindStructFieldType(structType, fieldName)
	}
	if !isSupportedFieldType(rawType) {
		return nil, false
	}

	fieldMap := getStructFieldMap(rawType, tp.tagName)
	if len(fieldMap) == 0 {
		return nil, false
	}

	fieldIndex, found := fieldMap[fieldName]
	if !found {
		return nil, false
	}

	fieldRawType := rawType.Field(fieldIndex)
	if !fieldRawType.IsExported() || !isSupportedType(fieldRawType.Type) {
		return nil, false
	}

	rawType = tp.toTargetType(fieldRawType.Type)
	typ := getTypeValue(rawType)

	ft := &types.FieldType{
		Type: typ,
		IsSet: func(obj any) bool {
			refVal := reflect.Indirect(reflect.ValueOf(obj))
			refField := refVal.Field(fieldIndex)
			return !refField.IsZero()
		},
	}

	getFrom := func(obj any) (reflect.Value, error) {
		if obj == nil {
			return reflect.Value{}, fmt.Errorf("failed to get field value: object is nil")
		}
		rawValue := reflect.ValueOf(obj)
		if rawValue.Kind() == reflect.Ptr {
			if rawValue.IsNil() {
				return reflect.Value{}, fmt.Errorf("failed to get field value: object is nil")
			}
			rawValue = rawValue.Elem()
		}

		refField := rawValue.Field(fieldIndex)
		return refField, nil
	}
	if rawType == fieldRawType.Type {
		ft.GetFrom = func(obj any) (any, error) {
			refField, err := getFrom(obj)
			if err != nil {
				return nil, err
			}
			return refField.Interface(), nil
		}
	} else {
		ft.GetFrom = func(obj any) (any, error) {
			refField, err := getFrom(obj)
			if err != nil {
				return nil, err
			}

			data, ok := tp.toTargetValue(fieldRawType.Type, refField)
			if !ok {
				return nil, fmt.Errorf("failed to convert field value")
			}
			return data[0].Interface(), nil
		}
	}

	return ft, true
}

// NewValue implements the types.Provider interface method.
func (tp *nativeTypeProvider) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	t, found := tp.nativeTypes[typeName]
	if !found {
		return tp.baseProvider.NewValue(typeName, fields)
	}
	refPtr := reflect.New(t.GetRawType())
	refVal := refPtr.Elem()

	fieldMap := getStructFieldMap(t.GetRawType(), tp.tagName)
	if len(fieldMap) == 0 {
		return tp.baseProvider.NewValue(typeName, fields)
	}

	for fieldName, val := range fields {
		fieldIndex, found := fieldMap[fieldName]
		if !found {
			return types.NewErr("no such field: %s", fieldName)
		}

		refFieldDef := t.GetRawType().Field(fieldIndex)
		if !refFieldDef.IsExported() || !isSupportedType(refFieldDef.Type) {
			return types.NewErr("no such field: %s", fieldName)
		}
		fieldVal, err := val.ConvertToNative(refFieldDef.Type)
		if err != nil {
			return types.NewErr(err.Error())
		}
		refField := refVal.FieldByIndex(refFieldDef.Index)
		refFieldVal := reflect.ValueOf(fieldVal)
		refField.Set(refFieldVal)
	}
	return tp.NativeToValue(refPtr.Interface())
}

// NativeToValue adapts native values to CEL values and will proxy to the composed type adapter
// for non-native types.
func (tp *nativeTypeProvider) NativeToValue(val any) ref.Val {
	if val == nil {
		return types.NullValue
	}
	if v, ok := val.(ref.Val); ok {
		return v
	}
	rawVal := reflect.ValueOf(val)

	out, ok := tp.toTargetValue(rawVal.Type(), rawVal)
	if ok {
		return out[0].Interface().(ref.Val)
	}

	// This isn't quite right if you're also supporting proto,
	// but maybe an acceptable limitation.
	switch rawVal.Kind() {
	case reflect.Array, reflect.Slice:
		switch val := val.(type) {
		case []byte:
			return tp.baseAdapter.NativeToValue(val)
		default:
			return newListObject(tp, val)
		}
	case reflect.Map:
		return newMapObject(tp, val)
	case reflect.Struct:
		switch val := val.(type) {
		case time.Time:
			return tp.baseAdapter.NativeToValue(val)
		default:
			return newStructObject(tp, tp.tagName, val, rawVal)
		}
	case reflect.Pointer:
		if rawVal.IsNil() {
			return types.NullValue
		}
		return tp.NativeToValue(rawVal.Elem().Interface())
	default:
		return tp.baseAdapter.NativeToValue(val)
	}
}
