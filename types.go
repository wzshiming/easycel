package easycel

import (
	"reflect"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

var (
	timestampType         = reflect.TypeOf(time.Now())
	typesTimestampType    = reflect.TypeOf(types.Timestamp{})
	durationType          = reflect.TypeOf(time.Nanosecond)
	typesDurationType     = reflect.TypeOf(types.Duration{})
	byteType              = reflect.TypeOf(byte(0))
	errorType             = reflect.TypeOf((*error)(nil)).Elem()
	traitsAdderType       = reflect.TypeOf((*traits.Adder)(nil)).Elem()
	traitsComparerType    = reflect.TypeOf((*traits.Comparer)(nil)).Elem()
	traitsContainerType   = reflect.TypeOf((*traits.Container)(nil)).Elem()
	traitsDividerType     = reflect.TypeOf((*traits.Divider)(nil)).Elem()
	traitsFieldTesterType = reflect.TypeOf((*traits.FieldTester)(nil)).Elem()
	traitsIndexerType     = reflect.TypeOf((*traits.Indexer)(nil)).Elem()
	traitsIterableType    = reflect.TypeOf((*traits.Iterable)(nil)).Elem()
	traitsIteratorType    = reflect.TypeOf((*traits.Iterator)(nil)).Elem()
	traitsMatcherType     = reflect.TypeOf((*traits.Matcher)(nil)).Elem()
	traitsModderType      = reflect.TypeOf((*traits.Modder)(nil)).Elem()
	traitsMultiplierType  = reflect.TypeOf((*traits.Multiplier)(nil)).Elem()
	traitsNegatorType     = reflect.TypeOf((*traits.Negater)(nil)).Elem()
	traitsReceiverType    = reflect.TypeOf((*traits.Receiver)(nil)).Elem()
	traitsSizerType       = reflect.TypeOf((*traits.Sizer)(nil)).Elem()
	traitsSubtractorType  = reflect.TypeOf((*traits.Subtractor)(nil)).Elem()
	refValType            = reflect.TypeOf((*ref.Val)(nil)).Elem()
)

var typeValueMap = map[reflect.Type]*types.Type{}

func getTypeValue(rawType reflect.Type) (typ *types.Type) {
	typ, ok := typeValueMap[rawType]
	if ok {
		return typ
	}

	switch rawType.Kind() {
	case reflect.Struct:
		typ = cel.ObjectType(rawTypeName(rawType), getTrait(rawType))
	case reflect.Bool:
		typ = cel.BoolType
	case reflect.Float32, reflect.Float64:
		typ = cel.DoubleType
	case reflect.Int64:
		if rawType == durationType || rawType == typesDurationType {
			typ = cel.DurationType
		} else {
			typ = cel.IntType
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		typ = cel.IntType
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		typ = cel.UintType
	case reflect.String:
		typ = cel.StringType
	case reflect.Slice:
		rawElem := rawType.Elem()
		if rawElem == byteType {
			typ = cel.BytesType
		} else {
			typ = cel.ListType(getTypeValue(rawElem))
		}
	case reflect.Array:
		typ = cel.ListType(getTypeValue(rawType.Elem()))
	case reflect.Map:
		typ = cel.MapType(getTypeValue(rawType.Key()), getTypeValue(rawType.Elem()))
	case reflect.Ptr:
		typ = cel.NullableType(getTypeValue(rawType.Elem()))
	}

	typeValueMap[rawType] = typ
	return typ
}

var typeTraitMap = map[reflect.Type]int{}

func getTrait(typ reflect.Type) int {
	trait, ok := typeTraitMap[typ]
	if ok {
		return trait
	}

	if typ.Implements(traitsAdderType) {
		trait |= traits.AdderType
	}
	if typ.Implements(traitsComparerType) {
		trait |= traits.ComparerType
	}
	if typ.Implements(traitsContainerType) {
		trait |= traits.ContainerType
	}
	if typ.Implements(traitsDividerType) {
		trait |= traits.DividerType
	}
	if typ.Implements(traitsFieldTesterType) {
		trait |= traits.FieldTesterType
	}
	if typ.Implements(traitsIndexerType) {
		trait |= traits.IndexerType
	}
	if typ.Implements(traitsIterableType) {
		trait |= traits.IterableType
	}
	if typ.Implements(traitsIteratorType) {
		trait |= traits.IteratorType
	}
	if typ.Implements(traitsMatcherType) {
		trait |= traits.MatcherType
	}
	if typ.Implements(traitsModderType) {
		trait |= traits.ModderType
	}
	if typ.Implements(traitsMultiplierType) {
		trait |= traits.MultiplierType
	}
	if typ.Implements(traitsNegatorType) {
		trait |= traits.NegatorType
	}
	if typ.Implements(traitsReceiverType) {
		trait |= traits.ReceiverType
	}
	if typ.Implements(traitsSizerType) {
		trait |= traits.SizerType
	}
	if typ.Implements(traitsSubtractorType) {
		trait |= traits.SubtractorType
	}
	typeTraitMap[typ] = trait
	return trait
}

func rawTypeName(rawType reflect.Type) string {
	switch rawType {
	case typesTimestampType, timestampType:
		return "google.protobuf.Timestamp"
	case typesDurationType, durationType:
		return "google.protobuf.Duration"
	}
	return rawType.String()
}
