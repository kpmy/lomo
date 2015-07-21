package st

import (
	"bytes"
	"encoding/xml"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"io"
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/ir/target"
	"lomo/ir/types"
	"reflect"
	"strconv"
)

//dynamic xml marshaller
type extern struct {
	x       interface{}
	shallow bool
}

func (u *extern) attr(start *xml.StartElement, name string, value interface{}) {
	str := func(value string) {
		assert.For(value != "", 20)
		a := xml.Attr{}
		a.Name.Local = name
		a.Value = value
		start.Attr = append(start.Attr, a)
	}
	switch v := value.(type) {
	case string:
		str(v)
	case bool:
		if v {
			str("true")
		} else {
			str("false")
		}
	case types.Type:
		str(v.String())
	default:
		halt.As(100, v)
	}
}

func (u *extern) data(t types.Type, _x interface{}) (ret xml.CharData) {
	switch x := _x.(type) {
	case string:
		return xml.CharData(x)
	default:
		halt.As(100, reflect.TypeOf(x))
	}
	return
}

func (u *extern) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	switch x := u.x.(type) {
	case *ir.Unit:
		start.Name.Local = "unit"
		u.attr(&start, "name", x.Name)
		err = e.EncodeToken(start)
		for _, v := range x.Variables {
			n := &extern{x: v}
			e.Encode(n)
		}
		err = e.EncodeToken(start.End())
	case ir.ForeignType:
		start.Name.Local = "definition"
		u.attr(&start, "name", x.Name())
		err = e.EncodeToken(start)
		for _, v := range x.Variables() {
			n := &extern{x: v, shallow: true}
			e.Encode(n)
		}
		err = e.EncodeToken(start.End())
	case *ir.Variable:
		switch x.Modifier {
		case mods.IN:
			start.Name.Local = "in"
		case mods.OUT:
			start.Name.Local = "out"
		case mods.REG:
			start.Name.Local = "reg"
		default:
			start.Name.Local = "var"
		}
		u.attr(&start, "name", x.Name)
		u.attr(&start, "builtin", x.Type.Basic)
		if x.Type.Basic {
			u.attr(&start, "type", x.Type.Builtin.Code.String())
		} else {
			u.attr(&start, "type", x.Type.Foreign.Name())
		}
		e.EncodeToken(start)
		if x.Type.Basic && !u.shallow {
			if r := x.Unit.Rules[x.Name]; r != nil {
				n := &extern{x: r}
				e.Encode(n)
			}
		} else if !u.shallow {
			if rr := x.Unit.ForeignRules[x.Name]; rr != nil {
				for k, v := range rr {
					rs := xml.StartElement{}
					rs.Name.Local = "foreign"
					u.attr(&rs, "id", k)
					e.EncodeToken(rs)
					n := &extern{x: v}
					e.Encode(n)
					e.EncodeToken(rs.End())
				}
			}
		}
		e.EncodeToken(start.End())
	case *ir.ConditionalRule:
		start.Name.Local = "select"
		e.EncodeToken(start)
		for _, _ = range x.Blocks {
			panic(0)
		}
		n := &extern{x: x.Default}
		e.Encode(n)
		e.EncodeToken(start.End())
	case *ir.ConstExpr:
		start.Name.Local = "constant"
		u.attr(&start, "type", x.Type)
		e.EncodeToken(start)
		e.EncodeToken(u.data(x.Type, x.Value))
		e.EncodeToken(start.End())
	case *ir.SelectExpr:
		start.Name.Local = "selector"
		u.attr(&start, "unit", x.Var.Unit.Name)
		u.attr(&start, "id", x.Var.Name)
		e.EncodeToken(start)
		e.EncodeToken(start.End())
	default:
		halt.As(100, reflect.TypeOf(x))
	}
	return
}

type intern struct {
	root    *ir.Unit
	x       interface{}
	consume func(interface{})
}

type futureForeignType struct {
	name     string
	fakeUnit *ir.Unit
}

func (f *futureForeignType) Name() string { return f.name }

func (f *futureForeignType) Variables() map[string]*ir.Variable { return f.fakeUnit.Variables }

func (i *intern) attr(start *xml.StartElement, name string) (ret interface{}) {
	for _, x := range start.Attr {
		if x.Name.Local == name {
			switch x.Value {
			case "true", "false":
				ret = (x.Value == "true")
			default:
				ret = x.Value
			}
			break
		}
	}
	return
}
func (i *intern) data(t types.Type, cd xml.CharData) (ret interface{}) {
	switch t {
	case types.INTEGER:
		x, err := strconv.Atoi(string(cd))
		assert.For(err == nil, 60)
		ret = x
	default:
		halt.As(100, t)
	}
	return
}

func (i *intern) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	var consumer func(interface{})
	switch start.Name.Local {
	case "unit":
		u := ir.NewUnit(i.attr(&start, "name").(string))
		i.x = u
		i.root = u
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case *ir.Variable:
				u.Variables[x.Name] = x
				x.Unit = u
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "definition":
		f := &futureForeignType{}
		f.name = i.attr(&start, "name").(string)
		i.x = f
		f.fakeUnit = ir.NewUnit(f.name)
		i.root = f.fakeUnit
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case *ir.Variable:
				f.fakeUnit.Variables[x.Name] = x
				x.Unit = f.fakeUnit
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "in", "var", "reg", "out":
		v := &ir.Variable{}
		v.Name = i.attr(&start, "name").(string)
		switch start.Name.Local {
		case "in":
			v.Modifier = mods.IN
		case "var":
			v.Modifier = mods.NONE
		case "reg":
			v.Modifier = mods.REG
		case "out":
			v.Modifier = mods.OUT
		default:
			halt.As(100, start.Name.Local)
		}
		v.Type.Basic = i.attr(&start, "builtin").(bool)
		i.x = v
		i.consume(v)
		if v.Type.Basic {
			v.Type.Builtin = &ir.BuiltinType{}
			v.Type.Builtin.Code = types.TypMap[i.attr(&start, "type").(string)]
			consumer = func(_x interface{}) {
				switch x := _x.(type) {
				case ir.Rule:
					v.Unit.Rules[v.Name] = x
				default:
					halt.As(100, reflect.TypeOf(x))
				}
			}
		} else {
			ff := &futureForeignType{}
			ff.name = i.attr(&start, "type").(string)
			ff.fakeUnit = ir.NewUnit(ff.name)
			v.Type.Foreign = ff
			rr := make(map[string]ir.Rule)
			v.Unit.ForeignRules[v.Name] = rr
			consumer = func(_x interface{}) {
				switch x := _x.(type) {
				case map[string]ir.Rule:
					for k, v := range x {
						rr[k] = v
					}
				default:
					halt.As(100, reflect.TypeOf(x))
				}
			}
		}
	case "foreign": //wrapper for local rules of foreign objects
		id := i.attr(&start, "id").(string)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Rule:
				fn := make(map[string]ir.Rule)
				fn[id] = x
				i.consume(fn)
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "select":
		r := &ir.ConditionalRule{}
		i.x = r
		i.consume(r)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				r.Default = x
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "constant":
		c := &ir.ConstExpr{}
		c.Type = types.TypMap[i.attr(&start, "type").(string)]
		sd, _ := d.Token()
		c.Value = i.data(c.Type, sd.(xml.CharData))
		i.consume(c)
		i.x = c
	case "selector":
		c := &ir.SelectExpr{}
		if un := i.attr(&start, "unit").(string); un == i.root.Name {
			go func(id string) {
				for _, ok := i.root.Variables[id]; !ok; {
				}
				c.Var = i.root.Variables[id]
				assert.For(c.Var != nil, 60)
			}(i.attr(&start, "id").(string))
		} else {
			halt.As(100, un)
		}
		i.x = c
		i.consume(c)
	default:
		halt.As(100, start.Name.Local)
	}
	var _t xml.Token
	for stop := false; !stop && err == nil; {
		_t, err = d.Token()
		switch t := _t.(type) {
		case xml.StartElement:
			x := &intern{root: i.root}
			x.consume = consumer
			d.DecodeElement(x, &t)
		case xml.EndElement:
			stop = t.Name == start.Name
		default:
			halt.As(100, reflect.TypeOf(t), t)
		}
	}
	return
}

type impl struct{}

func (i *impl) OldDef(rd io.Reader) (f ir.ForeignType) {
	it := &intern{}
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, rd)
	if err := xml.Unmarshal(buf.Bytes(), it); err == nil {
		f, _ = it.x.(ir.ForeignType)
	} else {
		halt.As(100, err)
	}
	return
}

func (i *impl) OldCode(rd io.Reader) (u *ir.Unit) {
	it := &intern{}
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, rd)
	if err := xml.Unmarshal(buf.Bytes(), it); err == nil {
		u, _ = it.x.(*ir.Unit)
	} else {
		halt.As(100, err)
	}
	return
}

func (i *impl) NewDef(u ir.ForeignType, wr io.Writer) {
	e := &extern{x: u}
	if data, err := xml.Marshal(e); err == nil {
		wr.Write([]byte(xml.Header))
		io.Copy(wr, bytes.NewBuffer(data))
	} else {
		halt.As(100, err)
	}
}

func (i *impl) NewCode(u *ir.Unit, wr io.Writer) {
	e := &extern{x: u}
	if data, err := xml.Marshal(e); err == nil {
		wr.Write([]byte(xml.Header))
		io.Copy(wr, bytes.NewBuffer(data))
	} else {
		halt.As(100, err)
	}
}

func init() {
	target.Impl = &impl{}
}
