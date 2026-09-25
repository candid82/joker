package core

// TransientSet keeps the map returned by each edit, since a small map may
// promote to a hash map during conj!.
type TransientSet struct {
	InfoHolder
	live bool
	m    TransientMapCollection
}

func (s *MapSet) AsTransient() TransientCollection {
	return &TransientSet{live: true, m: s.m.(Editable).AsTransient().(TransientMapCollection)}
}
func (s *TransientSet) check() {
	if !s.live {
		panic(RT.NewError("Transient used after persistent! call"))
	}
}
func (s *TransientSet) GetType() *Type                { return TYPE.TransientSet }
func (s *TransientSet) ToString(bool) string          { return "#<transient set>" }
func (s *TransientSet) Equals(other interface{}) bool { return s == other }
func (s *TransientSet) Hash() uint32                  { panic(RT.NewError("Cannot hash a transient set")) }
func (s *TransientSet) Count() int                    { s.check(); return s.m.Count() }
func (s *TransientSet) Get(key Object) (bool, Object) {
	s.check()
	if ok, _ := s.m.Get(key); ok {
		return true, key
	}
	return false, nil
}
func (s *TransientSet) Call(args []Object) Object {
	CheckArity(args, 1, 1)
	if ok, val := s.Get(args[0]); ok {
		return val
	}
	return NIL
}
func (s *TransientSet) ConjBang(obj Object) TransientCollection {
	s.check()
	s.m = s.m.AssocBang(obj, Boolean{B: true}).(TransientMapCollection)
	return s
}
func (s *TransientSet) DisjoinBang(obj Object) TransientSetCollection {
	s.check()
	s.m = s.m.WithoutBang(obj)
	return s
}
func (s *TransientSet) Persistent() Object {
	s.check()
	s.live = false
	return &MapSet{m: s.m.Persistent().(Map)}
}
