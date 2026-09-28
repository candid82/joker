package core

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

type (
	Traceable interface {
		Name() string
		Pos() Position
	}
	EvalError struct {
		msg  string
		pos  Position
		rt   *Runtime
		hash uint32
	}
	Frame struct {
		traceable Traceable
	}
	Callstack struct {
		frames []Frame
	}
	Runtime struct {
		callstack   *Callstack
		currentExpr Expr
		vm          *vmContext // execution handle; snapshots never retain VM storage
		GIL         sync.Mutex
	}
)

var RT *Runtime = &Runtime{
	callstack: &Callstack{frames: make([]Frame, 0, 50)},
}

// SuspendedExecution retains the ambient context of a native call while it
// releases the GIL. The context remains live until that call returns; it must
// be restored before the native code invokes Joker again after reacquiring it.
type SuspendedExecution struct {
	rt        *Runtime
	context   *vmContext
	expr      Expr
	callstack *Callstack
}

func (rt *Runtime) Suspend() SuspendedExecution {
	s := SuspendedExecution{rt: rt, context: rt.vm, expr: rt.currentExpr, callstack: rt.callstack}
	rt.GIL.Unlock()
	return s
}

func (s SuspendedExecution) Resume() {
	s.rt.GIL.Lock()
	if ctx := s.context; ctx != nil && (ctx.vm == nil || ctx.vm.context != ctx) {
		s.rt.vm, s.rt.currentExpr = nil, nil
		s.rt.callstack = &Callstack{}
		panic(s.rt.NewError("Cannot resume an expired execution"))
	}
	s.rt.vm, s.rt.currentExpr, s.rt.callstack = s.context, s.expr, s.callstack
}

// LockIndependent starts a host callback without borrowing the paused
// execution that last held the GIL. Its VM is established by CallIndependent.
func (rt *Runtime) LockIndependent() {
	rt.GIL.Lock()
	rt.vm, rt.currentExpr = nil, nil
	rt.callstack = &Callstack{}
}

// NativeSite is an immutable source location that a host callback may retain
// without retaining its creator's VM. The caller must hold the GIL.
func (rt *Runtime) NativeSite() Expr { return rt.currentExpr }

// SetNativeSite attributes errors from independent native work to its origin.
// The caller must hold the GIL and must not attach the origin's VM context.
func (rt *Runtime) SetNativeSite(site Expr) { rt.currentExpr = site }

func (rt *Runtime) clone() *Runtime {
	res := &Runtime{callstack: rt.callstack.clone(), currentExpr: rt.currentExpr}
	if rt.vm != nil {
		rt.vm.appendTrace(res.callstack)
		if vm := rt.vm.vm; vm != nil && vm.frameCount > 0 {
			frame := &vm.frames[vm.frameCount-1]
			pos := frame.arityProto.Chunk.positionAt(frame.lastOp)
			if pos.startLine > 0 {
				res.currentExpr = &CallExpr{Position: pos}
			}
		}
	}
	return res
}

func (rt *Runtime) NewError(msg string) *EvalError {
	res := &EvalError{
		msg: msg,
		rt:  rt.clone(),
	}
	if res.rt.currentExpr != nil {
		res.pos = res.rt.currentExpr.Pos()
	}
	return res
}

func (rt *Runtime) NewArgTypeError(index int, obj Object, expectedType string) *EvalError {
	name := rt.currentExpr.(Traceable).Name()
	return rt.NewError(fmt.Sprintf("Arg[%d] of %s must have type %s, got %s", index, name, expectedType, obj.GetType().ToString(false)))
}

func (rt *Runtime) NewErrorWithPos(msg string, pos Position) *EvalError {
	return &EvalError{
		msg: msg,
		pos: pos,
		rt:  rt.clone(),
	}
}

func (rt *Runtime) stacktrace() string {
	b := getBuffer()
	defer putBuffer(b)
	pos := Position{}
	if rt.currentExpr != nil {
		pos = rt.currentExpr.Pos()
	}
	name := "global"
	for _, f := range rt.callstack.frames {
		framePos := f.traceable.Pos()
		b.WriteString(fmt.Sprintf("  %s %s:%d:%d\n", name, framePos.Filename(), framePos.startLine, framePos.startColumn))
		name = f.traceable.Name()
		if strings.HasPrefix(name, "#'") {
			name = name[2:]
		}
	}
	b.WriteString(fmt.Sprintf("  %s %s:%d:%d", name, pos.Filename(), pos.startLine, pos.startColumn))
	return b.String()
}

func (s *Callstack) pushFrame(frame Frame) {
	s.frames = append(s.frames, frame)
}

func (s *Callstack) clone() *Callstack {
	res := &Callstack{frames: make([]Frame, len(s.frames))}
	copy(res.frames, s.frames)
	return res
}

func (s *Callstack) String() string {
	b := getBuffer()
	defer putBuffer(b)
	for _, f := range s.frames {
		pos := f.traceable.Pos()
		b.WriteString(fmt.Sprintf("%s %s:%d:%d\n", f.traceable.Name(), pos.Filename(), pos.startLine, pos.startColumn))
	}
	if b.Len() > 0 {
		b.Truncate(b.Len() - 1)
	}
	return b.String()
}

func MakeEvalError(msg string, pos Position, rt *Runtime) *EvalError {
	res := &EvalError{msg, pos, rt, 0}
	res.hash = HashPtr(uintptr(unsafe.Pointer(res)))
	return res
}

func (err *EvalError) ToString(escape bool) string {
	return err.Error()
}

func (err *EvalError) Equals(other interface{}) bool {
	return err == other
}

func (err *EvalError) GetInfo() *ObjectInfo {
	return nil
}

func (err *EvalError) GetType() *Type {
	return TYPE.EvalError
}

func (err *EvalError) Hash() uint32 {
	return err.hash
}

func (err *EvalError) WithInfo(info *ObjectInfo) Object {
	return err
}

func (err *EvalError) Message() Object {
	return MakeString(err.msg)
}

func (err *EvalError) Error() string {
	pos := err.pos
	if len(err.rt.callstack.frames) > 0 && !LINTER_MODE {
		return fmt.Sprintf("%s:%d:%d: Eval error: %s\nStacktrace:\n%s", pos.Filename(), pos.startLine, pos.startColumn, err.msg, err.rt.stacktrace())
	} else {
		if len(err.rt.callstack.frames) > 0 {
			pos = err.rt.callstack.frames[0].traceable.Pos()
		}
		return fmt.Sprintf("%s:%d:%d: Eval error: %s", pos.Filename(), pos.startLine, pos.startColumn, err.msg)
	}
}

func varCallableString(v *Var) string {
	if v.ns == GLOBAL_ENV.CoreNamespace {
		return "core/" + v.name.ToString(false)
	}
	return v.ns.Name.ToString(false) + "/" + v.name.ToString(false)
}

func (expr *CallExpr) Name() string {
	if expr.callName != "" {
		return expr.callName
	}
	switch c := expr.callable.(type) {
	case *VarRefExpr:
		return varCallableString(c.vr)
	case *BindingExpr:
		return c.binding.name.ToString(false)
	case *LiteralExpr:
		return c.obj.ToString(false)
	default:
		return "fn"
	}
}

func (expr *MacroCallExpr) Name() string {
	return expr.name
}

func PanicOnErr(err error) {
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
}
