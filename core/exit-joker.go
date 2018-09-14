package core

var ExitJoker func (rc int)

func SetExitJoker(fn func(rc int)) {
	ExitJoker = fn
}
