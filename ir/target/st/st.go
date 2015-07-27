package st

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"github.com/kpmy/trigo"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"io"
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/ir/ops"
	"lomo/ir/target"
	"lomo/ir/types"
	"reflect"
	"strconv"
	"strings"
)

const VERSION = 0.1

const prefix = "base64:"

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
	case int:
		str(strconv.Itoa(v))
	case types.Type:
		str(v.String())
	default:
		halt.As(100, reflect.TypeOf(v), v)
	}
}

func (u *extern) data(t types.Type, _x interface{}) (ret xml.CharData) {
	switch x := _x.(type) {
	case string:
		if t == types.STRING {
			ret = xml.CharData(base64.StdEncoding.EncodeToString([]byte(prefix + x)))
		} else {
			ret = xml.CharData(x)
		}
	case bool:
		if x {
			ret = xml.CharData("true")
		} else {
			ret = xml.CharData("false")
		}
	case nil:
		ret = xml.CharData("null")
	case int32:
		ret = xml.CharData(strconv.FormatUint(uint64(x), 16))
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
		for _, c := range x.Const {
			n := &extern{x: c}
			e.Encode(n)
		}
		for _, v := range x.Variables {
			n := &extern{x: v}
			e.Encode(n)
		}
		if len(x.Infix) > 0 {
			inf := xml.StartElement{}
			inf.Name.Local = "infix"
			u.attr(&inf, "num", len(x.Infix))
			for i, v := range x.Infix {
				u.attr(&inf, "arg"+strconv.Itoa(i), v.Name)
			}
			e.EncodeToken(inf)
			e.EncodeToken(inf.End())
		}
		for _, v := range x.Pre {
			pre := xml.StartElement{}
			pre.Name.Local = "precondition"
			e.EncodeToken(pre)
			n := &extern{x: v}
			e.Encode(n)
			e.EncodeToken(pre.End())
		}
		for _, v := range x.Post {
			post := xml.StartElement{}
			post.Name.Local = "postcondition"
			e.EncodeToken(post)
			n := &extern{x: v}
			e.Encode(n)
			e.EncodeToken(post.End())
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
		for _, v := range x.Imports() {
			imp := xml.StartElement{}
			imp.Name.Local = "import"
			u.attr(&imp, "name", v)
			e.EncodeToken(imp)
			e.EncodeToken(imp.End())
		}
		if len(x.Infix()) > 0 {
			inf := xml.StartElement{}
			inf.Name.Local = "infix"
			u.attr(&inf, "num", len(x.Infix()))
			for i, v := range x.Infix() {
				u.attr(&inf, "arg"+strconv.Itoa(i), v.Name)
			}
			e.EncodeToken(inf)
			e.EncodeToken(inf.End())
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
	case *ir.AssignRule:
		start.Name.Local = "becomes"
		e.EncodeToken(start)
		n := &extern{x: x.Expr}
		e.Encode(n)
		e.EncodeToken(start.End())
	case *ir.Const:
		start.Name.Local = "constant"
		u.attr(&start, "name", x.Name)
		e.EncodeToken(start)
		n := &extern{x: x.Expr}
		e.Encode(n)
		e.EncodeToken(start.End())
	case *ir.ConstExpr:
		start.Name.Local = "constant-expression"
		u.attr(&start, "type", x.Type)
		e.EncodeToken(start)
		e.EncodeToken(u.data(x.Type, x.Value))
		e.EncodeToken(start.End())
	case *ir.AtomExpr:
		start.Name.Local = "atom-expression"
		u.attr(&start, "value", x.Value)
		e.EncodeToken(start)
		e.EncodeToken(start.End())
	case *ir.SelectExpr:
		start.Name.Local = "selector-expression"
		if x.Var != nil {
			u.attr(&start, "unit", x.Var.Unit.Name)
			u.attr(&start, "variable", x.Var.Name)
			if x.Foreign != nil {
				u.attr(&start, "foreign", x.Foreign.Name)
			}
		} else if x.Const != nil {
			u.attr(&start, "unit", x.Const.Unit.Name)
			u.attr(&start, "constant", x.Const.Name)
		} else {
			halt.As(100)
		}
		u.attr(&start, "inner", x.Inner.String())
		e.EncodeToken(start)
		for _, v := range x.ExprList {
			n := &extern{x: v}
			e.Encode(n)
		}
		e.EncodeToken(start.End())
	case *ir.Monadic:
		start.Name.Local = "monadic-expression"
		u.attr(&start, "op", x.Op.String())
		e.EncodeToken(start)
		{
			n := &extern{x: x.Expr}
			e.Encode(n)
		}
		e.EncodeToken(start.End())
	case *ir.TypeTest:
		start.Name.Local = "type-test-expression"
		assert.For(x.Typ.Basic, 20)
		u.attr(&start, "type", x.Typ.Builtin.Code.String())
		e.EncodeToken(start)
		{
			n := &extern{x: x.Operand}
			e.Encode(n)
		}
		e.EncodeToken(start.End())
	case *ir.Dyadic:
		start.Name.Local = "dyadic-expression"
		u.attr(&start, "op", x.Op.String())
		e.EncodeToken(start)
		{
			n := &extern{x: x.Left}
			e.Encode(n)
		}
		{
			n := &extern{x: x.Right}
			e.Encode(n)
		}
		e.EncodeToken(start.End())
	case *ir.Ternary:
		start.Name.Local = "ternary-expression"
		e.EncodeToken(start)
		{
			n := &extern{x: x.If}
			e.Encode(n)
		}
		{
			n := &extern{x: x.Then}
			e.Encode(n)
		}
		{
			n := &extern{x: x.Else}
			e.Encode(n)
		}
		e.EncodeToken(start.End())
	case *ir.InfixExpr:
		start.Name.Local = "infix-expression"
		u.attr(&start, "unit", x.Unit.Name())
		e.EncodeToken(start)
		for _, v := range x.Args {
			n := &extern{x: v}
			e.Encode(n)
		}
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
	stop    bool
}

type futureForeignType struct {
	name     string
	fakeUnit *ir.Unit
	imps     []string
}

func (f *futureForeignType) Name() string { return f.name }

func (f *futureForeignType) Variables() map[string]*ir.Variable { return f.fakeUnit.Variables }

func (f *futureForeignType) Imports() []string { return f.imps }

func (f *futureForeignType) Infix() []*ir.Variable { return f.fakeUnit.Infix }

type pre struct {
	expr ir.Expression
}

type post struct {
	expr ir.Expression
}

func (p *pre) Print() string  { return "pre" }
func (p *post) Print() string { return "post" }

func (p *pre) Process() ir.Expression  { return p.expr }
func (p *post) Process() ir.Expression { return p.expr }

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
	case types.INTEGER, types.REAL:
		ret = string(cd)
	case types.BOOLEAN:
		ret = string(cd) == "true"
	case types.TRILEAN:
		if s := string(cd); s == "null" {
			ret = tri.NIL
		} else if s == "true" {
			ret = tri.TRUE
		} else {
			ret = tri.FALSE
		}
	case types.CHAR:
		c, _ := strconv.ParseUint(string(cd), 16, 64)
		ret = rune(c)
	case types.STRING:
		data, err := base64.StdEncoding.DecodeString(string(cd))
		assert.For(err == nil, 30)
		ret = strings.TrimPrefix(string(data), prefix)
	case types.ANY:
		assert.For(string(cd) == "null", 20)
		ret = nil
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
			case *ir.Const:
				u.Const[x.Name] = x
			case []string:
				for _, s := range x {
					u.Infix = append(u.Infix, u.Variables[s])
				}
			case *pre:
				u.Pre = append(u.Pre, x)
			case *post:
				u.Post = append(u.Post, x)
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
			case string:
				f.imps = append(f.imps, x)
			case []string:
				for _, s := range x {
					f.fakeUnit.Infix = append(f.fakeUnit.Infix, f.fakeUnit.Variables[s])
				}
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
	case "import":
		name := i.attr(&start, "name").(string)
		i.consume(name)
	case "precondition":
		p := &pre{}
		i.x = p
		i.consume(p)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				p.expr = x
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "postcondition":
		p := &post{}
		i.x = p
		i.consume(p)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				p.expr = x
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "infix":
		if num, err := strconv.Atoi(i.attr(&start, "num").(string)); err == nil {
			var ret []string
			for j := 0; j < num; j++ {
				ret = append(ret, i.attr(&start, "arg"+strconv.Itoa(j)).(string))
			}
			i.consume(ret)
		} else {
			halt.As(101, start)
		}
	case "becomes":
		r := &ir.AssignRule{}
		i.x = r
		i.consume(r)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				r.Expr = x
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "constant":
		c := &ir.Const{}
		c.Name = i.attr(&start, "name").(string)
		i.x = c
		i.consume(c)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				c.Expr = x
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "atom-expression":
		a := &ir.AtomExpr{}
		a.Value = i.attr(&start, "value").(string)
		i.x = a
		i.consume(a)
	case "constant-expression":
		c := &ir.ConstExpr{}
		c.Type = types.TypMap[i.attr(&start, "type").(string)]
		sd, _ := d.Token()
		if td, ok := sd.(xml.CharData); ok {
			c.Value = i.data(c.Type, td)
		} else {
			halt.As(100)
		}
		i.consume(c)
		i.x = c
	case "selector-expression":
		c := &ir.SelectExpr{}
		if un := i.attr(&start, "unit").(string); un == i.root.Name {
			if vn := i.attr(&start, "variable"); vn != nil {
				c.Var = &ir.Variable{Name: vn.(string)}
				if foreign, ok := i.attr(&start, "foreign").(string); ok {
					c.Foreign = &ir.Variable{Name: foreign}
				}
			} else if cn := i.attr(&start, "constant"); cn != nil {
				c.Const = &ir.Const{Name: cn.(string)}
			} else {
				halt.As(100)
			}
			c.Inner = mods.ModMap[i.attr(&start, "inner").(string)]
			consumer = func(_x interface{}) {
				switch x := _x.(type) {
				case ir.Expression:
					c.ExprList = append(c.ExprList, x)
				default:
					halt.As(100, reflect.TypeOf(x))
				}
			}
		} else {
			halt.As(100, un)
		}
		i.x = c
		i.consume(c)
	case "monadic-expression":
		m := &ir.Monadic{}
		op := i.attr(&start, "op").(string)
		m.Op = ops.OpMap[op]
		i.x = m
		i.consume(m)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				m.Expr = x
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "type-test-expression":
		t := &ir.TypeTest{}
		typ := i.attr(&start, "type").(string)
		t.Typ.Basic = true
		t.Typ.Builtin = &ir.BuiltinType{Code: types.TypMap[typ]}
		i.x = t
		i.consume(t)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				t.Operand = x
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "dyadic-expression":
		c := &ir.Dyadic{}
		op := i.attr(&start, "op").(string)
		c.Op = ops.OpMap[op]
		i.x = c
		i.consume(c)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				if c.Left == nil {
					c.Left = x
				} else {
					c.Right = x
				}
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "ternary-expression":
		t := &ir.Ternary{}
		i.x = t
		i.consume(t)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				if t.If == nil {
					t.If = x
				} else if t.Then == nil {
					t.Then = x
				} else if t.Else == nil {
					t.Else = x
				} else {
					halt.As(100, "too much")
				}
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
	case "infix-expression":
		inf := &ir.InfixExpr{}
		inf.Unit = &futureForeignType{name: i.attr(&start, "unit").(string)}
		i.x = inf
		i.consume(inf)
		consumer = func(_x interface{}) {
			switch x := _x.(type) {
			case ir.Expression:
				inf.Args = append(inf.Args, x)
			default:
				halt.As(100, reflect.TypeOf(x))
			}
		}
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

func Init() target.Target {
	return &impl{}
}
