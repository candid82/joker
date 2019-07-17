package core

import (
	"bytes"
	"fmt"
	"io"
)

type (
	Seq interface {
		Seqable
		Object
		First() Object
		Rest() Seq
		IsEmpty() bool
		Cons(obj Object) Seq
	}
	Seqable interface {
		Seq() Seq
	}
	SeqIterator struct {
		seq Seq
	}
	ConsSeq struct {
		InfoHolder
		MetaHolder
		first Object
		rest  Seq
	}
	ArraySeq struct {
		InfoHolder
		MetaHolder
		arr   []Object
		index int
	}
	LazySeq struct {
		InfoHolder
		MetaHolder
		fn  Callable
		seq Seq
	}
	MappingSeq struct {
		InfoHolder
		MetaHolder
		seq Seq
		fn  func(obj Object) Object
	}
)

func SeqsEqual(seq1, seq2 Seq) bool {
	iter2 := iter(seq2)
	for iter1 := iter(seq1); iter1.HasNext(); {
		if !iter2.HasNext() || !iter2.Next().Equals(iter1.Next()) {
			return false
		}
	}
	return !iter2.HasNext()
}

func IsSeqEqual(seq Seq, other interface{}) bool {
	if seq == other {
		return true
	}
	switch other := other.(type) {
	case Sequential:
		switch other := other.(type) {
		case Seqable:
			return SeqsEqual(seq, other.Seq())
		}
	}
	return false
}

func (seq *MappingSeq) Seq() Seq {
	return seq
}

func (seq *MappingSeq) Equals(other interface{}) bool {
	return IsSeqEqual(seq, other)
}

func (seq *MappingSeq) ToString(escape bool) string {
	return SeqToString(seq, escape)
}

func (seq *MappingSeq) Pprint(w io.Writer, indent int) int {
	return pprintSeq(seq, w, indent)
}

func (seq *MappingSeq) WithMeta(meta Map) Object {
	res := *seq
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (seq *MappingSeq) GetType() *Type {
	return TYPE.MappingSeq
}

func (seq *MappingSeq) Hash() uint32 {
	return hashOrdered(seq)
}

func (seq *MappingSeq) First() Object {
	return seq.fn(seq.seq.First())
}

func (seq *MappingSeq) Rest() Seq {
	return &MappingSeq{
		seq: seq.seq.Rest(),
		fn:  seq.fn,
	}
}

func (seq *MappingSeq) IsEmpty() bool {
	return seq.seq.IsEmpty()
}

func (seq *MappingSeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: seq}
}

func (seq *MappingSeq) sequential() {}

func (seq *LazySeq) Seq() Seq {
	return seq
}

func (seq *LazySeq) realize() {
	if seq.seq == nil {
		seq.seq = AssertSeqable(seq.fn.Call([]Object{}), "").Seq()
	}
}

func (seq *LazySeq) IsRealized() bool {
	return seq.seq != nil
}

func (seq *LazySeq) Equals(other interface{}) bool {
	return IsSeqEqual(seq, other)
}

func (seq *LazySeq) ToString(escape bool) string {
	return SeqToString(seq, escape)
}

func (seq *LazySeq) Pprint(w io.Writer, indent int) int {
	return pprintSeq(seq, w, indent)
}

func (seq *LazySeq) WithMeta(meta Map) Object {
	res := *seq
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (seq *LazySeq) GetType() *Type {
	return TYPE.LazySeq
}

func (seq *LazySeq) Hash() uint32 {
	return hashOrdered(seq)
}

func (seq *LazySeq) First() Object {
	seq.realize()
	return seq.seq.First()
}

func (seq *LazySeq) Rest() Seq {
	seq.realize()
	return seq.seq.Rest()
}

func (seq *LazySeq) IsEmpty() bool {
	seq.realize()
	return seq.seq.IsEmpty()
}

func (seq *LazySeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: seq}
}

func (seq *LazySeq) sequential() {}

func NewLazySeq(c Callable) *LazySeq {
	return &LazySeq{fn: c}
}

func (seq *ArraySeq) Seq() Seq {
	return seq
}

func (seq *ArraySeq) Equals(other interface{}) bool {
	return IsSeqEqual(seq, other)
}

func (seq *ArraySeq) ToString(escape bool) string {
	return SeqToString(seq, escape)
}

func (seq *ArraySeq) Pprint(w io.Writer, indent int) int {
	return pprintSeq(seq, w, indent)
}

func (seq *ArraySeq) WithMeta(meta Map) Object {
	res := *seq
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (seq *ArraySeq) GetType() *Type {
	return TYPE.ArraySeq
}

func (seq *ArraySeq) Hash() uint32 {
	return hashOrdered(seq)
}

func (seq *ArraySeq) First() Object {
	if seq.IsEmpty() {
		return NIL
	}
	return seq.arr[seq.index]
}

func (seq *ArraySeq) Rest() Seq {
	if seq.index+1 < len(seq.arr) {
		return &ArraySeq{index: seq.index + 1, arr: seq.arr}
	}
	return EmptyList
}

func (seq *ArraySeq) IsEmpty() bool {
	return seq.index >= len(seq.arr)
}

func (seq *ArraySeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: seq}
}

func (seq *ArraySeq) sequential() {}

func SeqToString(seq Seq, escape bool) string {
	var b bytes.Buffer
	b.WriteRune('(')
	for iter := iter(seq); iter.HasNext(); {
		b.WriteString(iter.Next().ToString(escape))
		if iter.HasNext() {
			b.WriteRune(' ')
		}
	}
	b.WriteRune(')')
	return b.String()
}

func (seq *ConsSeq) WithMeta(meta Map) Object {
	res := *seq
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (seq *ConsSeq) Seq() Seq {
	return seq
}

func (seq *ConsSeq) Equals(other interface{}) bool {
	return IsSeqEqual(seq, other)
}

func (seq *ConsSeq) ToString(escape bool) string {
	return SeqToString(seq, escape)
}

func (seq *ConsSeq) Pprint(w io.Writer, indent int) int {
	return pprintSeq(seq, w, indent)
}

func (seq *ConsSeq) GetType() *Type {
	return TYPE.ConsSeq
}

func (seq *ConsSeq) Hash() uint32 {
	return hashOrdered(seq)
}

func (seq *ConsSeq) First() Object {
	return seq.first
}

func (seq *ConsSeq) Rest() Seq {
	return seq.rest
}

func (seq *ConsSeq) IsEmpty() bool {
	return false
}

func (seq *ConsSeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: seq}
}

func (seq *ConsSeq) sequential() {}

func NewConsSeq(first Object, rest Seq) *ConsSeq {
	return &ConsSeq{
		first: first,
		rest:  rest,
	}
}

func iter(seq Seq) *SeqIterator {
	return &SeqIterator{seq: seq}
}

func (iter *SeqIterator) Next() Object {
	res := iter.seq.First()
	iter.seq = iter.seq.Rest()
	return res
}

func (iter *SeqIterator) HasNext() bool {
	return !iter.seq.IsEmpty()
}

func Second(seq Seq) Object {
	return seq.Rest().First()
}

func Third(seq Seq) Object {
	return seq.Rest().Rest().First()
}

func Fourth(seq Seq) Object {
	return seq.Rest().Rest().Rest().First()
}

func ToSlice(seq Seq) []Object {
	res := make([]Object, 0)
	for !seq.IsEmpty() {
		res = append(res, seq.First())
		seq = seq.Rest()
	}
	return res
}

func SeqCount(seq Seq) int {
	c := 0
	for !seq.IsEmpty() {
		switch obj := seq.(type) {
		case Counted:
			return c + obj.Count()
		}
		c++
		seq = seq.Rest()
	}
	return c
}

func SeqNth(seq Seq, n int) Object {
	if n < 0 {
		panic(RT.NewError(fmt.Sprintf("Negative index: %d", n)))
	}
	i := n
	for !seq.IsEmpty() {
		if i == 0 {
			return seq.First()
		}
		seq = seq.Rest()
		i--
	}
	panic(RT.NewError(fmt.Sprintf("Index %d exceeds seq's length %d", n, (n - i))))
}

func SeqTryNth(seq Seq, n int, d Object) Object {
	if n < 0 {
		return d
	}
	i := n
	for !seq.IsEmpty() {
		if i == 0 {
			return seq.First()
		}
		seq = seq.Rest()
		i--
	}
	return d
}

func hashUnordered(seq Seq, seed uint32) uint32 {
	for !seq.IsEmpty() {
		seed += seq.First().Hash()
		seq = seq.Rest()
	}
	h := getHash()
	h.Write(uint32ToBytes(seed))
	return h.Sum32()
}

func hashOrdered(seq Seq) uint32 {
	h := getHash()
	for !seq.IsEmpty() {
		h.Write(uint32ToBytes(seq.First().Hash()))
		seq = seq.Rest()
	}
	return h.Sum32()
}

func pprintSeq(seq Seq, w io.Writer, indent int) int {
	i := indent + 1
	fmt.Fprint(w, "(")
	for iter := iter(seq); iter.HasNext(); {
		i = pprintObject(iter.Next(), indent+1, w)
		if iter.HasNext() {
			fmt.Fprint(w, "\n")
			writeIndent(w, indent+1)
		}
	}
	fmt.Fprint(w, ")")
	return i + 1
}
