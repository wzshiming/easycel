package easycel

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

var (
	nativeObjTraitMask = traits.FieldTesterType | traits.IndexerType
)

func newNativeTypeProvider(tagName string, adapter ref.TypeAdapter, provider ref.TypeProvider) *nativeTypeProvider {
	return &nativeTypeProvider{
		tagName:      tagName,
		conversions:  make(map[reflect.Type]*convertType),
		nativeTypes:  make(map[string]*nativeType),
		baseAdapter:  adapter,
		baseProvider: provider,
	}
}

type nativeTypeProvider struct {
	tagName      string
	conversions  map[reflect.Type]*convertType
	nativeTypes  map[string]*nativeType
	baseAdapter  ref.TypeAdapter
	baseProvider ref.TypeProvider
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

func (tp *nativeTypeProvider) registerType(refType any) error {
	switch rt := refType.(type) {
	case reflect.Type:
		t, err := newNativeType(tp.tagName, rt)
		if err != nil {
			return err
		}
		tp.nativeTypes[t.TypeName()] = t
	case reflect.Value:
		t, err := newNativeType(tp.tagName, rt.Type())
		if err != nil {
			return err
		}
		tp.nativeTypes[t.TypeName()] = t
	default:
		return fmt.Errorf("unsupported native type: %v (%T) must be reflect.Type or reflect.Value", rt, rt)
	}
	return nil
}

// EnumValue proxies to the ref.TypeProvider configured at the times the NativeTypes
// option was configured.
func (tp *nativeTypeProvider) EnumValue(enumName string) ref.Val {
	return tp.baseProvider.EnumValue(enumName)
}

// FindIdent looks up natives type instances by qualified identifier, and if not found
// proxies to the composed ref.TypeProvider.
func (tp *nativeTypeProvider) FindIdent(typeName string) (ref.Val, bool) {
	if t, found := tp.nativeTypes[typeName]; found {
		return t, true
	}
	return tp.baseProvider.FindIdent(typeName)
}

// FindType looks up CEL type-checker type definition by qualified identifier, and if not found
// proxies to the composed ref.TypeProvider.
func (tp *nativeTypeProvider) FindType(typeName string) (*exprpb.Type, bool) {
	if _, found := tp.nativeTypes[typeName]; found {
		return decls.NewTypeType(decls.NewObjectType(typeName)), true
	}
	return tp.baseProvider.FindType(typeName)
}

// FindFieldType looks up a native type's field definition, and if the type name is not a native
// type then proxies to the composed ref.TypeProvider
func (tp *nativeTypeProvider) FindFieldType(typeName, fieldName string) (*ref.FieldType, bool) {
	t, found := tp.nativeTypes[typeName]
	if !found {
		return tp.baseProvider.FindFieldType(typeName, fieldName)
	}
	fieldMap := getStructFieldMap(t.refType, tp.tagName)
	if len(fieldMap) == 0 {
		return nil, false
	}

	fieldIndex, found := fieldMap[fieldName]
	if !found {
		return nil, false
	}

	refField, isDefined := t.hasField(fieldIndex)
	if !found || !isDefined {
		return nil, false
	}
	convertInfo, ok := tp.conversions[refField.Type]
	if ok {
		exprType, ok := convertToExprType(convertInfo.targetType)
		if !ok {
			return nil, false
		}
		return &ref.FieldType{
			Type: exprType,
			IsSet: func(obj any) bool {
				refVal := reflect.Indirect(reflect.ValueOf(obj))
				refField := refVal.Field(fieldIndex)
				return !refField.IsZero()
			},
			GetFrom: func(obj any) (any, error) {
				refVal := reflect.Indirect(reflect.ValueOf(obj))
				refField := refVal.Field(fieldIndex)
				out := convertInfo.convertFunc.Call([]reflect.Value{refField})
				return getFieldValue(tp, out[0]), nil
			},
		}, true
	}
	exprType, ok := convertToExprType(refField.Type)
	if !ok {
		return nil, false
	}
	return &ref.FieldType{
		Type: exprType,
		IsSet: func(obj any) bool {
			refVal := reflect.Indirect(reflect.ValueOf(obj))
			refField := refVal.Field(fieldIndex)
			return !refField.IsZero()
		},
		GetFrom: func(obj any) (any, error) {
			refVal := reflect.Indirect(reflect.ValueOf(obj))
			refField := refVal.Field(fieldIndex)
			return getFieldValue(tp, refField), nil
		},
	}, true
}

// NewValue implements the ref.TypeProvider interface method.
func (tp *nativeTypeProvider) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	t, found := tp.nativeTypes[typeName]
	if !found {
		return tp.baseProvider.NewValue(typeName, fields)
	}
	refPtr := reflect.New(t.refType)
	refVal := refPtr.Elem()

	fieldMap := getStructFieldMap(t.refType, tp.tagName)
	if len(fieldMap) == 0 {
		return tp.baseProvider.NewValue(typeName, fields)
	}

	for fieldName, val := range fields {
		fieldIndex, found := fieldMap[fieldName]
		if !found {
			return types.NewErr("no such field: %s", fieldName)
		}
		refFieldDef, isDefined := t.hasField(fieldIndex)
		if !isDefined {
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

	convertInfo, ok := tp.conversions[rawVal.Type()]
	if ok {
		out := convertInfo.convertFunc.Call([]reflect.Value{rawVal})
		return out[0].Interface().(ref.Val)
	}
	refVal := rawVal
	if refVal.Kind() == reflect.Ptr {
		refVal = reflect.Indirect(refVal)
	}
	// This isn't quite right if you're also supporting proto,
	// but maybe an acceptable limitation.
	switch refVal.Kind() {
	case reflect.Array, reflect.Slice:
		switch val := val.(type) {
		case []byte:
			return tp.baseAdapter.NativeToValue(val)
		default:
			return types.NewDynamicList(tp, val)
		}
	case reflect.Map:
		return types.NewDynamicMap(tp, val)
	case reflect.Struct:
		switch val := val.(type) {
		case time.Time:
			return tp.baseAdapter.NativeToValue(val)
		default:
			return newNativeObject(tp, tp.tagName, val, rawVal)
		}
	default:
		return tp.baseAdapter.NativeToValue(val)
	}
}

func newNativeObject(adapter ref.TypeAdapter, tagName string, val any, refValue reflect.Value) ref.Val {
	valType, err := newNativeType(tagName, refValue.Type())
	if err != nil {
		return types.WrapErr(err)
	}
	return &nativeObj{
		TypeAdapter: adapter,
		val:         val,
		valType:     valType,
		refValue:    refValue,
	}
}

type nativeObj struct {
	ref.TypeAdapter
	val      any
	valType  *nativeType
	refValue reflect.Value
}

// ConvertToNative implements the ref.Val interface method.
func (o *nativeObj) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if o.refValue.Type() == typeDesc {
		return o.val, nil
	}
	if o.refValue.Kind() == reflect.Pointer && o.refValue.Type().Elem() == typeDesc {
		return o.refValue.Elem().Interface(), nil
	}
	if typeDesc.Kind() == reflect.Pointer && o.refValue.Type() == typeDesc.Elem() {
		ptr := reflect.New(typeDesc.Elem())
		ptr.Elem().Set(o.refValue)
		return ptr.Interface(), nil
	}
	return nil, fmt.Errorf("type conversion error from '%v' to '%v'", o.Type(), typeDesc)
}

// ConvertToType implements the ref.Val interface method.
func (o *nativeObj) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case types.TypeType:
		return o.valType
	default:
		if typeVal.TypeName() == o.valType.typeName {
			return o
		}
	}
	return types.NewErr("type conversion error from '%s' to '%s'", o.Type(), typeVal)
}

// Equal implements the ref.Val interface method.
func (o *nativeObj) Equal(other ref.Val) ref.Val {
	otherNtv, ok := other.(*nativeObj)
	if !ok {
		return types.False
	}
	val := o.val
	otherVal := otherNtv.val
	refVal := o.refValue
	otherRefVal := otherNtv.refValue
	if refVal.Kind() != otherRefVal.Kind() {
		if refVal.Kind() == reflect.Pointer {
			val = refVal.Elem().Interface()
		} else if otherRefVal.Kind() == reflect.Pointer {
			otherVal = otherRefVal.Elem().Interface()
		}
	}
	return types.Bool(reflect.DeepEqual(val, otherVal))
}

// IsZeroValue indicates whether the contained Golang value is a zero value.
func (o *nativeObj) IsZeroValue() bool {
	return reflect.Indirect(o.refValue).IsZero()
}

// IsSet tests whether a field which is defined is set to a non-default value.
func (o *nativeObj) IsSet(field ref.Val) ref.Val {
	refField, refErr := o.getReflectedField(field)
	if refErr != nil {
		return refErr
	}
	return types.Bool(!refField.IsZero())
}

// Get returns the value fo a field name.
func (o *nativeObj) Get(field ref.Val) ref.Val {
	refField, refErr := o.getReflectedField(field)
	if refErr != nil {
		return refErr
	}
	return adaptFieldValue(o, refField)
}

func (o *nativeObj) getReflectedField(field ref.Val) (reflect.Value, ref.Val) {
	fieldName, ok := field.(types.String)
	if !ok {
		return reflect.Value{}, types.MaybeNoSuchOverloadErr(field)
	}
	fieldMap := getStructFieldMap(o.refValue.Type(), o.valType.tagName)
	if len(fieldMap) == 0 {
		return reflect.Value{}, types.NewErr("no such field: %s", fieldName)
	}

	fieldNameStr, found := fieldMap[string(fieldName)]
	if !found {
		return reflect.Value{}, types.NewErr("no such field: %s", fieldName)
	}

	refField, isDefined := o.valType.hasField(fieldNameStr)
	if !isDefined {
		return reflect.Value{}, types.NewErr("no such field: %s", fieldName)
	}
	refVal := reflect.Indirect(o.refValue)
	return refVal.FieldByIndex(refField.Index), nil
}

// Type implements the ref.Val interface method.
func (o *nativeObj) Type() ref.Type {
	return o.valType
}

// Value implements the ref.Val interface method.
func (o *nativeObj) Value() any {
	return o.val
}

func newNativeType(tagName string, rawType reflect.Type) (*nativeType, error) {
	refType := rawType
	if refType.Kind() == reflect.Pointer {
		refType = refType.Elem()
	}
	if !isValidObjectType(refType) {
		return nil, fmt.Errorf("unsupported reflect.Type %v, must be reflect.Struct", rawType)
	}
	return &nativeType{
		tagName:  tagName,
		typeName: typeName(refType),
		refType:  refType,
	}, nil
}

type nativeType struct {
	tagName  string
	typeName string
	refType  reflect.Type
}

// ConvertToNative implements ref.Val.ConvertToNative.
func (t *nativeType) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("type conversion error for type to '%v'", typeDesc)
}

// ConvertToType implements ref.Val.ConvertToType.
func (t *nativeType) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case types.TypeType:
		return types.TypeType
	}
	return types.NewErr("type conversion error from '%s' to '%s'", types.TypeType, typeVal)
}

// Equal returns true of both type names are equal to each other.
func (t *nativeType) Equal(other ref.Val) ref.Val {
	otherType, ok := other.(ref.Type)
	return types.Bool(ok && t.TypeName() == otherType.TypeName())
}

// HasTrait implements the ref.Type interface method.
func (t *nativeType) HasTrait(trait int) bool {
	return nativeObjTraitMask&trait == trait
}

// String implements the strings.Stringer interface method.
func (t *nativeType) String() string {
	return t.typeName
}

// Type implements the ref.Val interface method.
func (t *nativeType) Type() ref.Type {
	return types.TypeType
}

// TypeName implements the ref.Type interface method.
func (t *nativeType) TypeName() string {
	return t.typeName
}

// Value implements the ref.Val interface method.
func (t *nativeType) Value() any {
	return t.typeName
}

// hasField returns whether a field name has a corresponding Golang reflect.StructField
func (t *nativeType) hasField(fieldIndex int) (reflect.StructField, bool) {
	f := t.refType.Field(fieldIndex)
	if !f.IsExported() || !isSupportedType(f.Type) {
		return reflect.StructField{}, false
	}
	return f, true
}

func adaptFieldValue(adapter ref.TypeAdapter, refField reflect.Value) ref.Val {
	return adapter.NativeToValue(getFieldValue(adapter, refField))
}

func isValidObjectType(refType reflect.Type) bool {
	return refType.Kind() == reflect.Struct
}

func isSupportedType(refType reflect.Type) bool {
	switch refType.Kind() {
	case reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Func, reflect.UnsafePointer, reflect.Uintptr:
		return false
	case reflect.Array, reflect.Slice:
		return isSupportedType(refType.Elem())
	case reflect.Map:
		return isSupportedType(refType.Key()) && isSupportedType(refType.Elem())
	}
	return true
}

var structFieldMap = map[string]map[reflect.Type]map[string]int{}

func getStructFieldMap(typ reflect.Type, tagName string) map[string]int {
	if filedMap, ok := structFieldMap[tagName]; ok {
		if entries, ok := filedMap[typ]; ok {
			return entries
		}
	}
	entries := map[string]int{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		name := ""
		if tagName == "" {
			name = field.Name
		} else {
			var ok bool
			name, ok = fieldNameWithTag(field, tagName)
			if !ok {
				continue
			}
		}
		entries[name] = i
	}

	if _, ok := structFieldMap[tagName]; !ok {
		structFieldMap[tagName] = map[reflect.Type]map[string]int{}
	}

	structFieldMap[tagName][typ] = entries
	return entries
}
