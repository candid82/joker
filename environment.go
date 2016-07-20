package main

type (
	Env struct {
		namespaces       map[*string]*Namespace
		currentNamespace *Namespace
	}
)

func NewEnv(currentNs Symbol) *Env {
	res := &Env{
		namespaces: make(map[*string]*Namespace),
	}
	currentNamespace := res.ensureNamespace(currentNs)
	res.ensureNamespace(MakeSymbol("gclojure.core")).intern(MakeSymbol("*ns*"))
	res.SetCurrentNamespace(currentNamespace)
	return res
}

func (env *Env) ensureNamespace(sym Symbol) *Namespace {
	if env.namespaces[sym.name] == nil {
		env.namespaces[sym.name] = NewNamespace(sym)
	}
	return env.namespaces[sym.name]
}

func (env *Env) SetCurrentNamespace(ns *Namespace) {
	env.currentNamespace = ns
	v, _ := env.Resolve(MakeSymbol("gclojure.core/*ns*"))
	v.value = ns
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

func (env *Env) FindNamespace(s Symbol) *Namespace {
	if s.ns != nil {
		return nil
	}
	return env.namespaces[s.name]
}
