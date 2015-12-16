package main

import (
	"bytes"
)

type (
	Seq interface {
		Object
		First() Object
		Rest() Seq
		IsEmpty() bool
		Cons(obj Object) Seq
	}
	Sequenceable interface {
		Seq() Seq
	}
	SeqIterator struct {
		seq Seq
	}
	ConsSeq struct {
		first Object
		rest  Seq
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

func (seq *ConsSeq) Equals(other interface{}) bool {
	if seq == other {
		return true
	}
	switch s := other.(type) {
	case Sequenceable:
		return SeqsEqual(seq, s.Seq())
	default:
		return false
	}
}

func (seq *ConsSeq) ToString(escape bool) string {
	return SeqToString(seq, escape)
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
