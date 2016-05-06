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
	MacroCallExpr struct {
		Position
		macro Callable
		args  []Object
		name  string
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
		obj Object
		msg string
	}
	Callable interface {
		Call(args []Object) Object
	}
	Env struct {
		namespaces       map[*string]*Namespace
		currentNamespace *Namespace
	}
	Binding struct {
		name  Symbol
		index int
		frame int
	}
	Bindings struct {
		bindings map[*string]*Binding
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
	ctx.localBindings.bindings[sym.name] = &Binding{
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
		bindings: make(map[*string]*Binding),
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
		if b, ok := env.bindings[sym.name]; ok {
			return b
		}
		env = env.parent
	}
	return nil
}

func (env *Env) EnsureNamespace(sym Symbol) {
	if env.namespaces[sym.name] == nil {
		env.namespaces[sym.name] = NewNamespace(sym)
	}
}

func NewEnv(currentNs Symbol) *Env {
	res := &Env{
		namespaces: make(map[*string]*Namespace),
		currentNamespace: &Namespace{
			name:     currentNs,
			mappings: make(map[*string]*Var),
		},
	}
	res.namespaces[currentNs.name] = res.currentNamespace
	res.EnsureNamespace(MakeSymbol("gclojure.core"))
	return res
}

func (env *Env) Resolve(s Symbol) (*Var, bool) {
	var ns *Namespace
	if s.ns == nil {
		ns = env.currentNamespace
	} else {
		ns = env.namespaces[s.ns]
	}
	if ns == nil {
		return nil, false
	}
	v, ok := ns.mappings[s.name]
	return v, ok
}

func (pos Position) Pos() Position {
	return pos
}

func NewLiteralExpr(obj Object) *LiteralExpr {
	res := LiteralExpr{obj: obj}
	info := obj.GetInfo()
	if info != nil {
		res.line = info.line
		res.column = info.column
	}
	return &res
}

func (err ParseError) Error() string {
	line, column := 0, 0
	info := err.obj.GetInfo()
	if info != nil {
		line, column = info.line, info.column
	}
	return fmt.Sprintf("stdin:%d:%d: Parse error: %s", line, column, err.msg)
}

func (err ParseError) Type() Symbol {
	return MakeSymbol("ParseError")
}

func parseSeq(seq Seq, ctx *ParseContext) []Expr {
	res := make([]Expr, 0)
	for !seq.IsEmpty() {
		res = append(res, parse(seq.First(), ctx))
		seq = seq.Rest()
	}
	return res
}

func parseVector(v *Vector, pos Position, ctx *ParseContext) Expr {
	r := make([]Expr, v.count)
	for i := 0; i < v.count; i++ {
		r[i] = parse(v.at(i), ctx)
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
		res.keys[i] = parse(p.key, ctx)
		res.values[i] = parse(p.value, ctx)
	}
	return res
}

func parseSet(s *Set, pos Position, ctx *ParseContext) Expr {
	res := &SetExpr{
		elements: make([]Expr, s.m.Count()),
		Position: pos,
	}
	for iter, i := iter(s.Seq()), 0; iter.HasNext(); i++ {
		res.elements[i] = parse(iter.Next(), ctx)
	}
	return res
}

func checkForm(obj Object, min int, max int) int {
	list := obj.(*List)
	if list.count < min {
		panic(&ParseError{obj: obj, msg: "Too few arguments to " + list.first.ToString(false)})
	}
	if list.count > max {
		panic(&ParseError{obj: obj, msg: "Too many arguments to " + list.first.ToString(false)})
	}
	return list.count
}

func GetPosition(obj Object) Position {
	info := obj.GetInfo()
	if info != nil {
		return Position{line: info.line, column: info.column}
	}
	return Position{line: 0, column: 0}
}

func parseDef(obj Object, ctx *ParseContext) *DefExpr {
	count := checkForm(obj, 2, 4)
	seq := obj.(Seq)
	s := Second(seq)
	var meta *ArrayMap
	switch sym := s.(type) {
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
			Position: GetPosition(obj),
		}
		meta = sym.GetMeta()
		if count == 3 {
			res.value = parse(Third(seq), ctx)
		} else if count == 4 {
			res.value = parse(Forth(seq), ctx)
			docstring := Third(seq)
			switch docstring.(type) {
			case String:
				if meta != nil {
					meta = meta.Assoc(Keyword{k: ":doc"}, docstring)
				} else {
					meta = EmptyArrayMap()
					meta.Add(Keyword{k: ":doc"}, docstring)
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
		ro := seq.First()
		expr := parse(ro, ctx)
		seq = seq.Rest()
		if ctx.recur && !seq.IsEmpty() && !LINTER_MODE {
			panic(&ParseError{obj: ro, msg: "Can only recur from tail position"})
		}
		res = append(res, expr)
	}
	return res
}

func parseParams(params Object) (bindings []Symbol, isVariadic bool) {
	res := make([]Symbol, 0)
	v := params.(*Vector)
	for i := 0; i < v.count; i++ {
		ro := v.at(i)
		sym := ro
		if !IsSymbol(sym) {
			if LINTER_MODE {
				sym = generateSymbol("linter")
			} else {
				panic(&ParseError{obj: ro, msg: "Unsupported binding form: " + sym.ToString(false)})
			}
		}
		if MakeSymbol("&").Equals(sym) {
			if v.count > i+2 {
				ro := v.at(i + 2)
				panic(&ParseError{obj: ro, msg: "Unexpected parameter: " + ro.ToString(false)})
			}
			if v.count == i+2 {
				variadic := v.at(i + 1)
				if !IsSymbol(variadic) {
					panic(&ParseError{obj: variadic, msg: "Unsupported binding form: " + variadic.ToString(false)})
				}
				res = append(res, variadic.(Symbol))
				return res, true
			} else {
				return res, false
			}
		}
		res = append(res, sym.(Symbol))
	}
	return res, false
}

func addArity(fn *FnExpr, params Object, body Seq, ctx *ParseContext) {
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

func wrapWithMeta(fnExpr *FnExpr, obj Object, ctx *ParseContext) Expr {
	meta := obj.(Meta).GetMeta()
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
func parseFn(obj Object, ctx *ParseContext) Expr {
	res := &FnExpr{Position: GetPosition(obj)}
	bodies := obj.(Seq).Rest()
	p := bodies.First()
	if IsSymbol(p) { // self reference
		res.self = p.(Symbol)
		bodies = bodies.Rest()
		p = bodies.First()
		ctx.PushLocalFrame([]Symbol{res.self})
		defer ctx.PopLocalFrame()
	}
	if IsVector(p) { // single arity
		addArity(res, p, bodies.Rest(), ctx)
		return wrapWithMeta(res, obj, ctx)
	}
	// multiple arities
	if bodies.IsEmpty() {
		panic(&ParseError{obj: p, msg: "Parameter declaration missing"})
	}
	for !bodies.IsEmpty() {
		body := bodies.First()
		switch s := body.(type) {
		case Seq:
			params := s.First()
			if !IsVector(params) {
				panic(&ParseError{obj: params, msg: "Parameter declaration must be a vector. Got: " + params.ToString(false)})
			}
			addArity(res, params, s.Rest(), ctx)
		default:
			panic(&ParseError{obj: body, msg: "Function body must be a list. Got: " + s.ToString(false)})
		}
		bodies = bodies.Rest()
	}
	return wrapWithMeta(res, obj, ctx)
}

func isCatch(obj Object) bool {
	return IsSeq(obj) && obj.(Seq).First().Equals(MakeSymbol("catch"))
}

func isFinally(obj Object) bool {
	return IsSeq(obj) && obj.(Seq).First().Equals(MakeSymbol("finally"))
}

func parseCatch(obj Object, ctx *ParseContext) *CatchExpr {
	seq := obj.(Seq).Rest()
	if seq.IsEmpty() || seq.Rest().IsEmpty() {
		panic(&ParseError{obj: obj, msg: "catch requires at least two arguments: type symbol and binding symbol"})
	}
	excType := seq.First()
	if !IsSymbol(excType) {
		panic(&ParseError{obj: excType, msg: "Unable to resolve type: " + excType.ToString(false)})
	}
	excSymbol := Second(seq)
	if !IsSymbol(excSymbol) {
		panic(&ParseError{obj: excSymbol, msg: "Bad binding form, expected symbol, got: " + excSymbol.ToString(false)})
	}
	ctx.PushLocalFrame([]Symbol{excSymbol.(Symbol)})
	defer ctx.PopLocalFrame()
	noRecurAllowed := ctx.noRecurAllowed
	ctx.noRecurAllowed = true
	defer func() { ctx.noRecurAllowed = noRecurAllowed }()
	return &CatchExpr{
		Position:  GetPosition(obj),
		excType:   excType.(Symbol),
		excSymbol: excSymbol.(Symbol),
		body:      parseBody(seq.Rest().Rest(), ctx),
	}
}

func parseFinally(body Seq, ctx *ParseContext) []Expr {
	noRecurAllowed := ctx.noRecurAllowed
	ctx.noRecurAllowed = true
	defer func() { ctx.noRecurAllowed = noRecurAllowed }()
	return parseBody(body, ctx)
}

func parseTry(obj Object, ctx *ParseContext) *TryExpr {
	const (
		Regular = iota
		Catch   = iota
		Finally = iota
	)
	res := &TryExpr{Position: GetPosition(obj)}
	lastType := Regular
	seq := obj.(Seq).Rest()
	for !seq.IsEmpty() {
		obj = seq.First()
		if lastType == Finally {
			panic(&ParseError{obj: obj, msg: "finally clause must be last in try expression"})
		}
		if isCatch(obj) {
			res.catches = append(res.catches, parseCatch(obj, ctx))
			lastType = Catch
		} else if isFinally(obj) {
			res.finallyExpr = parseFinally(obj.(Seq).Rest(), ctx)
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

func parseLet(obj Object, ctx *ParseContext) *LetExpr {
	return parseLetLoop(obj, false, ctx)
}

func parseLoop(obj Object, ctx *ParseContext) *LoopExpr {
	return (*LoopExpr)(parseLetLoop(obj, true, ctx))
}

func parseLetLoop(obj Object, isLoop bool, ctx *ParseContext) *LetExpr {
	formName := "let"
	if isLoop {
		formName = "loop"
	}
	res := &LetExpr{
		Position: GetPosition(obj),
	}
	bindings := Second(obj.(Seq))
	switch b := bindings.(type) {
	case *Vector:
		if b.count%2 != 0 {
			panic(&ParseError{obj: bindings, msg: formName + " requires an even number of forms in binding vector"})
		}
		res.names = make([]Symbol, b.count/2)
		res.values = make([]Expr, b.count/2)
		ctx.PushEmptyLocalFrame()
		defer ctx.PopLocalFrame()
		for i := 0; i < b.count/2; i++ {
			s := b.at(i * 2)
			switch sym := s.(type) {
			case Symbol:
				res.names[i] = sym
			default:
				if LINTER_MODE {
					res.names[i] = generateSymbol("linter")
				} else {
					panic(&ParseError{obj: s, msg: "Unsupported binding form: " + sym.ToString(false)})
				}
			}
			res.values[i] = parse(b.at(i*2+1), ctx)
			ctx.AddLocalBinding(res.names[i], i)
		}

		if isLoop {
			ctx.PushLoopBindings(res.names)
			defer ctx.PopLoopBindings()
		}

		res.body = parseBody(obj.(Seq).Rest().Rest(), ctx)
		if len(res.body) == 0 {
			pos := GetPosition(obj)
			fmt.Fprintf(os.Stderr, "stdin:%d:%d: Parse warning: %s form with empty body\n", pos.line, pos.column, formName)
		}
	default:
		panic(&ParseError{obj: obj, msg: formName + " requires a vector for its bindings"})
	}
	return res
}

func parseRecur(obj Object, ctx *ParseContext) *RecurExpr {
	if ctx.noRecurAllowed {
		panic(&ParseError{obj: obj, msg: "Cannot recur across try"})
	}
	loopBindings := ctx.GetLoopBindings()
	if loopBindings == nil && !LINTER_MODE {
		panic(&ParseError{obj: obj, msg: "No recursion point for recur"})
	}
	seq := obj.(Seq)
	args := parseSeq(seq.Rest(), ctx)
	if len(loopBindings) != len(args) && !LINTER_MODE {
		panic(&ParseError{obj: obj, msg: fmt.Sprintf("Mismatched argument count to recur, expected: %d args, got: %d", len(loopBindings), len(args))})
	}
	ctx.recur = true
	return &RecurExpr{
		args:     args,
		Position: GetPosition(obj),
	}
}

func resolveMacro(obj Object, ctx *ParseContext) Callable {
	switch sym := obj.(type) {
	case Symbol:
		if ctx.GetLocalBinding(sym) != nil {
			return nil
		}
		vr, ok := ctx.globalEnv.Resolve(sym)
		if !ok || !vr.isMacro {
			return nil
		}
		return vr.value.(Callable)
	default:
		return nil
	}
}

func macroexpand1(seq Seq, ctx *ParseContext) Object {
	op := seq.First()
	macro := resolveMacro(op, ctx)
	if macro != nil {
		expr := &MacroCallExpr{
			Position: GetPosition(seq),
			macro:    macro,
			args:     ToSlice(seq.Rest()),
			name:     *op.(Symbol).name,
		}
		return eval(expr, nil)
	} else {
		return seq
	}
}

func parseList(obj Object, ctx *ParseContext) Expr {
	expanded := macroexpand1(obj.(Seq), ctx)
	if expanded != obj {
		return parse(expanded, ctx)
	}
	seq := obj.(Seq)
	if seq.IsEmpty() {
		return NewLiteralExpr(obj)
	}
	pos := GetPosition(obj)
	first := seq.First()
	switch v := first.(type) {
	case Symbol:
		switch *v.name {
		case "quote":
			return NewLiteralExpr(Second(seq))
		case "if":
			checkForm(obj, 3, 4)
			return &IfExpr{
				cond:     parse(Second(seq), ctx),
				positive: parse(Third(seq), ctx),
				negative: parse(Forth(seq), ctx),
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
			switch s := Second(seq).(type) {
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
				e:        parse(Second(seq), ctx),
			}
		case "try":
			return parseTry(obj, ctx)
		}
	}
	res := &CallExpr{
		callable: parse(seq.First(), ctx),
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

func parseSymbol(obj Object, ctx *ParseContext) Expr {
	sym := obj.(Symbol)
	b := ctx.GetLocalBinding(sym)
	if b != nil {
		return &BindingExpr{
			binding:  b,
			Position: GetPosition(obj),
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
		Position: GetPosition(obj),
	}
}

func parse(obj Object, ctx *ParseContext) Expr {
	pos := GetPosition(obj)
	var res Expr
	canHaveMeta := false
	switch v := obj.(type) {
	case Int, String, Char, Double, *BigInt, *BigFloat, Bool, Nil, *Ratio, Keyword, Regex, *Type:
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
		meta := obj.(Meta).GetMeta()
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

func TryParse(obj Object, ctx *ParseContext) (expr Expr, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	return parse(obj, ctx), nil
}
