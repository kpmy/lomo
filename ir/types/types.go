package types

import (
	"strconv"
)

type Type int

const (
	UNDEF Type = iota
	INTEGER

	none
)

func (t Type) String() string {
	switch t {
	case UNDEF:
		return "UNDEF"
	case INTEGER:
		return "INTEGER"
	default:
		return strconv.Itoa(int(t))
	}
}

var TypMap map[string]Type

func init() {
	TypMap = make(map[string]Type)
	for i := int(UNDEF); i < int(none); i++ {
		t := Type(i)
		TypMap[t.String()] = t
	}
}
