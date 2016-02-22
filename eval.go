package main

import (
	"bytes"
	"fmt"
)

type (
	EvalError struct {
		msg string
		pos Position
		rt  *Runtime
	}
	Frame struct {
		callExpr *CallExpr
	}
	CallStack struct {
		frames []Frame
	}
	Runtime struct {
		callstack   *CallStack
		currentExpr Expr
	}
)

var RT *Runtime = &Runtime{
	callstack: &CallStack{frames: make([]Frame, 0, 50)},
}

func (rt *Runtime) clone() *Runtime {
	return &Runtime{
		callstack:   rt.callstack.clone(),
		currentExpr: rt.currentExpr,
	}
}

func (rt *Runtime) newError(msg string) *EvalError {
	return &EvalError{
		msg: msg,
		pos: rt.currentExpr.Pos(),
		rt:  rt.clone(),
	}
}

func (rt *Runtime) stacktrace() string {
	var b bytes.Buffer
	pos := rt.currentExpr.Pos()
	// line, column := pos.line, pos.column
	name := "global"
	for _, f := range rt.callstack.frames {
		b.WriteString(fmt.Sprintf("%s %d:%d\n", name, f.callExpr.line, f.callExpr.column))
		name = f.callExpr.name
		// line, column = f.callExpr.line, f.callExpr.column
	}
	b.WriteString(fmt.Sprintf("%s %d:%d", name, pos.line, pos.column))
	return b.String()
}

func (rt *Runtime) pushFrame() {
	rt.callstack.pushFrame(Frame{callExpr: rt.currentExpr.(*CallExpr)})
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

func (s *CallStack) pushFrame(frame Frame) {
	s.frames = append(s.frames, frame)
}

func (s *CallStack) popFrame() {
	s.frames = s.frames[:len(s.frames)-1]
}

func (s *CallStack) clone() *CallStack {
	res := &CallStack{frames: make([]Frame, len(s.frames))}
	copy(res.frames, s.frames)
	return res
}

func (s *CallStack) String() string {
	var b bytes.Buffer
	for _, f := range s.frames {
		b.WriteString(fmt.Sprintf("%s %d:%d\n", f.callExpr.name, f.callExpr.line, f.callExpr.column))
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

func (expr *VarRefExpr) Eval(env *LocalEnv) Object {
	// TODO: Clojure returns clojure.lang.Var$Unbound object in this case.
	if expr.vr.value == nil {
		panic(&EvalError{
			msg: "Unbound var: " + expr.vr.ToString(false),
			pos: expr.Position,
		})
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
			panic(&EvalError{
				msg: "Duplicate key: " + key.ToString(false),
				pos: expr.Position,
			})
		}
	}
	return res
}

func (expr *SetExpr) Eval(env *LocalEnv) Object {
	res := EmptySet()
	for _, elemExpr := range expr.elements {
		el := eval(elemExpr, env)
		if !res.Add(el) {
			panic(&EvalError{
				msg: "Duplicate set element: " + el.ToString(false),
				pos: expr.Position,
			})
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
		panic(&EvalError{
			msg: "Enable to resolve var " + expr.symbol.ToString(false) + " in this context",
			pos: expr.Position,
		})
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
		panic(&EvalError{
			msg: callable.ToString(false) + " is not callable",
			pos: expr.callable.Pos(),
		})
	}
}

func evalBody(body []Expr, env *LocalEnv) Object {
	var res Object = NIL
	for _, expr := range body {
		res = eval(expr, env)
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
		return bool(obj)
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

func TryEval(expr Expr) (obj Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	return eval(expr, nil), nil
}
