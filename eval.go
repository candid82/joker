package main

import (
	"fmt"
)

type (
	EvalError struct {
		msg string
		pos Position
	}
)

var GLOBAL_ENV = NewEnv(MakeSymbol("user"))

func NewEnv(currentNs Symbol) *Env {
	res := &Env{
		namespaces: make(map[Symbol]*Namespace),
		currentNamespace: &Namespace{
			name:     currentNs,
			mappings: make(map[Symbol]*Var),
		},
	}
	res.namespaces[currentNs] = res.currentNamespace
	return res
}

func (err EvalError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: Eval error: %s", err.pos.line, err.pos.column, err.msg)
}

func (env *Env) Resolve(s Symbol) (*Var, bool) {
	var ns *Namespace
	if s.ns == nil {
		ns = env.currentNamespace
	} else {
		ns = env.namespaces[Symbol{name: s.ns}]
	}
	if ns == nil {
		return nil, false
	}
	v, ok := ns.mappings[Symbol{name: s.name}]
	return v, ok
}

func (expr *RefExpr) Eval(env *Env) Object {
	v, ok := env.Resolve(expr.symbol)
	if !ok {
		panic(&EvalError{
			msg: "Unbound symbol: " + expr.symbol.ToString(false),
			pos: expr.Position,
		})
	}
	// TODO: Clojure returns clojure.lang.Var$Unbound object in this case.
	if v.value == nil {
		panic(&EvalError{
			msg: "Unbound var: " + v.ToString(false),
			pos: expr.Position,
		})
	}
	return v.value
}

func (expr *LiteralExpr) Eval(env *Env) Object {
	return expr.obj
}

func (expr *VectorExpr) Eval(env *Env) Object {
	res := EmptyVector
	for _, e := range expr.v {
		res = res.conj(e.Eval(env))
	}
	return res
}

func (expr *MapExpr) Eval(env *Env) Object {
	res := EmptyArrayMap()
	for i := range expr.keys {
		key := expr.keys[i].Eval(env)
		if !res.Add(key, expr.values[i].Eval(env)) {
			panic(&EvalError{
				msg: "Duplicate key: " + key.ToString(false),
				pos: expr.Position,
			})
		}
	}
	return res
}

func (expr *SetExpr) Eval(env *Env) Object {
	res := EmptySet()
	for _, elemExpr := range expr.elements {
		el := elemExpr.Eval(env)
		if !res.Add(el) {
			panic(&EvalError{
				msg: "Duplicate set element: " + el.ToString(false),
				pos: expr.Position,
			})
		}
	}
	return res
}

func (expr *DefExpr) Eval(env *Env) Object {
	if expr.name.ns != nil && (Symbol{name: expr.name.ns} != env.currentNamespace.name) {
		panic(&EvalError{
			msg: "Can't create defs outside of current ns",
			pos: expr.Position,
		})
	}
	v := env.currentNamespace.intern(Symbol{name: expr.name.name})
	if expr.value != nil {
		v.value = expr.value.Eval(env)
	}
	if expr.meta != nil {
		v.meta = expr.meta.Eval(env).(*ArrayMap)
	}
	return v
}

func (expr *VarExpr) Eval(env *Env) Object {
	res, ok := env.Resolve(expr.symbol)
	if !ok {
		panic(&EvalError{
			msg: "Enable to resolve var " + expr.symbol.ToString(false) + " in this context",
			pos: expr.Position,
		})
	}
	return res
}

func (expr *MetaExpr) Eval(env *Env) Object {
	meta := expr.meta.Eval(env)
	res := expr.expr.Eval(env)
	return res.(Meta).WithMeta(meta.(*ArrayMap))
}

func evalSeq(exprs []Expr, env *Env) []Object {
	res := make([]Object, len(exprs))
	for i, expr := range exprs {
		res[i] = expr.Eval(env)
	}
	return res
}

func (expr *CallExpr) Eval(env *Env) Object {
	callable := expr.callable.Eval(env)
	switch callable := callable.(type) {
	case Callable:
		return callable.Call(evalSeq(expr.args, env))
	default:
		panic(&EvalError{
			msg: callable.ToString(false) + " is not callable",
			pos: expr.callable.Pos(),
		})
	}
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

func (expr *IfExpr) Eval(env *Env) Object {
	if toBool(expr.cond.Eval(env)) {
		return expr.positive.Eval(env)
	}
	return expr.negative.Eval(env)
}

func TryEval(expr Expr) (obj Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	return expr.Eval(GLOBAL_ENV), nil
}
