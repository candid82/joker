package main

import (
	"bytes"
	"fmt"
)

type (
	Traceable interface {
		Pos() Position
		Name() string
	}
	EvalError struct {
		msg string
		pos Position
		rt  *Runtime
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
	}
)

var RT *Runtime = &Runtime{
	callstack: &Callstack{frames: make([]Frame, 0, 50)},
}

func (rt *Runtime) clone() *Runtime {
	return &Runtime{
		callstack:   rt.callstack.clone(),
		currentExpr: rt.currentExpr,
	}
}

func (rt *Runtime) newError(msg string) *EvalError {
	res := &EvalError{
		msg: msg,
		rt:  rt.clone(),
	}
	if rt.currentExpr != nil {
		res.pos = rt.currentExpr.Pos()
	}
	return res
}

func (rt *Runtime) newArgTypeError(index int, typeName string) *EvalError {
	name := rt.currentExpr.(*CallExpr).name
	return rt.newError(fmt.Sprintf("Arg[%d] of %s must be of type %s", index, name, typeName))
}

func (rt *Runtime) newErrorWithPos(msg string, pos Position) *EvalError {
	return &EvalError{
		msg: msg,
		pos: pos,
		rt:  rt.clone(),
	}
}

func (rt *Runtime) stacktrace() string {
	var b bytes.Buffer
	pos := Position{}
	if rt.currentExpr != nil {
		pos = rt.currentExpr.Pos()
	}
	name := "global"
	for _, f := range rt.callstack.frames {
		pos := f.traceable.Pos()
		b.WriteString(fmt.Sprintf("%s %d:%d\n", name, pos.line, pos.column))
		name = f.traceable.Name()
	}
	b.WriteString(fmt.Sprintf("%s %d:%d", name, pos.line, pos.column))
	return b.String()
}

func (rt *Runtime) pushFrame() {
	rt.callstack.pushFrame(Frame{traceable: rt.currentExpr.(Traceable)})
}

func (rt *Runtime) popFrame() {
	rt.callstack.popFrame()
}

func eval(expr Expr, env *LocalEnv) Object {
	parentExpr := RT.currentExpr
	RT.currentExpr = expr
	defer (func() { RT.currentExpr = parentExpr })()
	return expr.Eval(env)
}

func (s *Callstack) pushFrame(frame Frame) {
	s.frames = append(s.frames, frame)
}

func (s *Callstack) popFrame() {
	s.frames = s.frames[:len(s.frames)-1]
}

func (s *Callstack) clone() *Callstack {
	res := &Callstack{frames: make([]Frame, len(s.frames))}
	copy(res.frames, s.frames)
	return res
}

func (s *Callstack) String() string {
	var b bytes.Buffer
	for _, f := range s.frames {
		pos := f.traceable.Pos()
		b.WriteString(fmt.Sprintf("%s %d:%d\n", f.traceable.Pos(), pos.line, pos.column))
	}
	if b.Len() > 0 {
		b.Truncate(b.Len() - 1)
	}
	return b.String()
}

func (err EvalError) Error() string {
	if len(err.rt.callstack.frames) > 0 {
		return fmt.Sprintf("stdin:%d:%d: Eval error: %s\nStacktrace:\n%s", err.pos.line, err.pos.column, err.msg, err.rt.stacktrace())
	} else {
		return fmt.Sprintf("stdin:%d:%d: Eval error: %s", err.pos.line, err.pos.column, err.msg)
	}
}

func (err EvalError) Type() Symbol {
	return MakeSymbol("EvalError")
}

func (expr *VarRefExpr) Eval(env *LocalEnv) Object {
	// TODO: Clojure returns clojure.lang.Var$Unbound object in this case.
	if expr.vr.value == nil {
		panic(RT.newError("Unbound var: " + expr.vr.ToString(false)))
	}
	return expr.vr.value
}

func (expr *BindingExpr) Eval(env *LocalEnv) Object {
	for i := env.frame; i > expr.binding.frame; i-- {
		env = env.parent
	}
	return env.bindings[expr.binding.index]
}

func (expr *LiteralExpr) Eval(env *LocalEnv) Object {
	return expr.obj
}

func (expr *VectorExpr) Eval(env *LocalEnv) Object {
	res := EmptyVector
	for _, e := range expr.v {
		res = res.conj(eval(e, env))
	}
	return res
}

func (expr *MapExpr) Eval(env *LocalEnv) Object {
	res := EmptyArrayMap()
	for i := range expr.keys {
		key := eval(expr.keys[i], env)
		if !res.Add(key, eval(expr.values[i], env)) {
			panic(RT.newError("Duplicate key: " + key.ToString(false)))
		}
	}
	return res
}

func (expr *SetExpr) Eval(env *LocalEnv) Object {
	res := EmptySet()
	for _, elemExpr := range expr.elements {
		el := eval(elemExpr, env)
		if !res.Add(el) {
			panic(RT.newError("Duplicate set element: " + el.ToString(false)))
		}
	}
	return res
}

func (expr *DefExpr) Eval(env *LocalEnv) Object {
	if expr.value != nil {
		expr.vr.value = eval(expr.value, env)
	}
	if expr.meta != nil {
		expr.vr.meta = eval(expr.meta, env).(*ArrayMap)
	}
	return expr.vr
}

func (expr *VarExpr) Eval(env *LocalEnv) Object {
	res, ok := GLOBAL_ENV.Resolve(expr.symbol)
	if !ok {
		panic(RT.newError("Enable to resolve var " + expr.symbol.ToString(false) + " in this context"))
	}
	return res
}

func (expr *MetaExpr) Eval(env *LocalEnv) Object {
	meta := eval(expr.meta, env)
	res := eval(expr.expr, env)
	return res.(Meta).WithMeta(meta.(*ArrayMap))
}

func evalSeq(exprs []Expr, env *LocalEnv) []Object {
	res := make([]Object, len(exprs))
	for i, expr := range exprs {
		res[i] = eval(expr, env)
	}
	return res
}

func (expr *CallExpr) Eval(env *LocalEnv) Object {
	callable := eval(expr.callable, env)
	switch callable := callable.(type) {
	case Callable:
		args := evalSeq(expr.args, env)
		return callable.Call(args)
	default:
		panic(RT.newErrorWithPos(callable.ToString(false)+" is not callable", expr.callable.Pos()))
	}
}

func (expr *CallExpr) Name() string {
	return expr.name
}

func (expr *ThrowExpr) Eval(env *LocalEnv) Object {
	e := eval(expr.e, env)
	switch e.(type) {
	case Error:
		panic(e)
	default:
		panic(RT.newError("Cannot throw " + e.ToString(false)))
	}
}

func (expr *TryExpr) Eval(env *LocalEnv) (obj Object) {
	defer func() {
		defer func() {
			if expr.finallyExpr != nil {
				evalBody(expr.finallyExpr, env)
			}
		}()
		if r := recover(); r != nil {
			switch r := r.(type) {
			case Error:
				for _, catchExpr := range expr.catches {
					if catchExpr.excType.Equals(r.Type()) {
						obj = evalBody(catchExpr.body, env.addFrame([]Object{r}))
						return
					}
				}
				panic(r)
			default:
				panic(r)
			}
		}
	}()
	return evalBody(expr.body, env)
}

func evalBody(body []Expr, env *LocalEnv) Object {
	var res Object = NIL
	for _, expr := range body {
		res = eval(expr, env)
	}
	return res
}

func evalLoop(body []Expr, env *LocalEnv) Object {
	var res Object = NIL
loop:
	for _, expr := range body {
		res = eval(expr, env)
	}
	switch res := res.(type) {
	default:
		return res
	case RecurBindings:
		env = env.replaceFrame(res)
		goto loop
	}
	return res
}

func (doExpr *DoExpr) Eval(env *LocalEnv) Object {
	return evalBody(doExpr.body, env)
}

func toBool(obj Object) bool {
	switch obj := obj.(type) {
	case Nil:
		return false
	case Bool:
		return obj.b
	default:
		return true
	}
}

func (expr *IfExpr) Eval(env *LocalEnv) Object {
	if toBool(eval(expr.cond, env)) {
		return eval(expr.positive, env)
	}
	return eval(expr.negative, env)
}

func (expr *FnExpr) Eval(env *LocalEnv) Object {
	res := &Fn{fnExpr: expr}
	if expr.self.name != nil {
		env = env.addFrame([]Object{res})
	}
	res.env = env
	return res
}

func (expr *LetExpr) Eval(env *LocalEnv) Object {
	env = env.addEmptyFrame(len(expr.names))
	for _, bindingExpr := range expr.values {
		env.addBinding(eval(bindingExpr, env))
	}
	return evalBody(expr.body, env)
}

func (expr *LoopExpr) Eval(env *LocalEnv) Object {
	env = env.addEmptyFrame(len(expr.names))
	for _, bindingExpr := range expr.values {
		env.addBinding(eval(bindingExpr, env))
	}
	return evalLoop(expr.body, env)
}

func (expr *RecurExpr) Eval(env *LocalEnv) Object {
	return RecurBindings(evalSeq(expr.args, env))
}

func (expr *MacroCallExpr) Eval(env *LocalEnv) Object {
	return expr.macro.Call(expr.args)
}

func (expr *MacroCallExpr) Name() string {
	return expr.name
}

func TryEval(expr Expr) (obj Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	return eval(expr, nil), nil
}
