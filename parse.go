package main

import (
	"fmt"
)

type (
	Expr interface {
		Eval(env *Env) Object
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
		meta  Expr
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
	VarExpr struct {
		Position
		symbol Symbol
	}
	MetaExpr struct {
		Position
		meta *MapExpr
		expr Expr
	}
	ParseError struct {
		obj ReadObject
		msg string
	}
	Callable interface {
		Call(args []Object) Object
	}
	Namespace struct {
		name     Symbol
		mappings map[Symbol]*Var
	}
	Env struct {
		namespaces       map[Symbol]*Namespace
		currentNamespace *Namespace
	}
)

// sym must be not qualified
func (ns *Namespace) intern(sym Symbol) *Var {
	sym.meta = nil
	v, ok := ns.mappings[sym]
	if !ok {
		v = &Var{
			ns:   ns,
			name: sym,
		}
		ns.mappings[sym] = v
	}
	return v
}

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
	return fmt.Sprintf("stdin:%d:%d: Parse error: %s", err.obj.line, err.obj.column, err.msg)
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

func parseMap(m *ArrayMap, pos Position) *MapExpr {
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

func checkForm(obj ReadObject, min int, max int) int {
	list := obj.obj.(*List)
	if list.count < min {
		panic(&ParseError{obj: obj, msg: "Too few arguments to " + list.first.ToString(false)})
	}
	if list.count > max {
		panic(&ParseError{obj: obj, msg: "Too many arguments to " + list.first.ToString(false)})
	}
	return list.count
}

func parseDef(obj ReadObject) *DefExpr {
	count := checkForm(obj, 2, 4)
	seq := obj.obj.(Seq)
	s := ensureReadObject(Second(seq))
	var meta *ArrayMap
	switch v := s.obj.(type) {
	case Symbol:
		res := &DefExpr{
			name:     v,
			value:    nil,
			Position: Position{line: obj.line, column: obj.column},
		}
		meta = v.GetMeta()
		if count == 3 {
			res.value = parse(ensureReadObject(Third(seq)))
		} else if count == 4 {
			res.value = parse(ensureReadObject(Forth(seq)))
			docstring := ensureReadObject(Third(seq))
			switch docstring.obj.(type) {
			case String:
				if meta != nil {
					meta = meta.Assoc(Keyword(":doc"), docstring)
				} else {
					meta = EmptyArrayMap()
					meta.Add(Keyword(":doc"), docstring)
				}
			default:
				panic(&ParseError{obj: docstring, msg: "Docstring must be a string"})
			}
		}
		if meta != nil {
			res.meta = parse(DeriveReadObject(obj, meta))
		}
		return res
	default:
		panic(&ParseError{obj: s, msg: "First argument to def must be a Symbol"})
	}
}

func parseList(obj ReadObject) Expr {
	seq := obj.obj.(Seq)
	if seq.IsEmpty() {
		return NewLiteralExpr(obj)
	}
	first := ensureReadObject(seq.First())
	switch v := first.obj.(type) {
	case Symbol:
		switch *v.name {
		case "quote":
			// TODO: this probably needs unwrapping from ReadObject to Object
			// for collections
			return NewLiteralExpr(ensureReadObject(Second(seq)))
		case "if":
			checkForm(obj, 3, 4)
			return &IfExpr{
				cond:     parse(ensureReadObject(Second(seq))),
				positive: parse(ensureReadObject(Third(seq))),
				negative: parse(ensureReadObject(Forth(seq))),
				Position: Position{line: obj.line, column: obj.column},
			}
		case "def":
			return parseDef(obj)
		case "var":
			checkForm(obj, 2, 2)
			switch s := (ensureReadObject(Second(seq)).obj).(type) {
			case Symbol:
				return &VarExpr{
					symbol:   s,
					Position: Position{line: obj.line, column: obj.column},
				}
			default:
				panic(&ParseError{obj: obj, msg: "var's argument must be a symbol"})
			}
		}
	}
	return &CallExpr{
		callable: parse(ensureReadObject(seq.First())),
		args:     parseSeq(seq.Rest()),
		Position: Position{line: obj.line, column: obj.column},
	}
}

func parse(obj ReadObject) Expr {
	pos := Position{line: obj.line, column: obj.column}
	var res Expr
	canHaveMeta := false
	switch v := obj.obj.(type) {
	case Int, String, Char, Double, *BigInt, *BigFloat, Bool, Nil, *Ratio, Keyword, Regex:
		res = NewLiteralExpr(obj)
	case *Vector:
		canHaveMeta = true
		res = parseVector(v, pos)
	case *ArrayMap:
		canHaveMeta = true
		res = parseMap(v, pos)
	case *Set:
		canHaveMeta = true
		res = parseSet(v, pos)
	case Seq:
		res = parseList(obj)
	case Symbol:
		res = &RefExpr{
			symbol:   v,
			Position: pos,
		}
	default:
		panic(&ParseError{obj: obj, msg: "Cannot parse form: " + obj.ToString(false)})
	}
	if canHaveMeta {
		meta := obj.obj.(Meta).GetMeta()
		if meta != nil {
			return &MetaExpr{
				meta:     parseMap(meta, pos),
				expr:     res,
				Position: pos,
			}
		}
	}
	return res
}

func TryParse(obj ReadObject) (expr Expr, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	return parse(obj), nil
}
