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
	EvalError struct {
		msg string
		pos Position
	}
	Callable interface {
		Call(args []Object) Object
	}
	Env map[Symbol]Object
)

var GLOBAL_ENV = Env{}

func (pos Position) Pos() Position {
	return pos
}

func NewLiteralExpr(obj ReadObject) *LiteralExpr {
	res := LiteralExpr{obj: obj.obj}
	res.line = obj.line
	res.column = obj.column
	return &res
}

func NewVectorExpr(exprs []Expr, pos Position) *VectorExpr {
	res := VectorExpr{v: exprs}
	res.line = pos.line
	res.column = pos.column
	return &res
}

func NewMapExpr(count int, pos Position) *MapExpr {
	res := MapExpr{
		keys:   make([]Expr, count),
		values: make([]Expr, count),
	}
	res.line = pos.line
	res.column = pos.column
	return &res
}

func NewSetExpr(count int, pos Position) *SetExpr {
	res := SetExpr{
		elements: make([]Expr, count),
	}
	res.line = pos.line
	res.column = pos.column
	return &res
}

func NewIfExpr(cond, positive, negative Expr, pos Position) *IfExpr {
	res := &IfExpr{
		cond:     cond,
		positive: positive,
		negative: negative,
	}
	res.line = pos.line
	res.column = pos.column
	return res
}

func NewDefExpr(name Symbol, value Expr, pos Position) *DefExpr {
	res := &DefExpr{name: name, value: value}
	res.line = pos.line
	res.column = pos.column
	return res
}

func NewCallExpr(callable Expr, args []Expr, pos Position) *CallExpr {
	res := &CallExpr{callable: callable, args: args}
	res.line = pos.line
	res.column = pos.column
	return res
}

func NewRefExpr(symbol Symbol, pos Position) *RefExpr {
	res := &RefExpr{symbol: symbol}
	res.line = pos.line
	res.column = pos.column
	return res
}

func (err ParseError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: %s", err.obj.line, err.obj.column, err.msg)
}

func (err EvalError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: %s", err.pos.line, err.pos.column, err.msg)
}

func (expr *RefExpr) Eval(env Env) Object {
	v, ok := env[expr.symbol]
	if !ok {
		panic(&EvalError{
			msg: "Unbound symbol: " + expr.symbol.ToString(false),
			pos: expr.Position,
		})
	}
	return v
}

func (expr *LiteralExpr) Eval(env Env) Object {
	return expr.obj
}

func (expr *VectorExpr) Eval(env Env) Object {
	res := EmptyVector
	for _, e := range expr.v {
		res = res.conj(e.Eval(env))
	}
	return res
}

func (expr *MapExpr) Eval(env Env) Object {
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

func (expr *SetExpr) Eval(env Env) Object {
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

func (expr *DefExpr) Eval(env Env) Object {
	v := expr.value.Eval(env)
	GLOBAL_ENV[expr.name] = v
	return v
}

func evalSeq(exprs []Expr, env Env) []Object {
	res := make([]Object, len(exprs))
	for i, expr := range exprs {
		res[i] = expr.Eval(env)
	}
	return res
}

func (expr *CallExpr) Eval(env Env) Object {
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

func (expr *IfExpr) Eval(env Env) Object {
	if toBool(expr.cond.Eval(env)) {
		return expr.positive.Eval(env)
	}
	return expr.negative.Eval(env)
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
	return NewVectorExpr(r, pos)
}

func parseMap(m *ArrayMap, pos Position) Expr {
	res := NewMapExpr(m.Count(), pos)
	for iter, i := m.iter(), 0; iter.HasNext(); i++ {
		p := iter.Next()
		res.keys[i] = parse(ensureReadObject(p.key))
		res.values[i] = parse(ensureReadObject(p.value))
	}
	return res
}

func parseSet(s *Set, pos Position) Expr {
	res := NewSetExpr(s.m.Count(), pos)
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
		switch string(v) {
		case "quote":
			// TODO: this probably needs unwrapping from ReadObject to Object
			// for collections
			return NewLiteralExpr(ensureReadObject(list.Second()))
		case "if":
			checkForm(obj, 3, 4)
			return NewIfExpr(
				parse(ensureReadObject(list.Second())),
				parse(ensureReadObject(list.Third())),
				parse(ensureReadObject(list.Forth())),
				Position{line: obj.line, column: obj.column})
		case "def":
			checkForm(obj, 3, 3)
			s := ensureReadObject(list.Second())
			switch v := s.obj.(type) {
			case Symbol:
				return NewDefExpr(v, parse(ensureReadObject(list.Third())), Position{line: obj.line, column: obj.column})
			default:
				panic(&ParseError{obj: s, msg: "First argument to def must be a Symbol"})
			}
		}
	}
	return NewCallExpr(parse(ensureReadObject(list.first)), parseSeq(list.rest), Position{line: obj.line, column: obj.column})
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
		return NewRefExpr(v, pos)
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
	return expr.Eval(GLOBAL_ENV), nil
}
