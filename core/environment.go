package core

type (
	Env struct {
		Namespaces       map[*string]*Namespace
		CurrentNamespace *Namespace
	}
)

func NewEnv(currentNs Symbol) *Env {
	res := &Env{
		Namespaces: make(map[*string]*Namespace),
	}
	currentNamespace := res.EnsureNamespace(currentNs)
	res.EnsureNamespace(MakeSymbol("gclojure.core")).Intern(MakeSymbol("*ns*"))
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
	v, _ := env.Resolve(MakeSymbol("gclojure.core/*ns*"))
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
	if s.Equals(MakeSymbol("gclojure.core")) {
		panic(RT.NewError("Cannot remove gclojure.core namespace"))
	}
	ns := env.Namespaces[s.name]
	delete(env.Namespaces, s.name)
	return ns
}
