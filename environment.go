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
	currentNamespace := res.EnsureNamespace(currentNs)
	res.EnsureNamespace(MakeSymbol("gclojure.core")).intern(MakeSymbol("*ns*"))
	res.SetCurrentNamespace(currentNamespace)
	return res
}

func (env *Env) EnsureNamespace(sym Symbol) *Namespace {
	if sym.ns != nil {
		panic(RT.newError("Namespace's name cannot be qualified: " + sym.ToString(false)))
	}
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
		ns = env.currentNamespace.aliases[s.ns]
		if ns == nil {
			ns = env.namespaces[s.ns]
		}
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

func (env *Env) RemoveNamespace(s Symbol) *Namespace {
	if s.ns != nil {
		return nil
	}
	if s.Equals(MakeSymbol("gclojure.core")) {
		panic(RT.newError("Cannot remove gclojure.core namespace"))
	}
	ns := env.namespaces[s.name]
	delete(env.namespaces, s.name)
	return ns
}
