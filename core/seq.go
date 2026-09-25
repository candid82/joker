package core

import (
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
		step  int
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
	TransformSeq struct {
		InfoHolder
		MetaHolder
		kind   transformSeqKind
		fn     Callable
		source Seqable
		inner  Seq
		seq    Seq
		arg    [1]Object
		keep   bool
	}
	transformSeqKind uint8
)

const (
	transformMap transformSeqKind = iota + 1
	transformFilter
	transformMapcat
	transformConcat
)

func NewMapSeq(fn Callable, source Seqable) Seq {
	return &TransformSeq{kind: transformMap, fn: fn, source: source}
}

func NewFilterSeq(fn Callable, source Seqable, keep bool) Seq {
	return &TransformSeq{kind: transformFilter, fn: fn, source: source, keep: keep}
}

func NewMapcatSeq(fn Callable, source Seqable) Seq {
	return &TransformSeq{kind: transformMapcat, fn: fn, source: source}
}

func NewConcatSeq(sources []Object) Seq {
	arr := append([]Object(nil), sources...)
	return &TransformSeq{kind: transformConcat, source: &ArraySeq{arr: arr}}
}

func (seq *TransformSeq) call(obj Object) Object {
	seq.arg[0] = obj
	return seq.fn.Call(seq.arg[:])
}

func (seq *TransformSeq) finish(realized Seq) {
	seq.seq = realized
	seq.fn = nil
	seq.source = nil
	seq.inner = nil
	seq.arg[0] = nil
}

func (seq *TransformSeq) realize() {
	if seq.seq != nil {
		return
	}
	source := seq.source.Seq()
	switch seq.kind {
	case transformMap:
		if source.IsEmpty() {
			seq.finish(EmptyList)
			return
		}
		first := seq.call(source.First())
		rest := NewMapSeq(seq.fn, source.Rest())
		seq.finish(&ConsSeq{first: first, rest: rest})
	case transformFilter:
		for !source.IsEmpty() {
			first := source.First()
			rest := source.Rest()
			if ToBool(seq.call(first)) == seq.keep {
				restSeq := NewFilterSeq(seq.fn, rest, seq.keep)
				seq.finish(&ConsSeq{first: first, rest: restSeq})
				return
			}
			source = rest
		}
		seq.finish(EmptyList)
	case transformMapcat, transformConcat:
		inner := seq.inner
		for inner == nil || inner.IsEmpty() {
			if source.IsEmpty() {
				seq.finish(EmptyList)
				return
			}
			var next Object
			if seq.kind == transformMapcat {
				next = seq.call(source.First())
			} else {
				next = source.First()
			}
			inner = EnsureObjectIsSeqable(next, "").Seq()
			source = source.Rest()
		}
		first := inner.First()
		rest := &TransformSeq{
			kind:   seq.kind,
			fn:     seq.fn,
			source: source,
			inner:  inner.Rest(),
		}
		seq.finish(&ConsSeq{first: first, rest: rest})
	default:
		panic(RT.NewError("Unknown transforming sequence operation"))
	}
}

func (seq *TransformSeq) GetInfo() *ObjectInfo {
	return seq.info
}

func (seq *TransformSeq) WithInfo(info *ObjectInfo) Object {
	res := *seq
	res.info = info
	return &res
}

func (seq *TransformSeq) WithMeta(meta Map) Object {
	res := *seq
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (seq *TransformSeq) GetMeta() Map {
	return seq.meta
}

func (seq *TransformSeq) GetType() *Type {
	return TYPE.LazySeq
}

func (seq *TransformSeq) Seq() Seq {
	return seq
}

func (seq *TransformSeq) Equals(other interface{}) bool {
	return IsSeqEqual(seq, other)
}

func (seq *TransformSeq) ToString(escape bool) string {
	return SeqToString(seq, escape)
}

func (seq *TransformSeq) Pprint(w io.Writer, indent int) int {
	return pprintSeq(seq, w, indent)
}

func (seq *TransformSeq) Format(w io.Writer, indent int) int {
	return formatSeq(seq, w, indent)
}

func (seq *TransformSeq) Hash() uint32 {
	return hashOrdered(seq)
}

func (seq *TransformSeq) First() Object {
	seq.realize()
	return seq.seq.First()
}

func (seq *TransformSeq) Rest() Seq {
	seq.realize()
	return seq.seq.Rest()
}

func (seq *TransformSeq) IsEmpty() bool {
	seq.realize()
	return seq.seq.IsEmpty()
}

func (seq *TransformSeq) IsRealized() bool {
	return seq.seq != nil
}

func (seq *TransformSeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: seq}
}

func (seq *TransformSeq) reduce(c Callable) Object {
	return seqReduce(seq, c)
}

func (seq *TransformSeq) reduceInit(c Callable, init Object) Object {
	return seqReduceInit(seq, c, init)
}

func (seq *TransformSeq) sequential() {}

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

func (seq *MappingSeq) Format(w io.Writer, indent int) int {
	return formatSeq(seq, w, indent)
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

func (seq *MappingSeq) reduce(c Callable) Object {
	return seqReduce(seq, c)
}

func (seq *MappingSeq) reduceInit(c Callable, init Object) Object {
	return seqReduceInit(seq, c, init)
}

func (seq *MappingSeq) sequential() {}

func (seq *LazySeq) Seq() Seq {
	return seq
}

func (seq *LazySeq) realize() {
	if seq.seq == nil {
		seq.seq = EnsureObjectIsSeqable(seq.fn.Call([]Object{}), "").Seq()
		seq.fn = nil
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

func (seq *LazySeq) Format(w io.Writer, indent int) int {
	return formatSeq(seq, w, indent)
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

func (seq *LazySeq) reduce(c Callable) Object {
	seq.realize()
	return seqReduce(seq.seq, c)
}

func (seq *LazySeq) reduceInit(c Callable, init Object) Object {
	seq.realize()
	return seqReduceInit(seq.seq, c, init)
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

func (seq *ArraySeq) Format(w io.Writer, indent int) int {
	return formatSeq(seq, w, indent)
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

func (seq *ArraySeq) stride() int {
	if seq.step == 0 {
		return 1
	}
	return seq.step
}

func (seq *ArraySeq) Rest() Seq {
	next := seq.index + seq.stride()
	if next < len(seq.arr) {
		return &ArraySeq{index: next, step: seq.step, arr: seq.arr}
	}
	return EmptyList
}

func (seq *ArraySeq) IsEmpty() bool {
	return seq.index >= len(seq.arr)
}

func (seq *ArraySeq) Count() int {
	n := len(seq.arr) - seq.index
	if n <= 0 {
		return 0
	}
	step := seq.stride()
	return (n + step - 1) / step
}

func (seq *ArraySeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: seq}
}

func (seq *ArraySeq) reduce(c Callable) Object {
	if seq.IsEmpty() {
		return c.Call(nil)
	}
	step := seq.stride()
	res := seq.arr[seq.index]
	args := []Object{res, NIL}
	for i := seq.index + step; i < len(seq.arr); i += step {
		args[1] = seq.arr[i]
		res = c.Call(args)
		args[0] = res
	}
	return res
}

func (seq *ArraySeq) reduceInit(c Callable, init Object) Object {
	res := init
	args := []Object{res, NIL}
	step := seq.stride()
	for i := seq.index; i < len(seq.arr); i += step {
		args[1] = seq.arr[i]
		res = c.Call(args)
		args[0] = res
	}
	return res
}

func (seq *ArraySeq) sequential() {}

func SeqToString(seq Seq, escape bool) string {
	b := getBuffer()
	defer putBuffer(b)
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

func (seq *ConsSeq) Format(w io.Writer, indent int) int {
	return formatSeq(seq, w, indent)
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

func (seq *ConsSeq) reduce(c Callable) Object {
	return seqReduce(seq, c)
}

func (seq *ConsSeq) reduceInit(c Callable, init Object) Object {
	return seqReduceInit(seq, c, init)
}

func (seq *ConsSeq) sequential() {}

func NewConsSeq(first Object, rest Seq) *ConsSeq {
	return &ConsSeq{
		first: first,
		rest:  rest,
	}
}

func seqReduce(seq Seq, c Callable) Object {
	if seq.IsEmpty() {
		return c.Call(nil)
	}
	res := seq.First()
	return seqReduceInit(seq.Rest(), c, res)
}

func seqReduceInit(seq Seq, c Callable, init Object) Object {
	res := init
	args := []Object{res, NIL}
	for !seq.IsEmpty() {
		args[1] = seq.First()
		res = c.Call(args)
		args[0] = res
		seq = seq.Rest()
	}
	return res
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
	if c, ok := seq.(Counted); ok {
		return c.Count()
	}
	n := 0
	for !seq.IsEmpty() {
		if c, ok := seq.(Counted); ok {
			return n + c.Count()
		}
		n++
		seq = seq.Rest()
	}
	return n
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
