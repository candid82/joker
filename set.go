package main

import (
	"bytes"
)

type (
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

func (set *ArraySet) Disjoin(obj Object) *ArraySet {
	return &ArraySet{m: set.m.Without(obj).(*ArrayMap)}
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

func (set *ArraySet) WithInfo(info *ObjectInfo) Object {
	set.info = info
	return set
}

func (seq *ArraySet) GetType() *Type {
	return TYPES["Set"]
}

func (set *ArraySet) Seq() Seq {
	return set.m.Keys()
}

func (set *ArraySet) Count() int {
	return set.m.Count()
}
