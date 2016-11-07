package core

import "os"

type (
	Env struct {
		Namespaces       map[*string]*Namespace
		CurrentNamespace *Namespace
		CoreNamespace    *Namespace
		stdout           *Var
		stdin            *Var
		printReadably    *Var
		file             *Var
	}
)

func NewEnv(currentNs Symbol, stdout *os.File, stdin *os.File) *Env {
	res := &Env{
		Namespaces: make(map[*string]*Namespace),
	}
	currentNamespace := res.EnsureNamespace(currentNs)
	res.CoreNamespace = res.EnsureNamespace(MakeSymbol("joker.core"))
	res.CoreNamespace.Intern(MakeSymbol("*ns*"))
	res.stdout = res.CoreNamespace.Intern(MakeSymbol("*out*"))
	res.stdout.Value = &File{stdout}
	res.stdin = res.CoreNamespace.Intern(MakeSymbol("*in*"))
	res.stdin.Value = &File{stdin}
	res.file = res.CoreNamespace.Intern(MakeSymbol("*file*"))
	res.printReadably = res.CoreNamespace.Intern(MakeSymbol("*print-readably*"))
	res.printReadably.Value = Bool{B: true}
	res.SetCurrentNamespace(currentNamespace)
	return res
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

func (env *Env) SetCurrentNamespace(ns *Namespace) {
	env.CurrentNamespace = ns
	v, _ := env.Resolve(MakeSymbol("joker.core/*ns*"))
	v.Value = ns
}

func (env *Env) ResolveIn(n *Namespace, s Symbol) (*Var, bool) {
	var ns *Namespace
	if s.ns == nil {
		ns = n
	} else {
		ns = n.aliases[s.ns]
		if ns == nil {
			ns = env.Namespaces[s.ns]
		}
	}
	if ns == nil {
		return nil, false
	}
	v, ok := ns.mappings[s.name]
	return v, ok
}

func (env *Env) Resolve(s Symbol) (*Var, bool) {
	return env.ResolveIn(env.CurrentNamespace, s)
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
