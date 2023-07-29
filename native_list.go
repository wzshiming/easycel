package easycel

import (
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func newListObject(adapter types.Adapter, value any) ref.Val {
	switch v := value.(type) {
	case []string:
		return types.NewStringList(adapter, v)
	case []ref.Val:
		return types.NewDynamicList(adapter, v)
	}
	return types.NewDynamicList(adapter, value)
}
