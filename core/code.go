//go:build gen_code
// +build gen_code

// Helpers for gen_code.

package core

import (
	"fmt"
	"github.com/candid82/joker/core/gen_go"
	"reflect"
	"strings"
)

var tr = [][2]string{
	{"?", "Q"},
	{"!", "BANG"},
	{"<=", "LE"},
	{">=", "GE"},
	{"<", "LT"},
	{">", "GT"},
	{"=", "EQ"},
	{"'", "APOS"},
	{"+", "PLUS"},
	{"-", "DASH"},
	{"*", "STAR"},
	{"/", "SLASH"},
	{"&", "AMP"},
	{"#", "HASH"},
	{".", "DOT"},
	{"%", "PCT"},
	{".", "DOT"},
	{":", "COLON"},
}

// Convert an "arbitrary" string (really, only valid Joker symbols are supported) to a form
// that can serve as a substring of a Go symbol name.
func StringAsGoName(name string) string {
	for _, t := range tr {
		name = strings.ReplaceAll(name, t[0], "_"+t[1]+"_")
	}
	return name
}

// Convert e.g. "<joker.core>" to "joker.core", or "not-bracketed"
// as-is, with whether brackets were removed.
func filenameAndWhetherBracketed(name string) (filename string, bracketed bool) {
	if len(name) > 1 && name[0] == '<' && name[len(name)-1] == '>' {
		return name[1 : len(name)-1], true
	}
	return name, false
}

// Return filename without a pair of enclosing angle brackets (if any).
func filenameUnbracketed(name string) string {
	filename, _ := filenameAndWhetherBracketed(name)
	return filename
}

// Return filename without a pair of enclosing angle brackets, panic
// if there aren't any.
func CoreNameAsNamespaceName(name string) string {
	if filename, bracketed := filenameAndWhetherBracketed(name); bracketed {
		return filename
	}
	panic(fmt.Sprintf("Invalid syntax for core source file namespace id: `%s'", name))
}

func filenameAsGo(name string) string {
	return StringAsGoName(filenameUnbracketed(name))
}

func positionAsGo(filename *string, startLine, startColumn, endLine, endColumn int) string {
	name := ""
	if filename != nil {
		name = filenameAsGo(*filename)
		if name != "" && name != "_" {
			name += "_"
		}
	}
	return fmt.Sprintf("%s%d_%d__%d_%d", name, startLine, startColumn, endLine, endColumn)
}

func isPositionNil(p Position) bool {
	return p.endLine == 0 && p.endColumn == 0 && p.startLine == 0 && p.startColumn == 0 && (p.filename == nil || *p.filename == "")
}

func isObjectInfoNil(p *ObjectInfo) bool {
	return p == nil || (p.endLine == 0 && p.endColumn == 0 && p.startLine == 0 && p.startColumn == 0 && (p.filename == nil || *p.filename == ""))
}

func symAsGo(sym Symbol) string {
	if sym.name == nil {
		return "EMPTY"
	} else {
		return StringAsGoName(strings.ReplaceAll(sym.ToString(false), "/", "_FW_"))
	}
}

func (sym Symbol) AsGo() string {
	name := symAsGo(sym)
	pos := ""
	if f := sym.info; !isObjectInfoNil(f) {
		pos = fmt.Sprintf("_POS_%s", positionAsGo(f.filename, f.startLine, f.startColumn, f.endLine, f.endColumn))
	}
	return "symbol_" + name + pos
}

func fnExprAsGo(f *FnExpr) string {
	return symAsGo(f.self)
}

func (f *FnExpr) AsGo() string {
	name := fmt.Sprintf("fnExpr_POS_%s", positionAsGo(f.filename, f.startLine, f.startColumn, f.endLine, f.endColumn))
	return fmt.Sprintf("%s_NUM_%d", name, ordinalForObj(name, f))
}

func (fn *Fn) AsGo() string {
	if f := fn.fnExpr; f != nil {
		baseName := fmt.Sprintf("fn_%s_POS_%s", fnExprAsGo(f), positionAsGo(f.filename, f.startLine, f.startColumn, f.endLine, f.endColumn))
		return fmt.Sprintf("%s_NUM_%d", baseName, ordinalForObj(baseName, fn))
	}
	panic("(*Fn)Asgo(): fn.fnExpr == nil")
}

func (ns *Namespace) AsGo() string {
	file := ""
	if ns.Name.info != nil && ns.Name.info.filename != nil && *ns.Name.info.filename != *ns.Name.name && filenameUnbracketed(*ns.Name.info.filename) != *ns.Name.name {
		file = "_FILE_" + StringAsGoName(*ns.Name.info.filename)
	}
	return "ns_" + StringAsGoName(*ns.Name.name) + file
}

func (e *Env) AsGo() string {
	if e == GLOBAL_ENV {
		return "global_env"
	}
	panic("not GLOBAL_ENV")
}

func (t *Type) AsGo() string {
	return "ty_" + StringAsGoName(t.name)
}

func kwAsGo(kw Keyword) string {
	return StringAsGoName(strings.ReplaceAll(strings.ReplaceAll(kw.ToString(false), "/", "_FW_"), ":", ""))
}

func (kw Keyword) AsGo() string {
	name := kwAsGo(kw)
	if kw.name == nil {
		panic("empty keyword")
	}
	pos := ""
	if f := kw.info; f != nil {
		pos = fmt.Sprintf("_POS_%s", positionAsGo(f.filename, f.startLine, f.startColumn, f.endLine, f.endColumn))
	}
	return "kw_" + name + pos
}

func (oi *ObjectInfo) AsGo() string {
	if res, ok := infoHolderAsGoName(*oi); ok {
		return "objectInfo_" + res
	}
	panic("could not make useful name out of ObjectInfo")
}

func (v *Var) AsGo() string {
	sym := v.name
	name := symAsGo(sym)
	ns := ""
	if v.ns != nil {
		if sym.ns != nil && *sym.ns != *v.ns.Name.name {
			msg := fmt.Sprintf("Symbol namespace discrepancy: Var %s has %s, its sym has %s", name, *v.ns.Name.name, *sym.ns)
			fmt.Fprintln(Stderr, msg)
			panic(msg)
		}
		if sym.ns == nil {
			i := v.ns.Name.info
			if i == nil || i.filename == nil || filenameUnbracketed(*i.filename) != *v.ns.Name.name {
				ns = "_NS_" + StringAsGoName(*v.ns.Name.name)
			}
		}
	}
	pos := ""
	f := v.info
	if f == nil {
		f = sym.info
	}
	if f != nil {
		pos = fmt.Sprintf("_POS_%s", positionAsGo(f.filename, f.startLine, f.startColumn, f.endLine, f.endColumn))
	}
	return "var" + ns + "_NAME_" + name + pos
}

func (v *VarRefExpr) AsGo() string {
	s := *v.vr.name.name
	if res, ok := infoHolderAsGoName(*v); ok {
		return "varRef_" + StringAsGoName(s) + "_" + res
	}
	return fmt.Sprintf("%s_%d_%d", strings.Replace(s, "var_", "varRefExpr_", 1), v.startLine, v.startColumn)
}

// Returns typename of object as it should be represented in package
// core.
func typeInCore(e interface{}) string {
	return strings.Replace(fmt.Sprintf("%T", e), "core.", "", 1)
}

func typeInCoreAsGo(e interface{}) string {
	s := strings.Replace(typeInCore(e), "*", "", 1)
	return strings.ToLower(s[0:1]) + s[1:]
}

func infoHolderAsGoName(obj interface{}) (string, bool) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", false
	}
	vt := v.Type()
	sf, yes := vt.FieldByName("InfoHolder")
	if yes {
		if !sf.Anonymous {
			return "", false
		}
		v = v.FieldByName("InfoHolder")
		vt = v.Type()
		if vt.Kind() != reflect.Struct {
			return "", false
		}
		sf, yes = vt.FieldByName("info")
		if !yes || sf.Anonymous {
			return "", false
		}
		v = v.FieldByName("info")
		vt = v.Type()
		if vt.Kind() != reflect.Ptr {
			panic("'info' field not a pointer")
		}
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
		vt = v.Type()
	}
	sf, yes = vt.FieldByName("Position")
	if !yes || !sf.Anonymous {
		return "", false
	}
	v = v.FieldByName("Position")
	vt = v.Type()
	if vt.Kind() != reflect.Struct {
		return "", false
	}
	sf, yes = vt.FieldByName("startLine")
	if !yes || sf.Anonymous {
		return "", false
	}
	filename := ""
	filenamePtr := gen_go.UnsafeReflectValue(v.FieldByName("filename"))
	if !(filenamePtr.IsZero() || filenamePtr.IsNil()) {
		filename = filenameAsGo(filenamePtr.Elem().Interface().(string))
		if filename != "" && filename != "_" {
			filename = filename + "_"
		}
	}
	startLine := gen_go.UnsafeReflectValue(v.FieldByName("startLine")).Interface().(int)
	startColumn := gen_go.UnsafeReflectValue(v.FieldByName("startColumn")).Interface().(int)
	endLine := gen_go.UnsafeReflectValue(v.FieldByName("endLine")).Interface().(int)
	endColumn := gen_go.UnsafeReflectValue(v.FieldByName("endColumn")).Interface().(int)
	return "POS_" + positionAsGo(&filename, startLine, startColumn, endLine, endColumn), true
}

var generatedIds = map[string]*gIdInfo{}

type gIdInfo struct {
	gIds   map[interface{}]uint
	nextId uint
}

func ordinalForObj(id string, obj interface{}) uint {
	info, found := generatedIds[id]
	if !found {
		info = &gIdInfo{map[interface{}]uint{}, 0}
		generatedIds[id] = info
	}
	n, found := info.gIds[obj]
	if !found {
		info.nextId++
		n = info.nextId
		info.gIds[obj] = n
	}
	return n
}

// Tries to call obj.AsGo() and return the result. If that fails,
// cobbles together something reasonable and informative, and returns
// that.
func UniqueId(obj interface{}) (id string) {
	defer func() {
		if r := recover(); r != nil {
			id = typeInCoreAsGo(obj)
			pos, havePos := infoHolderAsGoName(obj)
			if havePos {
				id = fmt.Sprintf("%s_%s", id, pos)
			} else {
				origType := reflect.TypeOf(obj).String()
				if origType == "core.Keyword" || origType == "core.Symbol" {
					fmt.Fprintf(Stderr, "UniqueId: Using %s for %s due to %s\n", id, origType, r)
				}
			}
			n := ordinalForObj(id, obj)
			id = fmt.Sprintf("%s_NUM_%d", id, n)
		}
	}()
	id = obj.(interface{ AsGo() string }).AsGo()
	return
}

// Receivers for Joker objects that gen_code.go needs, but no other
// Joker code needs.  (These could be put into object.go, parse.go,
// ns.go, etc., as appropriate, if desired.)

type CodegenVarAnalysis struct {
	Required     []*Namespace
	DirectSource *Var
}

func AnalyzeCodegenVar(v *Var) CodegenVarAnalysis {
	if v.expr == nil {
		return CodegenVarAnalysis{}
	}
	v.hasDefinition = true
	if fnExpr := codegenFnExpr(v.expr); fnExpr != nil {
		v.fnSummary = compactFnSummary(fnExpr)
	} else {
		inferred := v.expr.InferValue(newInferEnv())
		v.inferredTypes = inferred.types
		v.inferredUnknown = inferred.unknown
		v.hasInferredValue = true
	}

	required := make(map[*Namespace]struct{})
	walkCodegenExpr(v.expr, func(ref *Var) {
		if ref != nil && ref.ns != nil {
			required[ref.ns] = struct{}{}
		}
	})
	res := CodegenVarAnalysis{Required: make([]*Namespace, 0, len(required))}
	for ns := range required {
		res.Required = append(res.Required, ns)
	}
	if ref, ok := v.expr.(*VarRefExpr); ok {
		res.DirectSource = ref.vr
	}
	return res
}

func StripCodegenVarExpr(v *Var) {
	v.expr = nil
}

func codegenFnExpr(expr Expr) *FnExpr {
	if meta, ok := expr.(*MetaExpr); ok {
		expr = meta.expr
	}
	fn, _ := expr.(*FnExpr)
	return fn
}

func walkCodegenExpr(expr Expr, visitVar func(*Var)) {
	if expr == nil || reflect.ValueOf(expr).IsNil() {
		return
	}
	walkBody := func(body []Expr) {
		for _, child := range body {
			walkCodegenExpr(child, visitVar)
		}
	}
	switch expr := expr.(type) {
	case *LiteralExpr, *MacroCallExpr:
	case *VectorExpr:
		walkBody(expr.v)
	case *MapExpr:
		walkBody(expr.keys)
		walkBody(expr.values)
	case *SetExpr:
		walkBody(expr.elements)
	case *IfExpr:
		walkCodegenExpr(expr.cond, visitVar)
		walkCodegenExpr(expr.positive, visitVar)
		walkCodegenExpr(expr.negative, visitVar)
	case *DefExpr:
		walkCodegenExpr(expr.value, visitVar)
		walkCodegenExpr(expr.meta, visitVar)
	case *CallExpr:
		walkCodegenExpr(expr.callable, visitVar)
		walkBody(expr.args)
	case *RecurExpr:
		walkBody(expr.args)
	case *VarRefExpr:
		visitVar(expr.vr)
	case *BindingExpr:
	case *MetaExpr:
		walkCodegenExpr(expr.meta, visitVar)
		walkCodegenExpr(expr.expr, visitVar)
	case *DoExpr:
		walkBody(expr.body)
	case *FnArityExpr:
		walkBody(expr.body)
	case *FnExpr:
		for i := range expr.arities {
			walkCodegenExpr(&expr.arities[i], visitVar)
		}
		walkCodegenExpr(expr.variadic, visitVar)
	case *LetExpr:
		walkBody(expr.values)
		walkBody(expr.body)
	case *LoopExpr:
		let := (*LetExpr)(expr)
		walkBody(let.values)
		walkBody(let.body)
	case *ThrowExpr:
		walkCodegenExpr(expr.e, visitVar)
	case *CatchExpr:
		walkBody(expr.body)
	case *TryExpr:
		walkBody(expr.body)
		for _, catch := range expr.catches {
			walkCodegenExpr(catch, visitVar)
		}
		walkBody(expr.finallyExpr)
	case *SetMacroExpr:
	default:
		panic(fmt.Sprintf("unhandled expression type %T", expr))
	}
}

func (v Var) Namespace() *Namespace {
	return v.ns
}
