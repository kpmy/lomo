package ir

type Rule interface {
	Show() string
}

type ConditionalBlock struct {
	Condition, Expr Expression
}

type ConditionalRule struct {
	Blocks  []ConditionalBlock
	Default Expression
}

func (r *ConditionalRule) Show() string { return "cond" }
