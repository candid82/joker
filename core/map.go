package core

import "bytes"

type (
	Map interface {
		Associative
		Seqable
		Without(key Object) Map
		Keys() Seq
		Vals() Seq
	}
	ArrayMap struct {
		InfoHolder
		MetaHolder
		arr []Object
	}
	ArrayMapIterator struct {
		m       *ArrayMap
		current int
	}
	Pair struct {
		key   Object
		value Object
	}
	ArrayMapSeq struct {
		InfoHolder
		MetaHolder
		m     *ArrayMap
		index int
	}
)

func (seq *ArrayMapSeq) sequential() {}

func (seq *ArrayMapSeq) Equals(other interface{}) bool {
	return IsSeqEqual(seq, other)
}

func (seq *ArrayMapSeq) ToString(escape bool) string {
	return SeqToString(seq, escape)
}

func (seq *ArrayMapSeq) WithMeta(meta *ArrayMap) Object {
	res := *seq
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (seq *ArrayMapSeq) GetType() *Type {
	return TYPES["ArrayMapSeq"]
}

func (seq *ArrayMapSeq) Hash() uint32 {
	return hashOrdered(seq)
}

func (seq *ArrayMapSeq) Seq() Seq {
	return seq
}

func (seq *ArrayMapSeq) First() Object {
	if seq.index < len(seq.m.arr) {
		return NewVectorFrom(seq.m.arr[seq.index], seq.m.arr[seq.index+1])
	}
	return NIL
}

func (seq *ArrayMapSeq) Rest() Seq {
	if seq.index < len(seq.m.arr) {
		return &ArrayMapSeq{m: seq.m, index: seq.index + 2}
	}
	return EmptyList
}

func (seq *ArrayMapSeq) IsEmpty() bool {
	return seq.index >= len(seq.m.arr)
}

func (seq *ArrayMapSeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: seq}
}

func (v *ArrayMap) WithMeta(meta *ArrayMap) Object {
	res := *v
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (iter *ArrayMapIterator) Next() Pair {
	res := Pair{
		key:   iter.m.arr[iter.current],
		value: iter.m.arr[iter.current+1],
	}
	iter.current += 2
	return res
}

func (iter *ArrayMapIterator) HasNext() bool {
	return iter.current < len(iter.m.arr)
}

func (m *ArrayMap) indexOf(key Object) int {
	for i := 0; i < len(m.arr); i += 2 {
		if m.arr[i].Equals(key) {
			return i
		}
	}
	return -1
}

func ArraySeqFromArrayMap(m *ArrayMap) *ArraySeq {
	return &ArraySeq{arr: m.arr}
}

func (m *ArrayMap) Get(key Object) (bool, Object) {
	i := m.indexOf(key)
	if i != -1 {
		return true, m.arr[i+1]
	}
	return false, nil
}

func (m *ArrayMap) Set(key Object, value Object) {
	i := m.indexOf(key)
	if i != -1 {
		m.arr[i+1] = value
	} else {
		m.arr = append(m.arr, key)
		m.arr = append(m.arr, value)
	}
}

func (m *ArrayMap) Add(key Object, value Object) bool {
	i := m.indexOf(key)
	if i != -1 {
		return false
	}
	m.arr = append(m.arr, key)
	m.arr = append(m.arr, value)
	return true
}

func (m *ArrayMap) Count() int {
	return len(m.arr) / 2
}

func (m *ArrayMap) Clone() *ArrayMap {
	result := ArrayMap{arr: make([]Object, len(m.arr), cap(m.arr))}
	copy(result.arr, m.arr)
	return &result
}

func (m *ArrayMap) Assoc(key Object, value Object) Associative {
	result := m.Clone()
	result.Set(key, value)
	return result
}

func (m *ArrayMap) EntryAt(key Object) *Vector {
	i := m.indexOf(key)
	if i != -1 {
		return NewVectorFrom(key, m.arr[i+1])
	}
	return nil
}

func (m *ArrayMap) Without(key Object) Map {
	result := ArrayMap{arr: make([]Object, len(m.arr), cap(m.arr))}
	var i, j int
	for i, j = 0, 0; i < len(m.arr); i += 2 {
		if m.arr[i].Equals(key) {
			continue
		}
		result.arr[j] = m.arr[i]
		result.arr[j+1] = m.arr[i+1]
		j += 2
	}
	if i != j {
		result.arr = result.arr[:j]
	}
	return &result
}

func (m *ArrayMap) Merge(other *ArrayMap) *ArrayMap {
	if other.Count() == 0 {
		return m
	}
	if m.Count() == 0 {
		return other
	}
	res := m.Clone()
	for iter := other.iter(); iter.HasNext(); {
		p := iter.Next()
		res.Set(p.key, p.value)
	}
	return res
}

func (m *ArrayMap) Keys() Seq {
	mlen := len(m.arr) / 2
	res := make([]Object, mlen)
	for i := 0; i < mlen; i++ {
		res[i] = m.arr[i*2]
	}
	return &ArraySeq{arr: res}
}

func (m *ArrayMap) Vals() Seq {
	mlen := len(m.arr) / 2
	res := make([]Object, mlen)
	for i := 0; i < mlen; i++ {
		res[i] = m.arr[i*2+1]
	}
	return &ArraySeq{arr: res}
}

func (m *ArrayMap) iter() *ArrayMapIterator {
	return &ArrayMapIterator{m: m}
}

func (m *ArrayMap) Conj(obj Object) Conjable {
	switch obj := obj.(type) {
	case *Vector:
		if obj.count != 2 {
			panic(RT.NewError("Vector argument to map's conj must be a vector with two elements"))
		}
		return m.Assoc(obj.at(0), obj.at(1))
	case *ArrayMap:
		return m.Merge(obj)
	default:
		panic(RT.NewError("Argument to map's conj must be a vector with two elements or a map"))
	}
}

func EmptyArrayMap() *ArrayMap {
	return &ArrayMap{}
}

func (m *ArrayMap) ToString(escape bool) string {
	var b bytes.Buffer
	b.WriteRune('{')
	if len(m.arr) > 0 {
		for i := 0; i < len(m.arr)-2; i += 2 {
			b.WriteString(m.arr[i].ToString(escape))
			b.WriteRune(' ')
			b.WriteString(m.arr[i+1].ToString(escape))
			b.WriteString(", ")
		}
		b.WriteString(m.arr[len(m.arr)-2].ToString(escape))
		b.WriteRune(' ')
		b.WriteString(m.arr[len(m.arr)-1].ToString(escape))
	}
	b.WriteRune('}')
	return b.String()
}

func (m *ArrayMap) Equals(other interface{}) bool {
	if m == other {
		return true
	}
	switch otherMap := other.(type) {
	case *ArrayMap:
		if len(m.arr) != len(otherMap.arr) {
			return false
		}
		for i := 0; i < len(m.arr); i += 2 {
			success, value := otherMap.Get(m.arr[i])
			if !success || !value.Equals(m.arr[i+1]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (m *ArrayMap) GetType() *Type {
	return TYPES["ArrayMap"]
}

func (m *ArrayMap) Hash() uint32 {
	return hashUnordered(m.Seq())
}

func (m *ArrayMap) Seq() Seq {
	return &ArrayMapSeq{m: m, index: 0}
}

func (m *ArrayMap) Call(args []Object) Object {
	checkArity(args, 1, 2)
	if ok, v := m.Get(args[0]); ok {
		return v
	}
	if len(args) == 2 {
		return args[1]
	}
	return NIL
}

func SafeMerge(m1, m2 *ArrayMap) *ArrayMap {
	if m1 == nil {
		return m2
	}
	return m1.Merge(m2)
}
