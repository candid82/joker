package core

type (
	Namespace struct {
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

// sym must be not qualified
func (ns *Namespace) Intern(sym Symbol) *Var {
	if TYPES[*sym.name] != nil {
		panic(RT.NewError("Can't intern type name " + *sym.name + " as a Var"))
	}
	sym.meta = nil
	v, ok := ns.mappings[sym.name]
	if !ok {
		v = &Var{
			ns:   ns,
			name: sym,
		}
		ns.mappings[sym.name] = v
	}
	return v
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
