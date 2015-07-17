package p

type variable struct {
	modifier int
}

type target struct {
	name string
}

func (t *target) do(mod string) {
	t.name = mod
}

func (t *target) obj(name string, opt variable) {

}
