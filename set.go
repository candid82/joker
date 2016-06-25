package main

import (
	"bytes"
)

type (
	Set struct {
		InfoHolder
		MetaHolder
		m *ArrayMap
	}
)

func (v *Set) WithMeta(meta *ArrayMap) Object {
	res := *v
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (set *Set) Disjoin(obj Object) *Set {
	return &Set{m: set.m.Without(obj).(*ArrayMap)}
}

func (set *Set) Add(obj Object) bool {
	return set.m.Add(obj, Bool{b: true})
}

func (set *Set) Conj(obj Object) Conjable {
	return &Set{m: set.m.Assoc(obj, Bool{b: true}).(*ArrayMap)}
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

func (set *Set) Get(key Object) (bool, Object) {
	if set.m.indexOf(key) != -1 {
		return true, key
	}
	return false, nil
}

func (set *Set) WithInfo(info *ObjectInfo) Object {
	set.info = info
	return set
}

func (seq *Set) GetType() *Type {
	return TYPES["Set"]
}

func (set *Set) Seq() Seq {
	return set.m.Keys()
}

func (set *Set) Count() int {
	return set.m.Count()
}
