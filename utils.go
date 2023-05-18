package easycel

import (
	"reflect"
	"strings"
	"time"
)

func getUniqTypeName(name string, fun reflect.Type) string {
	k := fun.String()
	k = strings.ReplaceAll(k, " ", "__")
	k = strings.ReplaceAll(k, ".", "_")
	return name + "___" + k
}

var (
	timestampType = reflect.TypeOf(time.Now())
	durationType  = reflect.TypeOf(time.Nanosecond)
	byteType      = reflect.TypeOf(byte(0))
	errType       = func() reflect.Type {
		var r error
		return reflect.TypeOf(&r).Elem()
	}()
)
