package core

import (
	"bytes"
	"fmt"
	"strings"
	"unsafe"
)

type (
	Traceable interface {
		Name() string
		Pos() Position
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

func (rt *Runtime) NewError(msg string) *EvalError {
	res := &EvalError{
		msg: msg,
		rt:  rt.clone(),
	}
	if rt.currentExpr != nil {
		res.pos = rt.currentExpr.Pos()
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
	var b bytes.Buffer
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

func (rt *Runtime) pushFrame() {
	// TODO: this is all wrong. We cannot rely on
	// currentExpr for stacktraces. Instead, each Callable
	// should know it's name / position.
	var tr Traceable
	if rt.currentExpr != nil {
		tr = rt.currentExpr.(Traceable)
	} else {
		tr = &CallExpr{}
	}
	rt.callstack.pushFrame(Frame{traceable: tr})
}

func (rt *Runtime) popFrame() {
	rt.callstack.popFrame()
}

func Eval(expr Expr, env *LocalEnv) Object {
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
		b.WriteString(fmt.Sprintf("%s %s:%d:%d\n", f.traceable.Name(), pos.Filename(), pos.startLine, pos.startColumn))
	}
	if b.Len() > 0 {
		b.Truncate(b.Len() - 1)
	}
	return b.String()
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
	return hashPtr(uintptr(unsafe.Pointer(err)))
}

func (err *EvalError) WithInfo(info *ObjectInfo) Object {
	return err
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

func (expr *VarRefExpr) Eval(env *LocalEnv) Object {
	// TODO: Clojure returns clojure.lang.Var$Unbound object in this case.
	return expr.vr.Resolve()
}

func (expr *SetMacroExpr) Eval(env *LocalEnv) Object {
	expr.vr.isMacro = true
	expr.vr.isUsed = false
	if fn, ok := expr.vr.Value.(*Fn); ok {
		fn.isMacro = true
	}
	setMacroMeta(expr.vr)
	return expr.vr
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
		res = res.Conjoin(Eval(e, env))
	}
	return res
}

func (expr *MapExpr) Eval(env *LocalEnv) Object {
	if len(expr.keys) > HASHMAP_THRESHOLD/2 {
		res := EmptyHashMap
		for i := range expr.keys {
			key := Eval(expr.keys[i], env)
			if res.containsKey(key) {
				panic(RT.NewError("Duplicate key: " + key.ToString(false)))
			}
			res = res.Assoc(key, Eval(expr.values[i], env)).(*HashMap)
		}
		return res
	}
	res := EmptyArrayMap()
	for i := range expr.keys {
		key := Eval(expr.keys[i], env)
		if !res.Add(key, Eval(expr.values[i], env)) {
			panic(RT.NewError("Duplicate key: " + key.ToString(false)))
		}
	}
	return res
}

func (expr *SetExpr) Eval(env *LocalEnv) Object {
	res := EmptySet()
	for _, elemExpr := range expr.elements {
		el := Eval(elemExpr, env)
		if !res.Add(el) {
			panic(RT.NewError("Duplicate set element: " + el.ToString(false)))
		}
	}
	return res
}

func (expr *DefExpr) Eval(env *LocalEnv) Object {
	if expr.value != nil {
		expr.vr.Value = Eval(expr.value, env)
	}
	meta := EmptyArrayMap()
	meta.Add(KEYWORDS.line, Int{I: expr.startLine})
	meta.Add(KEYWORDS.column, Int{I: expr.startColumn})
	meta.Add(KEYWORDS.file, String{S: *expr.filename})
	meta.Add(KEYWORDS.ns, expr.vr.ns)
	meta.Add(KEYWORDS.name, expr.vr.name)
	expr.vr.meta = meta
	if expr.meta != nil {
		expr.vr.meta = expr.vr.meta.Merge(Eval(expr.meta, env).(Map))
	}
	// isMacro can be set by set-macro__ during parse stage
	if expr.vr.isMacro {
		expr.vr.meta = expr.vr.meta.Assoc(KEYWORDS.macro, Boolean{B: true}).(Map)
	}
	return expr.vr
}

func (expr *MetaExpr) Eval(env *LocalEnv) Object {
	meta := Eval(expr.meta, env)
	res := Eval(expr.expr, env)
	return res.(Meta).WithMeta(meta.(Map))
}

func evalSeq(exprs []Expr, env *LocalEnv) []Object {
	res := make([]Object, len(exprs))
	for i, expr := range exprs {
		res[i] = Eval(expr, env)
	}
	return res
}

func (expr *CallExpr) Eval(env *LocalEnv) Object {
	callable := Eval(expr.callable, env)
	switch callable := callable.(type) {
	case Callable:
		args := evalSeq(expr.args, env)
		return callable.Call(args)
	default:
		panic(RT.NewErrorWithPos(callable.ToString(false)+" is not a Fn", expr.callable.Pos()))
	}
}

func varCallableString(v *Var) string {
	if v.ns == GLOBAL_ENV.CoreNamespace {
		return "core/" + v.name.ToString(false)
	}
	return v.ns.Name.ToString(false) + "/" + v.name.ToString(false)
}

func (expr *CallExpr) Name() string {
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

func (expr *ThrowExpr) Eval(env *LocalEnv) Object {
	e := Eval(expr.e, env)
	switch e.(type) {
	case Error:
		panic(e)
	default:
		panic(RT.NewError("Cannot throw " + e.ToString(false)))
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
					if IsInstance(catchExpr.excType, r) {
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

func (expr *CatchExpr) Eval(env *LocalEnv) (obj Object) {
	panic(RT.NewError("This should never happen!"))
}

func evalBody(body []Expr, env *LocalEnv) Object {
	var res Object = NIL
	for _, expr := range body {
		res = Eval(expr, env)
	}
	return res
}

func evalLoop(body []Expr, env *LocalEnv) Object {
	var res Object = NIL
loop:
	for _, expr := range body {
		res = Eval(expr, env)
	}
	switch res := res.(type) {
	default:
		return res
	case RecurBindings:
		env = env.replaceFrame(res)
		goto loop
	}
}

func (doExpr *DoExpr) Eval(env *LocalEnv) Object {
	return evalBody(doExpr.body, env)
}

func toBool(obj Object) bool {
	switch obj := obj.(type) {
	case Nil:
		return false
	case Boolean:
		return obj.B
	default:
		return true
	}
}

func (expr *IfExpr) Eval(env *LocalEnv) Object {
	if toBool(Eval(expr.cond, env)) {
		return Eval(expr.positive, env)
	}
	return Eval(expr.negative, env)
}

func (expr *FnExpr) Eval(env *LocalEnv) Object {
	res := &Fn{fnExpr: expr}
	if expr.self.name != nil {
		env = env.addFrame([]Object{res})
	}
	res.env = env
	return res
}

func (expr *FnArityExpr) Eval(env *LocalEnv) Object {
	panic(RT.NewError("This should never happen!"))
}

func (expr *LetExpr) Eval(env *LocalEnv) Object {
	env = env.addEmptyFrame(len(expr.names))
	for _, bindingExpr := range expr.values {
		env.addBinding(Eval(bindingExpr, env))
	}
	return evalBody(expr.body, env)
}

func (expr *LoopExpr) Eval(env *LocalEnv) Object {
	env = env.addEmptyFrame(len(expr.names))
	for _, bindingExpr := range expr.values {
		env.addBinding(Eval(bindingExpr, env))
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
			switch r.(type) {
			case *EvalError:
				err = r.(error)
			case *ExInfo:
				err = r.(error)
			default:
				panic(r)
			}
		}
	}()
	return Eval(expr, nil), nil
}

func PanicOnErr(err error) {
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
}
