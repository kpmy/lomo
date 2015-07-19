package st

import (
	"io"
	"lomo/ir"
	"lomo/ir/target"
)

type impl struct{}

func (i *impl) OldDef(io.Reader) *ir.ForeignType  { return nil }
func (i *impl) OldCode(io.Reader) *ir.Unit        { return nil }
func (i *impl) NewDef(*ir.ForeignType, io.Writer) {}
func (i *impl) NewCode(*ir.Unit, io.Writer)       {}

func init() {
	target.Impl = &impl{}
}
