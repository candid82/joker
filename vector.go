package main

import (
	"bytes"
	"fmt"
)

type (
	Vector struct {
		InfoHolder
		MetaHolder
		root  []interface{}
		tail  []interface{}
		count int
		shift uint
	}
	VectorSeq struct {
		InfoHolder
		vector *Vector
		index  int
	}
	IndexError struct {
		index int
		count int
	}
)

func (err IndexError) Error() string {
	return fmt.Sprintf("Index %d is out of bounds [0..%d]", err.index, err.count)
}

func (v *Vector) WithMeta(meta *ArrayMap) Object {
	res := *v
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func clone(s []interface{}) []interface{} {
	result := make([]interface{}, len(s), cap(s))
	copy(result, s)
	return result
}

func (v *Vector) tailoff() int {
	if v.count < 32 {
		return 0
	}
	return ((v.count - 1) >> 5) << 5
}

func (v *Vector) arrayFor(i int) []interface{} {
	if i >= v.count || i < 0 {
		panic(IndexError{index: i, count: v.count})
	}
	if i >= v.tailoff() {
		return v.tail
	}
	node := v.root
	for level := v.shift; level > 0; level -= 5 {
		node = node[(i>>level)&0x01F].([]interface{})
	}
	return node
}

func (v *Vector) at(i int) Object {
	return v.arrayFor(i)[i&0x01F].(Object)
}

func newPath(level uint, node []interface{}) []interface{} {
	if level == 0 {
		return node
	}
	result := make([]interface{}, 32)
	result[0] = newPath(level-5, node)
	return result
}

func (v *Vector) pushTail(level uint, parent []interface{}, tailNode []interface{}) []interface{} {
	subidx := ((v.count - 1) >> level) & 0x01F
	result := clone(parent)
	var nodeToInsert []interface{}
	if level == 5 {
		nodeToInsert = tailNode
	} else {
		if parent[subidx] != nil {
			nodeToInsert = v.pushTail(level-5, parent[subidx].([]interface{}), tailNode)
		} else {
			nodeToInsert = newPath(level-5, tailNode)
		}
	}
	result[subidx] = nodeToInsert
	return result
}

func (v *Vector) conj(obj Object) *Vector {
	var newTail []interface{}
	if v.count-v.tailoff() < 32 {
		newTail = append(clone(v.tail), obj)
		return &Vector{count: v.count + 1, shift: v.shift, root: v.root, tail: newTail}
	}
	var newRoot []interface{}
	newShift := v.shift
	if (v.count >> 5) > (1 << v.shift) {
		newRoot = make([]interface{}, 32)
		newRoot[0] = v.root
		newRoot[1] = newPath(v.shift, v.tail)
		newShift += 5
	} else {
		newRoot = v.pushTail(v.shift, v.root, v.tail)
	}
	newTail = make([]interface{}, 1, 32)
	newTail[0] = obj
	return &Vector{count: v.count + 1, shift: newShift, root: newRoot, tail: newTail}
}

func (v *Vector) ToString(escape bool) string {
	var b bytes.Buffer
	b.WriteRune('[')
	if v.count > 0 {
		for i := 0; i < v.count-1; i++ {
			b.WriteString(v.at(i).ToString(escape))
			b.WriteRune(' ')
		}
		b.WriteString(v.at(v.count - 1).ToString(escape))
	}
	b.WriteRune(']')
	return b.String()
}

func (v *Vector) Equals(other interface{}) bool {
	if v == other {
		return true
	}
	switch other := other.(type) {
	case *Vector:
		if v.count != other.count {
			return false
		}
		for i := 0; i < v.count; i++ {
			if !v.at(i).Equals(other.at(i)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (v *Vector) WithInfo(info *ObjectInfo) Object {
	v.info = info
	return v
}

func (vseq *VectorSeq) Equals(other interface{}) bool {
	if vseq == other {
		return true
	}
	switch s := other.(type) {
	case Sequenceable:
		return SeqsEqual(vseq, s.Seq())
	default:
		return false
	}
}

func (vseq *VectorSeq) ToString(escape bool) string {
	return SeqToString(vseq, escape)
}

func (vseq *VectorSeq) WithInfo(info *ObjectInfo) Object {
	vseq.info = info
	return vseq
}

func (vseq *VectorSeq) First() Object {
	if vseq.index < vseq.vector.count {
		return vseq.vector.at(vseq.index)
	}
	return NIL
}

func (vseq *VectorSeq) Rest() Seq {
	if vseq.index+1 < vseq.vector.count {
		return &VectorSeq{vector: vseq.vector, index: vseq.index + 1}
	}
	return EmptyList
}

func (vseq *VectorSeq) IsEmpty() bool {
	return vseq.index >= vseq.vector.count
}

func (vseq *VectorSeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: vseq}
}

func (v *Vector) Seq() Seq {
	return &VectorSeq{vector: v, index: 0}
}

func (v *Vector) Conj(obj Object) Conjable {
	return v.conj(obj)
}

var EmptyVector = &Vector{
	count: 0,
	shift: 5,
	root:  make([]interface{}, 32),
	tail:  make([]interface{}, 0, 32),
}

func NewVectorFrom(objs ...Object) *Vector {
	res := EmptyVector
	for i := 0; i < len(objs); i++ {
		res = res.conj(objs[i])
	}
	return res
}
