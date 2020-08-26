package easycel

import (
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type CelVal interface {
	CelVal() ref.Val
}

type PbType interface {
	PbType() *exprpb.Type
}

type CelMethods interface {
	CelMethods() []string
}
