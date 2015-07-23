package ir

type Rule interface {
	Show() string
}

type AssignRule struct {
	Expr Expression
}

func (r *AssignRule) Show() string { return "assign" }
