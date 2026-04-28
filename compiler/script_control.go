package compiler

import (
	"errors"
	"go/token"

	"github.com/samber/mo"
)

type execSignalKind int

const (
	execSignalNone execSignalKind = iota
	execSignalReturn
	execSignalBreak
	execSignalContinue
)

type execSignal struct {
	kind  execSignalKind
	pos   token.Pos
	end   token.Pos
	value any
}

func noExecSignal() execSignal {
	return execSignal{}
}

func returnExecSignal(pos, end token.Pos, value any) execSignal {
	return execSignal{kind: execSignalReturn, pos: pos, end: end, value: value}
}

func breakExecSignal(pos, end token.Pos) execSignal {
	return execSignal{kind: execSignalBreak, pos: pos, end: end}
}

func continueExecSignal(pos, end token.Pos) execSignal {
	return execSignal{kind: execSignalContinue, pos: pos, end: end}
}

func (s execSignal) IsNone() bool {
	return s.kind == execSignalNone
}

func (s execSignal) IsReturn() bool {
	return s.kind == execSignalReturn
}

func (s execSignal) IsBreak() bool {
	return s.kind == execSignalBreak
}

func (s execSignal) IsContinue() bool {
	return s.kind == execSignalContinue
}

func (s execSignal) Result() mo.Option[any] {
	if s.IsReturn() {
		return mo.Some[any](s.value)
	}
	return mo.None[any]()
}

func unexpectedLoopControlError(signal execSignal) error {
	switch {
	case signal.IsBreak():
		return errors.New("break is only allowed inside loops")
	case signal.IsContinue():
		return errors.New("continue is only allowed inside loops")
	default:
		return nil
	}
}
