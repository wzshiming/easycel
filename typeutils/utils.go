package typeutils

import (
	"reflect"
)

var structFieldMap = map[string]map[reflect.Type]map[string]string{}

func GetStructFieldMap(typ reflect.Type, tagName string) map[string]string {
	if filedMap, ok := structFieldMap[tagName]; ok {
		if entries, ok := filedMap[typ]; ok {
			return entries
		}
	}
	entries := map[string]string{}
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

		entries[name] = field.Name
	}

	if _, ok := structFieldMap[tagName]; !ok {
		structFieldMap[tagName] = map[reflect.Type]map[string]string{}
	}

	structFieldMap[tagName][typ] = entries
	return entries
}
