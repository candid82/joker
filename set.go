package main

import (
	"bytes"
)

type (
	Set struct {
		MetaHolder
		m *ArrayMap
	}
)

func (v *Set) WithMeta(meta *ArrayMap) Object {
	res := *v
	res.meta = meta
	return &res
}

func (set *Set) Cons(obj Object) *Set {
	return &Set{m: set.m.Assoc(obj, Bool(true))}
}

func (set *Set) Disjoin(obj Object) *Set {
	return &Set{m: set.m.Without(obj)}
}

func (set *Set) Add(obj Object) bool {
	return set.m.Add(obj, Bool(true))
}

func EmptySet() *Set {
	return &Set{m: EmptyArrayMap()}
}

func (set *Set) ToString(escape bool) string {
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

func (set *Set) Equals(other interface{}) bool {
	switch otherSet := other.(type) {
	case *Set:
		return set.m.Equals(otherSet.m)
	default:
		return false
	}
}

func (set *Set) Seq() Seq {
	return set.m.Keys()
}
