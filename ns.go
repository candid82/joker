package main

type (
	Namespace struct {
		name     Symbol
		mappings map[Symbol]*Var
	}
)

func NewNamespace(sym Symbol) *Namespace {
	return &Namespace{
		name:     sym,
		mappings: make(map[Symbol]*Var),
	}
}

func (ns *Namespace) Refer(sym Symbol, vr *Var) *Var {
	if sym.ns != nil {
		panic(RT.newError("Can't intern namespace-qualified symbol " + sym.ToString(false)))
	}
	ns.mappings[sym] = vr
	return vr
}

func (ns *Namespace) ReferAll(other *Namespace) {
	for sym, vr := range other.mappings {
		ns.Refer(sym, vr)
	}
}

// sym must be not qualified
func (ns *Namespace) intern(sym Symbol) *Var {
	sym.meta = nil
	v, ok := ns.mappings[sym]
	if !ok {
		v = &Var{
			ns:   ns,
			name: sym,
		}
		ns.mappings[sym] = v
	}
	return v
}
