package target

import (
	"io"
)

import (
	"github.com/kpmy/lomo/ir"
)

type Target interface {
	OldDef(io.Reader) ir.ForeignType
	OldCode(io.Reader) *ir.Unit
	NewDef(ir.ForeignType, io.Writer)
	NewCode(*ir.Unit, io.Writer)
}

var Impl Target
