package core

import (
	"fmt"
	"io"
	"strings"
)

type (
	Namespace struct {
		MetaHolder
		Name           Symbol
		Lazy           func()
		mappings       map[*string]*Var
		aliases        map[*string]*Namespace
		isUsed         bool
		isGloballyUsed bool
		hash           uint32
	}
)

func (ns *Namespace) ToString(escape bool) string {
	return ns.Name.ToString(escape)
}

func (ns *Namespace) Print(w io.Writer, printReadably bool) {
	fmt.Fprint(w, "#object[Namespace \""+ns.Name.ToString(true)+"\"]")
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
	return TYPE.Namespace
}

func (ns *Namespace) WithMeta(meta Map) Object {
	res := *ns
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (ns *Namespace) ResetMeta(newMeta Map) Map {
	ns.meta = newMeta
	return ns.meta
}

func (ns *Namespace) AlterMeta(fn *Fn, args []Object) Map {
	return AlterMeta(&ns.MetaHolder, fn, args)
}

func (ns *Namespace) Hash() uint32 {
	return ns.hash
}

func (ns *Namespace) MaybeLazy(doc string) {
	if ns.Lazy != nil {
		ns.Lazy()
		if VerbosityLevel > 0 {
			fmt.Fprintf(Stderr, "NamespaceFor: Lazily initialized %s for %s\n", *ns.Name.name, doc)
		}
		ns.Lazy = nil
	}
}

const nsHashMask uint32 = 0x90569f6f

func NewNamespace(sym Symbol) *Namespace {
	return &Namespace{
		Name:     sym,
		mappings: make(map[*string]*Var),
		aliases:  make(map[*string]*Namespace),
		hash:     sym.Hash() ^ nsHashMask,
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
		if !vr.isPrivate {
			ns.mappings[name] = vr
		}
	}
}

func (ns *Namespace) Intern(sym Symbol) *Var {
	if sym.ns != nil {
		panic(RT.NewError("Can't intern namespace-qualified symbol " + sym.ToString(false)))
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
		if existingVar.ns.Name.Equals(SYMBOLS.joker_core) {
			newVar := &Var{
				ns:   ns,
				name: sym,
			}
			ns.mappings[sym.name] = newVar
			if !strings.HasPrefix(ns.Name.Name(), "joker.") {
				printParseWarning(sym.GetInfo().Pos(), fmt.Sprintf("WARNING: %s already refers to: %s in namespace %s, being replaced by: %s\n",
					sym.ToString(false), existingVar.ToString(false), ns.Name.ToString(false), newVar.ToString(false)))
			}
			return newVar
		}
		panic(RT.NewErrorWithPos(fmt.Sprintf("WARNING: %s already refers to: %s in namespace %s",
			sym.ToString(false), existingVar.ToString(false), ns.ToString(false)), sym.GetInfo().Pos()))
	}
	if LINTER_MODE && existingVar.expr != nil && !existingVar.ns.Name.Equals(SYMBOLS.joker_core) {
		printParseWarning(sym.GetInfo().Pos(), "Duplicate def of "+existingVar.ToString(false))
	}
	return existingVar
}

func (ns *Namespace) InternVar(name string, val Object, meta *ArrayMap) *Var {
	vr := ns.Intern(MakeSymbol(name))
	vr.Value = val
	meta.Add(KEYWORDS.ns, ns)
	meta.Add(KEYWORDS.name, vr.name)
	vr.meta = meta
	return vr
}

func (ns *Namespace) AddAlias(alias Symbol, namespace *Namespace) {
	if alias.ns != nil {
		panic(RT.NewError("Alias can't be namespace-qualified"))
	}
	existing := ns.aliases[alias.name]
	if existing != nil && existing != namespace {
		msg := "Alias " + alias.ToString(false) + " already exists in namespace " + ns.Name.ToString(false) + ", aliasing " + existing.Name.ToString(false)
		if LINTER_MODE {
			printParseError(GetPosition(alias), msg)
			return
		}
		panic(RT.NewError(msg))
	}
	ns.aliases[alias.name] = namespace
}

func (ns *Namespace) Resolve(name string) *Var {
	return ns.mappings[STRINGS.Intern(name)]
}
