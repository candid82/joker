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
	ParseError struct {
		obj ReadObject
		msg string
	}
	EvalError struct {
		msg string
	}
)

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

func ensureReadObject(obj Object) ReadObject {
	switch obj := obj.(type) {
	case ReadObject:
		return obj
	default:
		return ReadObject{obj: obj}
	}
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

func parse(obj ReadObject) Expr {
	switch v := obj.obj.(type) {
	case Int, String, Char, Double, *BigInt, *BigFloat, Bool, Nil, *Ratio, Keyword, Regex:
		return &LiteralExpr{obj: obj.obj}
	case *Vector:
		return parseVector(v)
	case *ArrayMap:
		return parseMap(v)
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
