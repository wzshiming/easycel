package easycel

import (
	"reflect"
	"strings"

	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

func NewObject(typ reflect.Type) *exprpb.Type {
	return decls.NewObjectType(getUniqTypeName("object", typ))
}

func getUniqTypeName(name string, fun reflect.Type) string {
	k := fun.String()
	k = strings.ReplaceAll(k, " ", "__")
	k = strings.ReplaceAll(k, ".", "_")
	return name + "___" + k
}

var (
	errType = func() reflect.Type {
		var r error
		return reflect.TypeOf(&r).Elem()
	}()
	celVal = func() reflect.Type {
		var r ref.Val
		return reflect.TypeOf(&r).Elem()
	}()
)
