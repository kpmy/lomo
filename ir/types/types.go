package types

import (
	"strconv"
)

type Type int

const (
	UNDEF Type = iota
	INTEGER
	BOOLEAN
	TRILEAN
	ATOM
	REAL
	CHAR
	STRING
	ANY
	COMPLEX

	none
)

func (t Type) String() string {
	switch t {
	case UNDEF:
		return "UNDEF"
	case INTEGER:
		return "INTEGER"
	case BOOLEAN:
		return "BOOLEAN"
	case TRILEAN:
		return "TRILEAN"
	case ATOM:
		return "ATOM"
	case REAL:
		return "REAL"
	case CHAR:
		return "CHAR"
	case STRING:
		return "STRING"
	case ANY:
		return "ANY"
	case COMPLEX:
		return "COMPLEX"
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
