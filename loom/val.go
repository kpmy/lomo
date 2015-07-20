package loom

import (
	"lomo/ir/types"
)

type value struct {
	typ types.Type
	val interface{}
}
