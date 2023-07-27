package core

import (
	"fmt"
	"io"
)

var (
	VECTOR_THRESHOLD int = 16
)

type (
	ArrayVector struct {
		InfoHolder
		MetaHolder
		arr []Object
	}
)

func (v *ArrayVector) WithMeta(meta Map) Object {
	res := *v
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (v *ArrayVector) Clone() *ArrayVector {
	res := ArrayVector{arr: make([]Object, len(v.arr), cap(v.arr))}
	res.meta = v.meta
	copy(res.arr, v.arr)
	return &res
}

func (v *ArrayVector) Conjoin(obj Object) Vec {
	if v.Count() >= VECTOR_THRESHOLD {
		res := NewVectorFrom(v.arr...)
		res.meta = v.meta
		return res
	}
	res := v.Clone()
	res.arr = append(res.arr, obj)
	return res
}

func (v *ArrayVector) Append(obj Object) {
	v.arr = append(v.arr, obj)
}

func (v *ArrayVector) At(i int) Object {
	return v.arr[i]
}

func (v *ArrayVector) ToString(escape bool) string {
	return CountedIndexedToString(v, escape)
}

func (v *ArrayVector) Equals(other interface{}) bool {
	if v == other {
		return true
	}
	switch other := other.(type) {
	case CountedIndexed:
		return AreCountedIndexedEqual(v, other)
	default:
		return IsSeqEqual(v.Seq(), other)
	}
}

func (v *ArrayVector) GetType() *Type {
	return TYPE.ArrayVector
}

func (v *ArrayVector) Hash() uint32 {
	return CountedIndexedHash(v)
}

func (v *ArrayVector) Seq() Seq {
	return &VectorSeq{vector: v, index: 0}
}

func (v *ArrayVector) Conj(obj Object) Conjable {
	return v.Conjoin(obj)
}

func (v *ArrayVector) Count() int {
	return len(v.arr)
}

func (v *ArrayVector) Nth(i int) Object {
	if i < 0 || i >= v.Count() {
		panic(RT.NewError(fmt.Sprintf("Index %d is out of bounds [0..%d]", i, v.Count()-1)))
	}
	return v.At(i)
}

func (v *ArrayVector) TryNth(i int, d Object) Object {
	if i < 0 || i >= v.Count() {
		return d
	}
	return v.At(i)
}

func (v *ArrayVector) sequential() {}

func (v *ArrayVector) Compare(other Object) int {
	v2 := EnsureObjectIsCountedIndexed(other, "Cannot compare Vector: %s")
	return CountedIndexedCompare(v, v2)
}

func (v *ArrayVector) Peek() Object {
	if v.Count() > 0 {
		return v.At(v.Count() - 1)
	}
	return NIL
}

func (v *ArrayVector) Pop() Stack {
	if v.Count() == 0 {
		panic(RT.NewError("Can't pop empty vector"))
	}
	res := v.Clone()
	res.arr = res.arr[:len(res.arr)-1]
	return res
}

func (v *ArrayVector) Get(key Object) (bool, Object) {
	return CountedIndexedGet(v, key)
}

func (v *ArrayVector) EntryAt(key Object) *Vector {
	ok, val := v.Get(key)
	if ok {
		return NewVectorFrom(key, val)
	}
	return nil
}

func (v *ArrayVector) Assoc(key, val Object) Associative {
	i := assertInteger(key)
	if i < 0 || i > v.Count() {
		panic(RT.NewError((fmt.Sprintf("Index %d is out of bounds [0..%d]", i, v.Count()))))
	}
	if i == v.Count() {
		return v.Conjoin(val)
	}
	res := v.Clone()
	res.arr[i] = val
	return res
}

func (v *ArrayVector) Rseq() Seq {
	return &VectorRSeq{vector: v, index: v.Count() - 1}
}

func (v *ArrayVector) Call(args []Object) Object {
	CheckArity(args, 1, 1)
	i := assertInteger(args[0])
	return v.Nth(i)
}

func EmptyArrayVector() *ArrayVector {
	return &ArrayVector{}
}

func (v *ArrayVector) Empty() Collection {
	return EmptyArrayVector()
}

func (v *ArrayVector) kvreduce(c Callable, init Object) Object {
	return CountedIndexedKvreduce(v, c, init)
}

func (v *ArrayVector) Pprint(w io.Writer, indent int) int {
	return CountedIndexedPprint(v, w, indent)
}

func (v *ArrayVector) Format(w io.Writer, indent int) int {
	return CountedIndexedFormat(v, w, indent)
}
