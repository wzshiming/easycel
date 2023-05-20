package easycel

import (
	"reflect"
	"time"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/traits"
)

var (
	timestampType         = reflect.TypeOf(time.Now())
	durationType          = reflect.TypeOf(time.Nanosecond)
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

	traitMapperType = reflect.TypeOf((*traits.Mapper)(nil)).Elem()
	traitListerType = reflect.TypeOf((*traits.Lister)(nil)).Elem()
)

var typeValueMap = map[reflect.Type]*types.TypeValue{}

func GetTypeValue(typ reflect.Type) *types.TypeValue {
	if refType, ok := typeValueMap[typ]; ok {
		return refType
	}

	refType := types.NewTypeValue(typeName(typ), GetTrait(typ))
	typeValueMap[typ] = refType
	return refType
}

var typeTraitMap = map[reflect.Type]int{}

func GetTrait(typ reflect.Type) int {
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

func typeName(typ reflect.Type) string {
	return typ.String()
}
