package main

import (
	"fmt"
	"os"
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
	RecurExpr struct {
		Position
		args []Expr
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
	LoopExpr  LetExpr
	ThrowExpr struct {
		Position
		e Expr
	}
	CatchExpr struct {
		Position
		excType   Symbol
		excSymbol Symbol
		body      []Expr
	}
	TryExpr struct {
		Position
		body        []Expr
		catches     []*CatchExpr
		finallyExpr []Expr
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
	ParseContext struct {
		globalEnv      *Env
		localBindings  *Bindings
		loopBindings   [][]Symbol
		recur          bool
		noRecurAllowed bool
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

func (localEnv *LocalEnv) replaceFrame(values []Object) *LocalEnv {
	res := LocalEnv{
		bindings: values,
		parent:   localEnv.parent,
		frame:    localEnv.frame,
	}
	return &res
}

func (ctx *ParseContext) PushLoopBindings(bindings []Symbol) {
	ctx.loopBindings = append(ctx.loopBindings, bindings)
}

func (ctx *ParseContext) PopLoopBindings() {
	ctx.loopBindings = ctx.loopBindings[:len(ctx.loopBindings)-1]
}

func (ctx *ParseContext) GetLoopBindings() []Symbol {
	n := len(ctx.loopBindings)
	if n == 0 {
		return nil
	}
	return ctx.loopBindings[n-1]
}

func (ctx *ParseContext) AddLocalBinding(sym Symbol, index int) {
	ctx.localBindings.bindings[sym] = &Binding{
		name:  sym,
		frame: ctx.localBindings.frame,
		index: index,
	}
}

func (ctx *ParseContext) PushEmptyLocalFrame() {
	frame := 0
	if ctx.localBindings != nil {
		frame = ctx.localBindings.frame + 1
	}
	ctx.localBindings = &Bindings{
		bindings: make(map[Symbol]*Binding),
		parent:   ctx.localBindings,
		frame:    frame,
	}
}

func (ctx *ParseContext) PushLocalFrame(names []Symbol) {
	ctx.PushEmptyLocalFrame()
	for i, sym := range names {
		ctx.AddLocalBinding(sym, i)
	}
}

func (ctx *ParseContext) PopLocalFrame() {
	ctx.localBindings = ctx.localBindings.parent
}

func (ctx *ParseContext) GetLocalBinding(sym Symbol) *Binding {
	env := ctx.localBindings
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

func (err ParseError) Type() Symbol {
	return MakeSymbol("ParseError")
}

func ensureReadObject(obj Object) ReadObject {
	switch obj := obj.(type) {
	case ReadObject:
		return obj
	default:
		return ReadObject{obj: obj}
	}
}

func parseSeq(seq Seq, ctx *ParseContext) []Expr {
	res := make([]Expr, 0)
	for !seq.IsEmpty() {
		res = append(res, parse(ensureReadObject(seq.First()), ctx))
		seq = seq.Rest()
	}
	return res
}

func parseVector(v *Vector, pos Position, ctx *ParseContext) Expr {
	r := make([]Expr, v.count)
	for i := 0; i < v.count; i++ {
		r[i] = parse(ensureReadObject(v.at(i)), ctx)
	}
	return &VectorExpr{
		v:        r,
		Position: pos,
	}
}

func parseMap(m *ArrayMap, pos Position, ctx *ParseContext) *MapExpr {
	res := &MapExpr{
		keys:     make([]Expr, m.Count()),
		values:   make([]Expr, m.Count()),
		Position: pos,
	}
	for iter, i := m.iter(), 0; iter.HasNext(); i++ {
		p := iter.Next()
		res.keys[i] = parse(ensureReadObject(p.key), ctx)
		res.values[i] = parse(ensureReadObject(p.value), ctx)
	}
	return res
}

func parseSet(s *Set, pos Position, ctx *ParseContext) Expr {
	res := &SetExpr{
		elements: make([]Expr, s.m.Count()),
		Position: pos,
	}
	for iter, i := iter(s.Seq()), 0; iter.HasNext(); i++ {
		res.elements[i] = parse(ensureReadObject(iter.Next()), ctx)
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

func parseDef(obj ReadObject, ctx *ParseContext) *DefExpr {
	count := checkForm(obj, 2, 4)
	seq := obj.obj.(Seq)
	s := ensureReadObject(Second(seq))
	var meta *ArrayMap
	switch sym := s.obj.(type) {
	case Symbol:
		if sym.ns != nil && (Symbol{name: sym.ns} != ctx.globalEnv.currentNamespace.name) {
			panic(&ParseError{
				msg: "Can't create defs outside of current ns",
				obj: obj,
			})
		}
		vr := ctx.globalEnv.currentNamespace.intern(Symbol{name: sym.name})

		res := &DefExpr{
			vr:       vr,
			value:    nil,
			Position: Position{line: obj.line, column: obj.column},
		}
		meta = sym.GetMeta()
		if count == 3 {
			res.value = parse(ensureReadObject(Third(seq)), ctx)
		} else if count == 4 {
			res.value = parse(ensureReadObject(Forth(seq)), ctx)
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
			res.meta = parse(DeriveReadObject(obj, meta), ctx)
		}
		return res
	default:
		panic(&ParseError{obj: s, msg: "First argument to def must be a Symbol"})
	}
}

func parseBody(seq Seq, ctx *ParseContext) []Expr {
	recur := ctx.recur
	ctx.recur = false
	defer func() { ctx.recur = recur }()
	res := make([]Expr, 0)
	for !seq.IsEmpty() {
		ro := ensureReadObject(seq.First())
		expr := parse(ro, ctx)
		seq = seq.Rest()
		if ctx.recur && !seq.IsEmpty() && !LINTER_MODE {
			panic(&ParseError{obj: ro, msg: "Can only recur from tail position"})
		}
		res = append(res, expr)
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

func addArity(fn *FnExpr, params ReadObject, body Seq, ctx *ParseContext) {
	args, isVariadic := parseParams(params)
	ctx.PushLocalFrame(args)
	defer ctx.PopLocalFrame()
	ctx.PushLoopBindings(args)
	defer ctx.PopLoopBindings()
	arity := FnArityExpr{args: args, body: parseBody(body, ctx)}
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

func wrapWithMeta(fnExpr *FnExpr, obj ReadObject, ctx *ParseContext) Expr {
	meta := obj.obj.(Meta).GetMeta()
	if meta != nil {
		return &MetaExpr{
			meta:     parseMap(meta, fnExpr.Pos(), ctx),
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
func parseFn(obj ReadObject, ctx *ParseContext) Expr {
	res := &FnExpr{Position: Position{line: obj.line, column: obj.column}}
	bodies := obj.obj.(Seq).Rest()
	p := ensureReadObject(bodies.First())
	if IsSymbol(p.obj) { // self reference
		res.self = p.obj.(Symbol)
		bodies = bodies.Rest()
		p = ensureReadObject(bodies.First())
		ctx.PushLocalFrame([]Symbol{res.self})
		defer ctx.PopLocalFrame()
	}
	if IsVector(p.obj) { // single arity
		addArity(res, p, bodies.Rest(), ctx)
		return wrapWithMeta(res, obj, ctx)
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
			addArity(res, params, s.Rest(), ctx)
		default:
			panic(&ParseError{obj: body, msg: "Function body must be a list. Got: " + s.ToString(false)})
		}
		bodies = bodies.Rest()
	}
	return wrapWithMeta(res, obj, ctx)
}

func isCatch(obj ReadObject) bool {
	return IsSeq(obj.obj) && obj.obj.(Seq).First().Equals(MakeSymbol("catch"))
}

func isFinally(obj ReadObject) bool {
	return IsSeq(obj.obj) && obj.obj.(Seq).First().Equals(MakeSymbol("finally"))
}

func parseCatch(obj ReadObject, ctx *ParseContext) *CatchExpr {
	seq := obj.obj.(Seq).Rest()
	if seq.IsEmpty() || seq.Rest().IsEmpty() {
		panic(&ParseError{obj: obj, msg: "catch requires at least two arguments: type symbol and binding symbol"})
	}
	excType := ensureReadObject(seq.First())
	if !IsSymbol(excType.obj) {
		panic(&ParseError{obj: excType, msg: "Unable to resolve type: " + excType.obj.ToString(false)})
	}
	excSymbol := ensureReadObject(Second(seq))
	if !IsSymbol(excSymbol.obj) {
		panic(&ParseError{obj: excSymbol, msg: "Bad binding form, expected symbol, got: " + excSymbol.obj.ToString(false)})
	}
	ctx.PushLocalFrame([]Symbol{excSymbol.obj.(Symbol)})
	defer ctx.PopLocalFrame()
	noRecurAllowed := ctx.noRecurAllowed
	ctx.noRecurAllowed = true
	defer func() { ctx.noRecurAllowed = noRecurAllowed }()
	return &CatchExpr{
		Position:  Position{line: obj.line, column: obj.column},
		excType:   excType.obj.(Symbol),
		excSymbol: excSymbol.obj.(Symbol),
		body:      parseBody(seq.Rest().Rest(), ctx),
	}
}

func parseFinally(body Seq, ctx *ParseContext) []Expr {
	noRecurAllowed := ctx.noRecurAllowed
	ctx.noRecurAllowed = true
	defer func() { ctx.noRecurAllowed = noRecurAllowed }()
	return parseBody(body, ctx)
}

func parseTry(obj ReadObject, ctx *ParseContext) *TryExpr {
	const (
		Regular = iota
		Catch   = iota
		Finally = iota
	)
	res := &TryExpr{Position: Position{line: obj.line, column: obj.column}}
	lastType := Regular
	seq := obj.obj.(Seq).Rest()
	for !seq.IsEmpty() {
		obj = ensureReadObject(seq.First())
		if lastType == Finally {
			panic(&ParseError{obj: obj, msg: "finally clause must be last in try expression"})
		}
		if isCatch(obj) {
			res.catches = append(res.catches, parseCatch(obj, ctx))
			lastType = Catch
		} else if isFinally(obj) {
			res.finallyExpr = parseFinally(obj.obj.(Seq).Rest(), ctx)
			lastType = Finally
		} else {
			if lastType == Catch {
				panic(&ParseError{obj: obj, msg: "Only catch or finally clause can follow catch in try expression"})
			}
			res.body = append(res.body, parse(obj, ctx))
		}
		seq = seq.Rest()
	}
	return res
}

func parseLet(obj ReadObject, ctx *ParseContext) *LetExpr {
	return parseLetLoop(obj, false, ctx)
}

func parseLoop(obj ReadObject, ctx *ParseContext) *LoopExpr {
	return (*LoopExpr)(parseLetLoop(obj, true, ctx))
}

func parseLetLoop(obj ReadObject, isLoop bool, ctx *ParseContext) *LetExpr {
	formName := "let"
	if isLoop {
		formName = "loop"
	}
	res := &LetExpr{
		Position: Position{line: obj.line, column: obj.column},
	}
	bindings := ensureReadObject(Second(obj.obj.(Seq)))
	switch b := bindings.obj.(type) {
	case *Vector:
		if b.count%2 != 0 {
			panic(&ParseError{obj: bindings, msg: formName + " requires an even number of forms in binding vector"})
		}
		res.names = make([]Symbol, b.count/2)
		res.values = make([]Expr, b.count/2)
		ctx.PushEmptyLocalFrame()
		defer ctx.PopLocalFrame()
		for i := 0; i < b.count/2; i++ {
			s := ensureReadObject(b.at(i * 2))
			switch sym := s.obj.(type) {
			case Symbol:
				res.names[i] = sym
			default:
				if LINTER_MODE {
					res.names[i] = generateSymbol("linter")
				} else {
					panic(&ParseError{obj: s, msg: "Unsupported binding form: " + sym.ToString(false)})
				}
			}
			res.values[i] = parse(ensureReadObject(b.at(i*2+1)), ctx)
			ctx.AddLocalBinding(res.names[i], i)
		}

		if isLoop {
			ctx.PushLoopBindings(res.names)
			defer ctx.PopLoopBindings()
		}

		res.body = parseBody(obj.obj.(Seq).Rest().Rest(), ctx)
		if len(res.body) == 0 {
			fmt.Fprintf(os.Stderr, "stdin:%d:%d: Parse warning: %s form with empty body\n", obj.line, obj.column, formName)
		}
	default:
		panic(&ParseError{obj: obj, msg: formName + " requires a vector for its bindings"})
	}
	return res
}

func parseRecur(obj ReadObject, ctx *ParseContext) *RecurExpr {
	if ctx.noRecurAllowed {
		panic(&ParseError{obj: obj, msg: "Cannot recur across try"})
	}
	loopBindings := ctx.GetLoopBindings()
	if loopBindings == nil && !LINTER_MODE {
		panic(&ParseError{obj: obj, msg: "No recursion point for recur"})
	}
	seq := obj.obj.(Seq)
	args := parseSeq(seq.Rest(), ctx)
	if len(loopBindings) != len(args) && !LINTER_MODE {
		panic(&ParseError{obj: obj, msg: fmt.Sprintf("Mismatched argument count to recur, expected: %d args, got: %d", len(loopBindings), len(args))})
	}
	ctx.recur = true
	return &RecurExpr{
		args:     args,
		Position: Position{line: obj.line, column: obj.column},
	}
}

func parseList(obj ReadObject, ctx *ParseContext) Expr {
	// MACRO: do macroexpand1 here
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
				cond:     parse(ensureReadObject(Second(seq)), ctx),
				positive: parse(ensureReadObject(Third(seq)), ctx),
				negative: parse(ensureReadObject(Forth(seq)), ctx),
				Position: pos,
			}
		case "fn":
			return parseFn(obj, ctx)
		case "let":
			return parseLet(obj, ctx)
		case "loop":
			return parseLoop(obj, ctx)
		case "recur":
			return parseRecur(obj, ctx)
		case "def":
			return parseDef(obj, ctx)
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
				body:     parseBody(seq.Rest(), ctx),
				Position: pos,
			}
		case "throw":
			return &ThrowExpr{
				Position: pos,
				e:        parse(ensureReadObject(Second(seq)), ctx),
			}
		case "try":
			return parseTry(obj, ctx)
		}
	}
	res := &CallExpr{
		callable: parse(ensureReadObject(seq.First()), ctx),
		args:     parseSeq(seq.Rest(), ctx),
		Position: pos,
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

func parseSymbol(obj ReadObject, ctx *ParseContext) Expr {
	sym := obj.obj.(Symbol)
	b := ctx.GetLocalBinding(sym)
	if b != nil {
		return &BindingExpr{
			binding:  b,
			Position: Position{line: obj.line, column: obj.column},
		}
	}
	vr, ok := ctx.globalEnv.Resolve(sym)
	if !ok {
		if LINTER_MODE {
			vr = ctx.globalEnv.currentNamespace.intern(sym)
		} else {
			panic(&ParseError{obj: obj, msg: "Unable to resolve symbol: " + sym.ToString(false)})
		}
	}
	return &VarRefExpr{
		vr:       vr,
		Position: Position{line: obj.line, column: obj.column},
	}
}

func parse(obj ReadObject, ctx *ParseContext) Expr {
	pos := Position{line: obj.line, column: obj.column}
	var res Expr
	canHaveMeta := false
	switch v := obj.obj.(type) {
	case Int, String, Char, Double, *BigInt, *BigFloat, Bool, Nil, *Ratio, Keyword, Regex:
		res = NewLiteralExpr(obj)
	case *Vector:
		canHaveMeta = true
		res = parseVector(v, pos, ctx)
	case *ArrayMap:
		canHaveMeta = true
		res = parseMap(v, pos, ctx)
	case *Set:
		canHaveMeta = true
		res = parseSet(v, pos, ctx)
	case Seq:
		res = parseList(obj, ctx)
	case Symbol:
		res = parseSymbol(obj, ctx)
	default:
		panic(&ParseError{obj: obj, msg: "Cannot parse form: " + obj.ToString(false)})
	}
	if canHaveMeta {
		meta := obj.obj.(Meta).GetMeta()
		if meta != nil {
			return &MetaExpr{
				meta:     parseMap(meta, pos, ctx),
				expr:     res,
				Position: pos,
			}
		}
	}
	return res
}

func TryParse(obj ReadObject, ctx *ParseContext) (expr Expr, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	return parse(obj, ctx), nil
}
