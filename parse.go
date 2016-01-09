package main

import (
	"fmt"
)

type (
	Expr interface {
		Eval(env Env) Object
		Pos() Position
	}
	Position struct {
		line   int
		column int
	}
	LiteralExpr struct {
		Position
		obj Object
	}
	VectorExpr struct {
		Position
		v []Expr
	}
	MapExpr struct {
		Position
		keys   []Expr
		values []Expr
	}
	SetExpr struct {
		Position
		elements []Expr
	}
	IfExpr struct {
		Position
		cond     Expr
		positive Expr
		negative Expr
	}
	DefExpr struct {
		Position
		name  Symbol
		value Expr
	}
	CallExpr struct {
		Position
		callable Expr
		args     []Expr
	}
	RefExpr struct {
		Position
		symbol Symbol
	}
	ParseError struct {
		obj ReadObject
		msg string
	}
	Callable interface {
		Call(args []Object) Object
	}
	Env map[Symbol]Object
)

func (pos Position) Pos() Position {
	return pos
}

func NewLiteralExpr(obj ReadObject) *LiteralExpr {
	res := LiteralExpr{obj: obj.obj}
	res.line = obj.line
	res.column = obj.column
	return &res
}

func (err ParseError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: %s", err.obj.line, err.obj.column, err.msg)
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

func parseVector(v *Vector, pos Position) Expr {
	r := make([]Expr, v.count)
	for i := 0; i < v.count; i++ {
		r[i] = parse(ensureReadObject(v.at(i)))
	}
	return &VectorExpr{
		v:        r,
		Position: pos,
	}
}

func parseMap(m *ArrayMap, pos Position) Expr {
	res := &MapExpr{
		keys:     make([]Expr, m.Count()),
		values:   make([]Expr, m.Count()),
		Position: pos,
	}
	for iter, i := m.iter(), 0; iter.HasNext(); i++ {
		p := iter.Next()
		res.keys[i] = parse(ensureReadObject(p.key))
		res.values[i] = parse(ensureReadObject(p.value))
	}
	return res
}

func parseSet(s *Set, pos Position) Expr {
	res := &SetExpr{
		elements: make([]Expr, s.m.Count()),
		Position: pos,
	}
	for iter, i := iter(s.Seq()), 0; iter.HasNext(); i++ {
		res.elements[i] = parse(ensureReadObject(iter.Next()))
	}
	return res
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
		return NewLiteralExpr(obj)
	}
	first := ensureReadObject(list.first)
	switch v := first.obj.(type) {
	case Symbol:
		switch *v.name {
		case "quote":
			// TODO: this probably needs unwrapping from ReadObject to Object
			// for collections
			return NewLiteralExpr(ensureReadObject(list.Second()))
		case "if":
			checkForm(obj, 3, 4)
			return &IfExpr{
				cond:     parse(ensureReadObject(list.Second())),
				positive: parse(ensureReadObject(list.Third())),
				negative: parse(ensureReadObject(list.Forth())),
				Position: Position{line: obj.line, column: obj.column},
			}
		case "def":
			checkForm(obj, 3, 3)
			s := ensureReadObject(list.Second())
			switch v := s.obj.(type) {
			case Symbol:
				return &DefExpr{
					name:     v,
					value:    parse(ensureReadObject(list.Third())),
					Position: Position{line: obj.line, column: obj.column},
				}
			default:
				panic(&ParseError{obj: s, msg: "First argument to def must be a Symbol"})
			}
		}
	}
	return &CallExpr{
		callable: parse(ensureReadObject(list.first)),
		args:     parseSeq(list.rest),
		Position: Position{line: obj.line, column: obj.column},
	}
}

func parse(obj ReadObject) Expr {
	pos := Position{line: obj.line, column: obj.column}
	switch v := obj.obj.(type) {
	case Int, String, Char, Double, *BigInt, *BigFloat, Bool, Nil, *Ratio, Keyword, Regex:
		return NewLiteralExpr(obj)
	case *Vector:
		return parseVector(v, pos)
	case *ArrayMap:
		return parseMap(v, pos)
	case *Set:
		return parseSet(v, pos)
	case *List:
		return parseList(obj)
	case Symbol:
		return &RefExpr{
			symbol:   v,
			Position: pos,
		}
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
