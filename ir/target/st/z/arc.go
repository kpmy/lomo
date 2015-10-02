package z

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"github.com/kpmy/lomo/ir"
	"github.com/kpmy/lomo/ir/target"
	"github.com/kpmy/lomo/ir/target/st"
	"github.com/kpmy/ypk/assert"
	"io"
	"time"
)

const CODE = "code"
const VERSION = "version"

type Version struct {
	Generator float64
	Code      int64
}

type impl struct {
	base target.Target
}

func (i *impl) OldDef(rd io.Reader) (f ir.ForeignType) {
	return i.base.OldDef(rd)
}

func (i *impl) OldCode(rd io.Reader) (u *ir.Unit) {
	buf := bytes.NewBuffer(nil)
	len, _ := io.Copy(buf, rd)
	if r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), len); err == nil {
		for _, f := range r.File {
			if f.Name == VERSION {
				rd, _ := f.Open()
				ver := &Version{}
				buf := bytes.NewBuffer(nil)
				io.Copy(buf, rd)
				xml.Unmarshal(buf.Bytes(), ver)
				assert.For(ver.Generator == st.VERSION, 40, "incompatible code version")
			}
			if f.Name == CODE {
				rd, _ := f.Open()
				u = i.base.OldCode(rd)
				//data, _ = ioutil.ReadAll(r)
			}
		}
	}
	return
}

func (i *impl) NewDef(u ir.ForeignType, wr io.Writer) {
	i.base.NewDef(u, wr)
}

func (i *impl) NewCode(u *ir.Unit, wr io.Writer) {
	zw := zip.NewWriter(wr)
	if wr, err := zw.Create(CODE); err == nil {
		buf := bytes.NewBuffer(nil)
		i.base.NewCode(u, buf)
		io.Copy(wr, buf)
	}
	if wr, err := zw.Create(VERSION); err == nil {
		ver := &Version{}
		ver.Generator = st.VERSION
		ver.Code = time.Now().Unix()
		data, _ := xml.Marshal(ver)
		buf := bytes.NewBuffer(data)
		io.Copy(wr, buf)
	}
	zw.Close()
}

func init() {
	i := st.Init()
	target.Impl = &impl{base: i}
}
