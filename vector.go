package main

import (
	"bytes"
)

type (
	Vector struct {
		root  []interface{}
		tail  []interface{}
		count int
		shift uint
	}
	VectorSeq struct {
		vector *Vector
		index  int
	}
	IndexError struct {
		index int
		count int
	}
)

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
	switch otherVector := other.(type) {
	case *Vector:
		if v == otherVector {
			return true
		}
		if v.count != otherVector.count {
			return false
		}
		for i := 0; i < v.count-1; i++ {
			if v.at(i) != otherVector.at(i) {
				return false
			}
		}
		return true
	default:
		return false
	}
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

func (vseq *VectorSeq) First() Object {
	return vseq.vector.at(vseq.index)
}

func (vseq *VectorSeq) Rest() Seq {
	if vseq.index+1 < vseq.vector.count {
		return &VectorSeq{vector: vseq.vector, index: vseq.index + 1}
	}
	return EmptyList
}

func (vseq *VectorSeq) IsEmpty() bool {
	return false
}

func (vseq *VectorSeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: vseq}
}

func (v *Vector) Seq() Seq {
	return &VectorSeq{vector: v, index: 0}
}

var EmptyVector = &Vector{
	count: 0,
	shift: 5,
	root:  make([]interface{}, 32),
	tail:  make([]interface{}, 0, 32),
}
