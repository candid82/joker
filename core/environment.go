package core

import (
	"os"
	"strings"
)

type (
	Env struct {
		Namespaces    map[*string]*Namespace
		CoreNamespace *Namespace
		stdout        *Var
		stdin         *Var
		printReadably *Var
		file          *Var
		args          *Var
		ns            *Var
	}
)

func NewEnv(currentNs Symbol, stdout *os.File, stdin *os.File) *Env {
	res := &Env{
		Namespaces: make(map[*string]*Namespace),
	}
	res.CoreNamespace = res.EnsureNamespace(MakeSymbol("joker.core"))
	res.ns = res.CoreNamespace.Intern(MakeSymbol("*ns*"))
	res.ns.Value = res.EnsureNamespace(currentNs)
	res.stdout = res.CoreNamespace.Intern(MakeSymbol("*out*"))
	res.stdout.Value = &File{stdout}
	res.stdin = res.CoreNamespace.Intern(MakeSymbol("*in*"))
	res.stdin.Value = &File{stdin}
	res.file = res.CoreNamespace.Intern(MakeSymbol("*file*"))
	res.args = res.CoreNamespace.Intern(MakeSymbol("*command-line-args*"))
	args := EmptyVector
	for _, arg := range os.Args {
		args = args.Conjoin(String{S: arg})
	}
	if args.Count() > 1 {
		res.args.Value = args
	} else {
		res.args.Value = NIL
	}
	res.printReadably = res.CoreNamespace.Intern(MakeSymbol("*print-readably*"))
	res.printReadably.Value = Bool{B: true}
	return res
}

func (env *Env) CurrentNamespace() *Namespace {
	return AssertNamespace(env.ns.Value, "")
}

func (env *Env) EnsureNamespace(sym Symbol) *Namespace {
	if sym.ns != nil {
		panic(RT.NewError("Namespace's name cannot be qualified: " + sym.ToString(false)))
	}
	if env.Namespaces[sym.name] == nil {
		env.Namespaces[sym.name] = NewNamespace(sym)
	}
	return env.Namespaces[sym.name]
}

func (env *Env) NamespaceFor(ns *Namespace, s Symbol) *Namespace {
	var res *Namespace
	if s.ns == nil {
		res = ns
	} else {
		res = ns.aliases[s.ns]
		if res == nil {
			res = env.Namespaces[s.ns]
		}
	}
	return res
}

func (env *Env) ResolveIn(n *Namespace, s Symbol) (*Var, bool) {
	ns := env.NamespaceFor(n, s)
	if ns == nil {
		return nil, false
	}
	v, ok := ns.mappings[s.name]
	return v, ok
}

func (env *Env) Resolve(s Symbol) (*Var, bool) {
	return env.ResolveIn(env.CurrentNamespace(), s)
}

func (env *Env) FindNamespace(s Symbol) *Namespace {
	if s.ns != nil {
		return nil
	}
	return env.Namespaces[s.name]
}

func (env *Env) RemoveNamespace(s Symbol) *Namespace {
	if s.ns != nil {
		return nil
	}
	if s.Equals(MakeSymbol("joker.core")) {
		panic(RT.NewError("Cannot remove joker.core namespace"))
	}
	ns := env.Namespaces[s.name]
	delete(env.Namespaces, s.name)
	return ns
}

func (env *Env) ResolveSymbol(s Symbol) Symbol {
	if strings.ContainsRune(*s.name, '.') {
		return s
	}
	currentNs := env.CurrentNamespace()
	if s.ns != nil {
		ns := env.NamespaceFor(currentNs, s)
		if ns == nil || ns.Name.name == s.ns {
			return s
		}
		return Symbol{
			name: s.name,
			ns:   ns.Name.name,
		}
	}
	vr, ok := currentNs.mappings[s.name]
	if !ok {
		return Symbol{
			name: s.name,
			ns:   currentNs.Name.name,
		}
	}
	return Symbol{
		name: vr.name.name,
		ns:   vr.ns.Name.name,
	}
}
