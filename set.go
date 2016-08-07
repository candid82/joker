package main

import (
	"bytes"
)

type (
	Set interface {
		Conjable
		Gettable
		Disjoin(key Object) Set
	}
	ArraySet struct {
		InfoHolder
		MetaHolder
		m *ArrayMap
	}
)

func (v *ArraySet) WithMeta(meta *ArrayMap) Object {
	res := *v
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (set *ArraySet) Disjoin(key Object) Set {
	return &ArraySet{m: set.m.Without(key).(*ArrayMap)}
}

func (set *ArraySet) Add(obj Object) bool {
	return set.m.Add(obj, Bool{b: true})
}

func (set *ArraySet) Conj(obj Object) Conjable {
	return &ArraySet{m: set.m.Assoc(obj, Bool{b: true}).(*ArrayMap)}
}

func EmptySet() *ArraySet {
	return &ArraySet{m: EmptyArrayMap()}
}

func (set *ArraySet) ToString(escape bool) string {
	var b bytes.Buffer
	b.WriteString("#{")
	if len(set.m.arr) > 0 {
		for i := 0; i < len(set.m.arr)-2; i += 2 {
			b.WriteString(set.m.arr[i].ToString(escape))
			b.WriteRune(' ')
		}
		b.WriteString(set.m.arr[len(set.m.arr)-2].ToString(escape))
	}
	b.WriteRune('}')
	return b.String()
}

func (set *ArraySet) Equals(other interface{}) bool {
	switch otherSet := other.(type) {
	case *ArraySet:
		return set.m.Equals(otherSet.m)
	default:
		return false
	}
}

func (set *ArraySet) Get(key Object) (bool, Object) {
	if set.m.indexOf(key) != -1 {
		return true, key
	}
	return false, nil
}

func (seq *ArraySet) GetType() *Type {
	return TYPES["ArraySet"]
}

func (set *ArraySet) Seq() Seq {
	return set.m.Keys()
}

func (set *ArraySet) Count() int {
	return set.m.Count()
}

func (set *ArraySet) Call(args []Object) Object {
	checkArity(args, 1, 1)
	if ok, _ := set.Get(args[0]); ok {
		return args[0]
	}
	return NIL
}
