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
			mappings: make(map[Symbol]Object),
		},
	}
	res.namespaces[currentNs] = res.currentNamespace
	return res
}

func (err EvalError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: %s", err.pos.line, err.pos.column, err.msg)
}

func (env *Env) Resolve(s Symbol) (Object, bool) {
	var ns *Namespace
	if s.ns == nil {
		ns = env.currentNamespace
	} else {
		ns = env.namespaces[Symbol{name: s.ns}]
	}
	if ns == nil {
		return nil, false
	}
	obj, ok := ns.mappings[Symbol{name: s.name}]
	return obj, ok
}

func (expr *RefExpr) Eval(env *Env) Object {
	v, ok := env.Resolve(expr.symbol)
	if !ok {
		panic(&EvalError{
			msg: "Unbound symbol: " + expr.symbol.ToString(false),
			pos: expr.Position,
		})
	}
	return v
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
	v := expr.value.Eval(env)
	env.currentNamespace.mappings[Symbol{name: expr.name.name}] = v
	return v
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
