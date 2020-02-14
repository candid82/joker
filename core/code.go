// +build gen_code

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

func NameAsGo(name string) string {
	for _, t := range tr {
		name = strings.ReplaceAll(name, t[0], "_"+t[1]+"_")
	}
	return name
}

func FilenameUnbracketed(name string) string {
	if name[0] == '<' && name[len(name)-1] == '>' {
		name = name[1 : len(name)-1]
	}
	return name
}

func FilenameAsGo(name string) string {
	return NameAsGo(FilenameUnbracketed(name))
}

func PositionAsGo(endLine, endColumn, startLine, startColumn int, filename *string) string {
	name := ""
	if filename != nil {
		name = *filename
		name = "_" + FilenameAsGo(name)
	}
	return fmt.Sprintf("%d_%d__%d_%d%s", startLine, startColumn, endLine, endColumn, name)
}

func symAsGo(sym Symbol) string {
	name := "_EMPTY_"
	if sym.name != nil {
		name = NameAsGo(strings.ReplaceAll(sym.ToString(false), "/", "_FW_"))
	}
	if sym.info == nil {
		return name
	}
	f := sym.info
	return fmt.Sprintf("%s_%s", name, PositionAsGo(f.endLine, f.endColumn, f.startLine, f.startColumn, f.filename))
}

func (f *FnExpr) AsGo() string {
	return fmt.Sprintf("fnExpr_%s_%d", PositionAsGo(f.endLine, f.endColumn, f.startLine, f.startColumn, f.filename), ordinalForObj("fnExpr", f))
}

func (fn *Fn) AsGo() string {
	if f := fn.fnExpr; f != nil {
		return fmt.Sprintf("fn_%s_%d", PositionAsGo(f.endLine, f.endColumn, f.startLine, f.startColumn, f.filename), ordinalForObj("fn", fn))
	}
	panic("(*Fn)Asgo(): fn.fnExpr == nil")
}

func (sym Symbol) AsGo() string {
	return "symbol_" + symAsGo(sym)
}

func (ns *Namespace) AsGo() string {
	if ns.Name.info != nil && ns.Name.info.filename != nil && *ns.Name.info.filename != *ns.Name.name && FilenameUnbracketed(*ns.Name.info.filename) != *ns.Name.name {
		return "ns_" + NameAsGo(*ns.Name.name) + "_as_" + NameAsGo(*ns.Name.info.filename)
	}
	return "ns_" + NameAsGo(*ns.Name.name)
}

func (e *Env) AsGo() string {
	if e == GLOBAL_ENV {
		return "global_env"
	}
	panic("not GLOBAL_ENV")
}

func (t *Type) AsGo() string {
	return "ty_" + NameAsGo(t.name)
}

func kwAsGo(kw Keyword) string {
	name := NameAsGo(strings.ReplaceAll(strings.ReplaceAll(kw.ToString(false), "/", "_FW_"), ":", ""))
	if kw.info == nil {
		return name
	}
	return fmt.Sprintf("%s_%d_%d__%d_%d", name, kw.info.startLine, kw.info.startColumn, kw.info.endLine, kw.info.endColumn)
}

func (kw Keyword) AsGo() string {
	if kw.name != nil {
		return "keyword_" + kwAsGo(kw)
	}
	panic("empty keyword")
}

func (oi *ObjectInfo) AsGo() string {
	if res, ok := infoHolderNameAsGo(*oi); ok {
		return "objectInfo_" + res
	}
	panic("could not make useful name out of ObjectInfo")
}

func (v *Var) AsGo() string {
	name := symAsGo(v.name)
	ns := ""
	if v.ns != nil {
		if v.name.ns != nil && *v.name.ns != *v.ns.Name.name {
			msg := fmt.Sprintf("Symbol namespace discrepancy: Var %s has %s, its sym has %s", name, *v.ns.Name.name, *v.name.ns)
			fmt.Fprintln(Stderr, msg)
			panic(msg)
		}
		if v.name.ns == nil {
			i := v.ns.Name.info
			if i == nil || i.filename == nil || FilenameUnbracketed(*i.filename) != *v.ns.Name.name {
				ns = NameAsGo(*v.ns.Name.name)
			}
		}
	}
	return "var_" + ns + "_" + name
}

func (v *VarRefExpr) AsGo() string {
	s := *v.vr.name.name
	if res, ok := infoHolderNameAsGo(*v); ok {
		return "varref_" + NameAsGo(s) + "_" + res
	}
	return fmt.Sprintf("%s_%d_%d", strings.Replace(s, "var_", "varRefExpr_", 1), v.startLine, v.startColumn)
}

func (v *Var) Expr() Expr {
	return v.expr
}

func (v Var) Namespace() *Namespace {
	return v.ns
}

func infoHolderNameAsGo(obj interface{}) (string, bool) {
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
		filename = filenamePtr.Elem().Interface().(string)
		filename = "_" + FilenameAsGo(filename)
	}
	startLine := gen_go.UnsafeReflectValue(v.FieldByName("startLine")).Interface().(int)
	startColumn := gen_go.UnsafeReflectValue(v.FieldByName("startColumn")).Interface().(int)
	endLine := gen_go.UnsafeReflectValue(v.FieldByName("endLine")).Interface().(int)
	endColumn := gen_go.UnsafeReflectValue(v.FieldByName("endColumn")).Interface().(int)
	return fmt.Sprintf("%d_%d__%d_%d%s", startLine, startColumn, endLine, endColumn, filename), true
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

func UniqueId(obj interface{}) (id string) {
	defer func() {
		if r := recover(); r != nil {
			id = coreTypeAsGo(obj)
			n := ordinalForObj(id, obj)
			pos, havePos := infoHolderNameAsGo(obj)
			if havePos {
				id = fmt.Sprintf("%s_%s__%d", id, pos, n)
			} else {
				id = fmt.Sprintf("%s__%d", id, n)
				origType := reflect.TypeOf(obj).String()
				if origType == "core.Keyword" || origType == "core.Symbol" {
					fmt.Fprintf(Stderr, "UniqueId: Using %s for %s due to %s\n", id, origType, r)
				}
			}
		}
	}()
	id = obj.(interface{ AsGo() string }).AsGo()
	return
}

func coreType(e interface{}) string {
	return strings.Replace(fmt.Sprintf("%T", e), "core.", "", 1)
}

func coreTypeAsGo(e interface{}) string {
	s := strings.Replace(coreType(e), "*", "", 1)
	return strings.ToLower(s[0:1]) + s[1:]
}

func (v *VarRefExpr) Var() *Var {
	return v.vr
}

func (ns *Namespace) Mappings() map[*string]*Var {
	return ns.mappings
}

func (pos Position) String() string {
	return fmt.Sprintf("%s:[%d:%d-%d:%d]",
		pos.Filename(),
		pos.startLine,
		pos.startColumn,
		pos.endLine,
		pos.endColumn)
}

// Convert e.g. "<joker.core>" to "joker.core". Panic if not "<*>".
func CoreNameAsNamespaceName(s string) string {
	if s[0] != '<' || s[len(s)-1] != '>' {
		panic(fmt.Sprintf("Invalid syntax for core source file namespace id: `%s'", s))
	}
	return s[1 : len(s)-1]
}
