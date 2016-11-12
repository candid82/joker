package core

import (
	"fmt"
	"os"
	"unsafe"
)

type (
	Namespace struct {
		MetaHolder
		Name     Symbol
		mappings map[*string]*Var
		aliases  map[*string]*Namespace
	}
)

func (ns *Namespace) ToString(escape bool) string {
	return "#object[Namespace \"" + ns.Name.ToString(escape) + "\"]"
}

func (ns *Namespace) Equals(other interface{}) bool {
	return ns == other
}

func (ns *Namespace) GetInfo() *ObjectInfo {
	return nil
}

func (ns *Namespace) WithInfo(info *ObjectInfo) Object {
	return ns
}

func (ns *Namespace) GetType() *Type {
	return TYPES["Namespace"]
}

func (ns *Namespace) WithMeta(meta Map) Object {
	res := *ns
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (ns *Namespace) AlterMeta(fn *Fn, args []Object) Map {
	meta := ns.meta
	if meta == nil {
		meta = NIL
	}
	fargs := append([]Object{meta}, args...)
	ns.meta = AssertMap(fn.Call(fargs), "")
	return ns.meta
}

func (ns *Namespace) ResetMeta(newMeta Map) Map {
	ns.meta = newMeta
	return ns.meta
}

func (ns *Namespace) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(ns)))
}

func NewNamespace(sym Symbol) *Namespace {
	return &Namespace{
		Name:     sym,
		mappings: make(map[*string]*Var),
		aliases:  make(map[*string]*Namespace),
	}
}

func (ns *Namespace) Refer(sym Symbol, vr *Var) *Var {
	if sym.ns != nil {
		panic(RT.NewError("Can't intern namespace-qualified symbol " + sym.ToString(false)))
	}
	ns.mappings[sym.name] = vr
	return vr
}

func (ns *Namespace) ReferAll(other *Namespace) {
	for name, vr := range other.mappings {
		ns.mappings[name] = vr
	}
}

func (ns *Namespace) Intern(sym Symbol) *Var {
	if sym.ns != nil {
		panic(RT.NewError("Can't intern namespace-qualified symbol " + sym.ToString(false)))
	}
	if TYPES[*sym.name] != nil {
		panic(RT.NewError("Can't intern type name " + *sym.name + " as a Var"))
	}
	sym.meta = nil
	existingVar, ok := ns.mappings[sym.name]
	if !ok {
		newVar := &Var{
			ns:   ns,
			name: sym,
		}
		ns.mappings[sym.name] = newVar
		return newVar
	}
	if existingVar.ns != ns {
		if existingVar.ns.Name.Equals(MakeSymbol("joker.core")) {
			newVar := &Var{
				ns:   ns,
				name: sym,
			}
			ns.mappings[sym.name] = newVar
			fmt.Fprintf(os.Stderr, "WARNING: %s already refers to: %s in namespace %s, being replaced by: %s\n",
				sym.ToString(false), existingVar.ToString(false), ns.Name.ToString(false), newVar.ToString(false))
			return newVar
		}
		panic(RT.NewError(fmt.Sprintf("WARNING: %s already refers to: %s in namespace %s",
			sym.ToString(false), existingVar.ToString(false), ns.ToString(false))))
	}
	return existingVar
}

func (ns *Namespace) AddAlias(alias Symbol, namespace *Namespace) {
	if alias.ns != nil {
		panic(RT.NewError("Alias can't be namespace-qualified"))
	}
	existing := ns.aliases[alias.name]
	if existing != nil && existing != namespace {
		panic(RT.NewError("Alias " + alias.ToString(false) + " already exists in namespace " + ns.Name.ToString(false) + ", aliasing " + existing.Name.ToString(false)))
	}
	ns.aliases[alias.name] = namespace
}
