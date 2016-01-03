package main

import (
	"fmt"
)

type (
	Expr interface {
		Eval() Object
	}
	LiteralExpr struct {
		obj Object
	}
	VectorExpr struct {
		v []Expr
	}
	MapExpr struct {
		keys   []Expr
		values []Expr
	}
	SetExpr struct {
		elements []Expr
	}
	IfExpr struct {
		cond Expr
		pos  Expr
		neg  Expr
	}
	DefExpr struct {
		name  Symbol
		value Expr
	}
	CallExpr struct {
		callable Expr
		args     []Expr
	}
	ParseError struct {
		obj ReadObject
		msg string
	}
	EvalError struct {
		msg string
	}
	Callable interface {
		Call(args []Object) Object
	}
)

var GLOBAL_ENV = map[Symbol]Object{}

func (err ParseError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: %s", err.obj.line, err.obj.column, err.msg)
}

func (err EvalError) Error() string {
	return fmt.Sprintf("%s", err.msg)
}

func (expr *LiteralExpr) Eval() Object {
	return expr.obj
}

func (expr *VectorExpr) Eval() Object {
	res := EmptyVector
	for _, e := range expr.v {
		res = res.conj(e.Eval())
	}
	return res
}

func (expr *MapExpr) Eval() Object {
	res := EmptyArrayMap()
	for i := range expr.keys {
		key := expr.keys[i].Eval()
		if !res.Add(key, expr.values[i].Eval()) {
			panic(&EvalError{msg: "Duplicate key: " + key.ToString(false)})
		}
	}
	return res
}

func (expr *SetExpr) Eval() Object {
	res := EmptySet()
	for _, elemExpr := range expr.elements {
		el := elemExpr.Eval()
		if !res.Add(el) {
			panic(&EvalError{msg: "Duplicate set element: " + el.ToString(false)})
		}
	}
	return res
}

func (expr *DefExpr) Eval() Object {
	v := expr.value.Eval()
	GLOBAL_ENV[expr.name] = v
	return v
}

func evalSeq(exprs []Expr) []Object {
	res := make([]Object, len(exprs))
	for i, expr := range exprs {
		res[i] = expr.Eval()
	}
	return res
}

func (expr *CallExpr) Eval() Object {
	callable := expr.callable.Eval()
	switch callable := callable.(type) {
	case Callable:
		return callable.Call(evalSeq(expr.args))
	default:
		panic(EvalError{msg: callable.ToString(false) + " is not callable"})
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

func (expr *IfExpr) Eval() Object {
	if toBool(expr.cond.Eval()) {
		return expr.pos.Eval()
	}
	return expr.neg.Eval()
}

func ensureReadObject(obj Object) ReadObject {
	switch obj := obj.(type) {
	case ReadObject:
		return obj
	default:
		return ReadObject{obj: obj}
	}
}

func parseSeq(seq Seq) []Expr {
	res := make([]Expr, 0)
	for !seq.IsEmpty() {
		res = append(res, parse(ensureReadObject(seq.First())))
		seq = seq.Rest()
	}
	return res
}

func parseVector(v *Vector) Expr {
	r := make([]Expr, v.count)
	for i := 0; i < v.count; i++ {
		r[i] = parse(ensureReadObject(v.at(i)))
	}
	return &VectorExpr{v: r}
}

func parseMap(m *ArrayMap) Expr {
	res := MapExpr{
		keys:   make([]Expr, m.Count()),
		values: make([]Expr, m.Count()),
	}
	for iter, i := m.iter(), 0; iter.HasNext(); i++ {
		p := iter.Next()
		res.keys[i] = parse(ensureReadObject(p.key))
		res.values[i] = parse(ensureReadObject(p.value))
	}
	return &res
}

func parseSet(s *Set) Expr {
	res := SetExpr{
		elements: make([]Expr, s.m.Count()),
	}
	for iter, i := iter(s.Seq()), 0; iter.HasNext(); i++ {
		res.elements[i] = parse(ensureReadObject(iter.Next()))
	}
	return &res
}

func checkForm(obj ReadObject, min int, max int) {
	list := obj.obj.(*List)
	if list.count < min {
		panic(&ParseError{obj: obj, msg: "Too few arguments to " + list.first.ToString(false)})
	}
}

func parseList(obj ReadObject) Expr {
	list := obj.obj.(*List)
	if list.count == 0 {
		return &LiteralExpr{obj: list}
	}
	first := ensureReadObject(list.first)
	switch v := first.obj.(type) {
	case Symbol:
		switch string(v) {
		case "quote":
			// TODO: this probably needs unwrapping from ReadObject to Object
			// for collections
			return &LiteralExpr{obj: ensureReadObject(list.Second()).obj}
		case "if":
			checkForm(obj, 3, 4)
			return &IfExpr{
				cond: parse(ensureReadObject(list.Second())),
				pos:  parse(ensureReadObject(list.Third())),
				neg:  parse(ensureReadObject(list.Forth())),
			}
		case "def":
			checkForm(obj, 3, 3)
			s := ensureReadObject(list.Second())
			switch v := s.obj.(type) {
			case Symbol:
				return &DefExpr{name: v, value: parse(ensureReadObject(list.Third()))}
			default:
				panic(&ParseError{obj: s, msg: "First argument to def must be a Symbol"})
			}
		}
	}
	return &CallExpr{callable: parse(ensureReadObject(list.first)), args: parseSeq(list.rest)}
}

func parse(obj ReadObject) Expr {
	switch v := obj.obj.(type) {
	case Int, String, Char, Double, *BigInt, *BigFloat, Bool, Nil, *Ratio, Keyword, Regex:
		return &LiteralExpr{obj: obj.obj}
	case *Vector:
		return parseVector(v)
	case *ArrayMap:
		return parseMap(v)
	case *Set:
		return parseSet(v)
	case *List:
		return parseList(obj)
	default:
		panic(&ParseError{obj: obj, msg: "Cannot parse form: " + obj.ToString(false)})
	}
}

func TryParse(obj ReadObject) (expr Expr, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	return parse(obj), nil
}

func TryEval(expr Expr) (obj Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	return expr.Eval(), nil
}
