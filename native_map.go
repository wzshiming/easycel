package easycel

import (
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func newMapObject(adapter types.Adapter, value any) ref.Val {
	switch v := value.(type) {
	case map[string]string:
		return types.NewStringStringMap(adapter, v)
	case map[string]any:
		return types.NewStringInterfaceMap(adapter, v)
	case map[ref.Val]ref.Val:
		return types.NewRefValMap(adapter, v)
	}
	return types.NewDynamicMap(adapter, value)
}
