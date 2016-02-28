package main

import (
	"fmt"
)

type (
	Expr interface {
		Eval(env *LocalEnv) Object
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
		vr    *Var
		value Expr
		meta  Expr
	}
	CallExpr struct {
		Position
		callable Expr
		args     []Expr
		name     string
	}
	VarRefExpr struct {
		Position
		vr *Var
	}
	BindingExpr struct {
		Position
		binding *Binding
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
	DoExpr struct {
		Position
		body []Expr
	}
	FnArityExpr struct {
		Position
		args []Symbol
		body []Expr
	}
	FnExpr struct {
		Position
		arities  []FnArityExpr
		variadic *FnArityExpr
		self     Symbol
	}
	LetExpr struct {
		Position
		names  []Symbol
		values []Expr
		body   []Expr
	}
	ParseError struct {
		obj ReadObject
		msg string
	}
	Callable interface {
		Call(args []Object) Object
	}
	Env struct {
		namespaces       map[Symbol]*Namespace
		currentNamespace *Namespace
	}
	Binding struct {
		name  Symbol
		index int
		frame int
	}
	Bindings struct {
		bindings map[Symbol]*Binding
		parent   *Bindings
		frame    int
	}
	LocalEnv struct {
		bindings []Object
		parent   *LocalEnv
		frame    int
	}
)

var GLOBAL_ENV = NewEnv(MakeSymbol("user"))
var LOCAL_BINDINGS *Bindings = nil

func (localEnv *LocalEnv) addEmptyFrame(capacity int) *LocalEnv {
	res := LocalEnv{
		bindings: make([]Object, 0, capacity),
		parent:   localEnv,
	}
	if localEnv != nil {
		res.frame = localEnv.frame + 1
	}
	return &res
}

func (localEnv *LocalEnv) addBinding(obj Object) {
	localEnv.bindings = append(localEnv.bindings, obj)
}

func (localEnv *LocalEnv) addFrame(values []Object) *LocalEnv {
	res := LocalEnv{
		bindings: values,
		parent:   localEnv,
	}
	if localEnv != nil {
		res.frame = localEnv.frame + 1
	}
	return &res
}

func addLocalBinding(sym Symbol, index int) {
	LOCAL_BINDINGS.bindings[sym] = &Binding{
		name:  sym,
		frame: LOCAL_BINDINGS.frame,
		index: index,
	}
}

func pushEmptyLocalFrame() {
	frame := 0
	if LOCAL_BINDINGS != nil {
		frame = LOCAL_BINDINGS.frame + 1
	}
	LOCAL_BINDINGS = &Bindings{
		bindings: make(map[Symbol]*Binding),
		parent:   LOCAL_BINDINGS,
		frame:    frame,
	}
}

func pushLocalFrame(names []Symbol) {
	pushEmptyLocalFrame()
	for i, sym := range names {
		addLocalBinding(sym, i)
	}
}

func popLocalFrame() {
	LOCAL_BINDINGS = LOCAL_BINDINGS.parent
}

func getLocalBinding(sym Symbol) *Binding {
	env := LOCAL_BINDINGS
	for env != nil {
		if b, ok := env.bindings[sym]; ok {
			return b
		}
		env = env.parent
	}
	return nil
}

func (env *Env) EnsureNamespace(sym Symbol) {
	if env.namespaces[sym] == nil {
		env.namespaces[sym] = NewNamespace(sym)
	}
}

func NewEnv(currentNs Symbol) *Env {
	res := &Env{
		namespaces: make(map[Symbol]*Namespace),
		currentNamespace: &Namespace{
			name:     currentNs,
			mappings: make(map[Symbol]*Var),
		},
	}
	res.namespaces[currentNs] = res.currentNamespace
	res.EnsureNamespace(MakeSymbol("gclojure.core"))
	return res
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
	switch sym := s.obj.(type) {
	case Symbol:
		if sym.ns != nil && (Symbol{name: sym.ns} != GLOBAL_ENV.currentNamespace.name) {
			panic(&ParseError{
				msg: "Can't create defs outside of current ns",
				obj: obj,
			})
		}
		vr := GLOBAL_ENV.currentNamespace.intern(Symbol{name: sym.name})

		res := &DefExpr{
			vr:       vr,
			value:    nil,
			Position: Position{line: obj.line, column: obj.column},
		}
		meta = sym.GetMeta()
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

func parseBody(seq Seq) []Expr {
	res := make([]Expr, 0)
	for !seq.IsEmpty() {
		res = append(res, parse(ensureReadObject(seq.First())))
		seq = seq.Rest()
	}
	return res
}

func parseParams(params ReadObject) (bindings []Symbol, isVariadic bool) {
	res := make([]Symbol, 0)
	v := params.obj.(*Vector)
	for i := 0; i < v.count; i++ {
		ro := ensureReadObject(v.at(i))
		sym := ro.obj
		if !IsSymbol(sym) {
			if LINTER_MODE {
				sym = generateSymbol("linter")
			} else {
				panic(&ParseError{obj: ro, msg: "Unsupported binding form: " + sym.ToString(false)})
			}
		}
		if sym == MakeSymbol("&") {
			if v.count > i+2 {
				ro := ensureReadObject(v.at(i + 2))
				panic(&ParseError{obj: ro, msg: "Unexpected parameter: " + ro.obj.ToString(false)})
			}
			if v.count == i+2 {
				variadic := ensureReadObject(v.at(i + 1))
				if !IsSymbol(variadic.obj) {
					panic(&ParseError{obj: variadic, msg: "Unsupported binding form: " + variadic.obj.ToString(false)})
				}
				res = append(res, variadic.obj.(Symbol))
				return res, true
			} else {
				return res, false
			}
		}
		res = append(res, sym.(Symbol))
	}
	return res, false
}

func addArity(fn *FnExpr, params ReadObject, body Seq) {
	args, isVariadic := parseParams(params)
	pushLocalFrame(args)
	defer popLocalFrame()
	arity := FnArityExpr{args: args, body: parseBody(body)}
	if isVariadic {
		if fn.variadic != nil {
			panic(&ParseError{obj: params, msg: "Can't have more than 1 variadic overload"})
		}
		for _, arity := range fn.arities {
			if len(arity.args) >= len(args) {
				panic(&ParseError{obj: params, msg: "Can't have fixed arity function with more params than variadic function"})
			}
		}
		fn.variadic = &arity
	} else {
		for _, arity := range fn.arities {
			if len(arity.args) == len(args) {
				panic(&ParseError{obj: params, msg: "Can't have 2 overloads with same arity"})
			}
		}
		if fn.variadic != nil && len(args) >= len(fn.variadic.args) {
			panic(&ParseError{obj: params, msg: "Can't have fixed arity function with more params than variadic function"})
		}
		fn.arities = append(fn.arities, arity)
	}
}

func wrapWithMeta(fnExpr *FnExpr, obj ReadObject) Expr {
	meta := obj.obj.(Meta).GetMeta()
	if meta != nil {
		return &MetaExpr{
			meta:     parseMap(meta, fnExpr.Pos()),
			expr:     fnExpr,
			Position: fnExpr.Pos(),
		}
	}
	return fnExpr
}

// Examples:
// (fn f [] 1 2)
// (fn f ([] 1 2)
//       ([a] a 3)
//       ([a & b] a b))
func parseFn(obj ReadObject) Expr {
	res := &FnExpr{Position: Position{line: obj.line, column: obj.column}}
	bodies := obj.obj.(Seq).Rest()
	p := ensureReadObject(bodies.First())
	if IsSymbol(p.obj) { // self reference
		res.self = p.obj.(Symbol)
		bodies = bodies.Rest()
		p = ensureReadObject(bodies.First())
		pushLocalFrame([]Symbol{res.self})
		defer popLocalFrame()
	}
	if IsVector(p.obj) { // single arity
		addArity(res, p, bodies.Rest())
		return wrapWithMeta(res, obj)
	}
	// multiple arities
	if bodies.IsEmpty() {
		panic(&ParseError{obj: p, msg: "Parameter declaration missing"})
	}
	for !bodies.IsEmpty() {
		body := ensureReadObject(bodies.First())
		switch s := body.obj.(type) {
		case Seq:
			params := ensureReadObject(s.First())
			if !IsVector(params.obj) {
				panic(&ParseError{obj: params, msg: "Parameter declaration must be a vector. Got: " + params.obj.ToString(false)})
			}
			addArity(res, params, s.Rest())
		default:
			panic(&ParseError{obj: body, msg: "Function body must be a list. Got: " + s.ToString(false)})
		}
		bodies = bodies.Rest()
	}
	return wrapWithMeta(res, obj)
}

func parseLet(obj ReadObject) Expr {
	res := &LetExpr{
		Position: Position{line: obj.line, column: obj.column}}
	bindings := ensureReadObject(obj.obj.(Seq).Rest().First())
	switch b := bindings.obj.(type) {
	case *Vector:
		if b.count%2 != 0 {
			panic(&ParseError{obj: bindings, msg: "let requires an even number of forms in binding vector"})
		}
		res.names = make([]Symbol, b.count/2)
		res.values = make([]Expr, b.count/2)
		pushEmptyLocalFrame()
		defer popLocalFrame()
		for i := 0; i < b.count/2; i++ {
			s := ensureReadObject(b.at(i * 2))
			switch sym := s.obj.(type) {
			case Symbol:
				res.names[i] = sym
			default:
				panic(&ParseError{obj: s, msg: "Unsupported binding form: " + sym.ToString(false)})
			}
			res.values[i] = parse(ensureReadObject(b.at(i*2 + 1)))
			addLocalBinding(res.names[i], i)
		}
		res.body = parseBody(obj.obj.(Seq).Rest().Rest())
	default:
		panic(&ParseError{obj: obj, msg: "Let requires a vector for its bindings"})
	}
	return res
}

func parseList(obj ReadObject) Expr {
	seq := obj.obj.(Seq)
	if seq.IsEmpty() {
		return NewLiteralExpr(obj)
	}
	pos := Position{line: obj.line, column: obj.column}
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
				Position: pos,
			}
		case "fn":
			return parseFn(obj)
		case "let":
			return parseLet(obj)
		case "def":
			return parseDef(obj)
		case "var":
			checkForm(obj, 2, 2)
			switch s := (ensureReadObject(Second(seq)).obj).(type) {
			case Symbol:
				return &VarExpr{
					symbol:   s,
					Position: pos,
				}
			default:
				panic(&ParseError{obj: obj, msg: "var's argument must be a symbol"})
			}
		case "do":
			return &DoExpr{
				body:     parseBody(seq.Rest()),
				Position: pos,
			}
		}
	}
	res := &CallExpr{
		callable: parse(ensureReadObject(seq.First())),
		args:     parseSeq(seq.Rest()),
		Position: Position{line: obj.line, column: obj.column},
		name:     "fn",
	}
	switch c := res.callable.(type) {
	case *VarRefExpr:
		res.name = c.vr.ToString(false)
	case *BindingExpr:
		res.name = c.binding.name.ToString(false)
	}
	return res
}

func parseSymbol(obj ReadObject) Expr {
	sym := obj.obj.(Symbol)
	b := getLocalBinding(sym)
	if b != nil {
		return &BindingExpr{
			binding:  b,
			Position: Position{line: obj.line, column: obj.column},
		}
	}
	vr, ok := GLOBAL_ENV.Resolve(sym)
	if !ok {
		if LINTER_MODE {
			vr = GLOBAL_ENV.currentNamespace.intern(sym)
		} else {
			panic(&ParseError{obj: obj, msg: "Unable to resolve symbol: " + sym.ToString(false)})
		}
	}
	return &VarRefExpr{
		vr:       vr,
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
		res = parseSymbol(obj)
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
