//go:generate go-bindata -pkg core -o bindata.go data
//go:generate go run gen/gen_types.go assert Comparable *Vector Char String Symbol Keyword Regex Bool Number Seqable Callable *Type Meta Int Stack Map Set Associative Reversible Named Comparator *Ratio *Namespace *Var
//go:generate go run gen/gen_types.go info *List *ArrayMapSeq *ArrayMap *HashMap *ExInfo *Fn *Var Nil *Ratio *BigInt *BigFloat Char Double Int Bool Keyword Regex Symbol String *LazySeq *MappingSeq *ArraySeq *ConsSeq *NodeSeq *ArrayNodeSeq *ArraySet *Vector *VectorSeq *VectorRSeq

package core

import (
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"math/big"
	"reflect"
	"strings"
	"unsafe"
)

type (
	Position struct {
		line   int
		column int
	}
	Equality interface {
		Equals(interface{}) bool
	}
	Type struct {
		name        string
		reflectType reflect.Type
	}
	Object interface {
		Equality
		ToString(escape bool) string
		GetInfo() *ObjectInfo
		WithInfo(*ObjectInfo) Object
		GetType() *Type
		Hash() uint32
	}
	Conjable interface {
		Object
		Conj(obj Object) Conjable
	}
	Counted interface {
		Count() int
	}
	Error interface {
		error
		Object
	}
	Meta interface {
		GetMeta() Map
		WithMeta(Map) Object
	}
	MetaHolder struct {
		meta Map
	}
	ObjectInfo struct {
		Position
	}
	InfoHolder struct {
		info *ObjectInfo
	}
	Char struct {
		InfoHolder
		ch rune
	}
	Double struct {
		InfoHolder
		D float64
	}
	Int struct {
		InfoHolder
		I int
	}
	BigInt struct {
		InfoHolder
		b big.Int
	}
	BigFloat struct {
		InfoHolder
		b big.Float
	}
	Ratio struct {
		InfoHolder
		r big.Rat
	}
	Bool struct {
		InfoHolder
		B bool
	}
	Nil struct {
		InfoHolder
		n struct{}
	}
	Keyword struct {
		InfoHolder
		ns   *string
		name *string
		hash uint32
	}
	Symbol struct {
		InfoHolder
		MetaHolder
		ns   *string
		name *string
		hash uint32
	}
	String struct {
		InfoHolder
		S string
	}
	Regex struct {
		InfoHolder
		R string
	}
	Var struct {
		InfoHolder
		MetaHolder
		ns      *Namespace
		name    Symbol
		Value   Object
		expr    Expr
		isMacro bool
	}
	Proc func([]Object) Object
	Fn   struct {
		InfoHolder
		MetaHolder
		fnExpr *FnExpr
		env    *LocalEnv
	}
	ExInfo struct {
		InfoHolder
		msg  String
		data *ArrayMap
		rt   *Runtime
	}
	RecurBindings []Object
	Delay         struct {
		fn    Callable
		value Object
	}
	Sequential interface {
		sequential()
	}
	Comparable interface {
		Compare(other Object) int
	}
	Indexed interface {
		Nth(i int) Object
		TryNth(i int, d Object) Object
	}
	Stack interface {
		Peek() Object
		Pop() Stack
	}
	Gettable interface {
		Get(key Object) (bool, Object)
	}
	Associative interface {
		Conjable
		Gettable
		EntryAt(key Object) *Vector
		Assoc(key, val Object) Associative
	}
	Reversible interface {
		Rseq() Seq
	}
	Named interface {
		Name() string
		Namespace() string
	}
	Comparator interface {
		Compare(a, b Object) int
	}
	SortableSlice struct {
		s   []Object
		cmp Comparator
	}
)

var TYPES = map[string]*Type{}

func init() {
	TYPES["String"] = &Type{name: "String", reflectType: reflect.TypeOf((*String)(nil)).Elem()}
	TYPES["Int"] = &Type{name: "Int", reflectType: reflect.TypeOf((*Int)(nil)).Elem()}
	TYPES["Type"] = &Type{name: "Type", reflectType: reflect.TypeOf((*Type)(nil))}
	TYPES["Delay"] = &Type{name: "Delay", reflectType: reflect.TypeOf((*Delay)(nil))}
	TYPES["Char"] = &Type{name: "Char", reflectType: reflect.TypeOf((*Char)(nil)).Elem()}
	TYPES["Double"] = &Type{name: "Double", reflectType: reflect.TypeOf((*Double)(nil)).Elem()}
	TYPES["BigInt"] = &Type{name: "BigInt", reflectType: reflect.TypeOf((*BigInt)(nil))}
	TYPES["BigFloat"] = &Type{name: "BigFloat", reflectType: reflect.TypeOf((*BigFloat)(nil))}
	TYPES["Ratio"] = &Type{name: "Ratio", reflectType: reflect.TypeOf((*Ratio)(nil))}
	TYPES["Bool"] = &Type{name: "Bool", reflectType: reflect.TypeOf((*Bool)(nil)).Elem()}
	TYPES["Nil"] = &Type{name: "Nil", reflectType: reflect.TypeOf((*Nil)(nil)).Elem()}
	TYPES["Keyword"] = &Type{name: "Keyword", reflectType: reflect.TypeOf((*Keyword)(nil)).Elem()}
	TYPES["Symbol"] = &Type{name: "Symbol", reflectType: reflect.TypeOf((*Symbol)(nil)).Elem()}
	TYPES["Regex"] = &Type{name: "Regex", reflectType: reflect.TypeOf((*Regex)(nil)).Elem()}
	TYPES["Var"] = &Type{name: "Var", reflectType: reflect.TypeOf((*Var)(nil))}
	TYPES["Proc"] = &Type{name: "Proc", reflectType: reflect.TypeOf((*Proc)(nil)).Elem()}
	TYPES["Fn"] = &Type{name: "Fn", reflectType: reflect.TypeOf((*Fn)(nil))}
	TYPES["ExInfo"] = &Type{name: "ExInfo", reflectType: reflect.TypeOf((*ExInfo)(nil))}
	TYPES["RecurBindings"] = &Type{name: "RecurBindings", reflectType: reflect.TypeOf((*RecurBindings)(nil)).Elem()}
	TYPES["Vector"] = &Type{name: "Vector", reflectType: reflect.TypeOf((*Vector)(nil))}
	TYPES["ArrayMap"] = &Type{name: "ArrayMap", reflectType: reflect.TypeOf((*ArrayMap)(nil))}
	TYPES["HashMap"] = &Type{name: "HashMap", reflectType: reflect.TypeOf((*HashMap)(nil))}
	TYPES["ArraySet"] = &Type{name: "ArraySet", reflectType: reflect.TypeOf((*ArraySet)(nil))}
	TYPES["List"] = &Type{name: "List", reflectType: reflect.TypeOf((*List)(nil))}
	TYPES["ArrayMapSeq"] = &Type{name: "ArrayMapSeq", reflectType: reflect.TypeOf((*ArrayMapSeq)(nil))}
	TYPES["ArraySeq"] = &Type{name: "ArraySeq", reflectType: reflect.TypeOf((*ArraySeq)(nil))}
	TYPES["ArrayNodeSeq"] = &Type{name: "ArrayNodeSeq", reflectType: reflect.TypeOf((*ArrayNodeSeq)(nil))}
	TYPES["NodeSeq"] = &Type{name: "NodeSeq", reflectType: reflect.TypeOf((*NodeSeq)(nil))}
	TYPES["ConsSeq"] = &Type{name: "ConsSeq", reflectType: reflect.TypeOf((*ConsSeq)(nil))}
	TYPES["NodeSeq"] = &Type{name: "NodeSeq", reflectType: reflect.TypeOf((*NodeSeq)(nil))}
	TYPES["LazySeq"] = &Type{name: "LazySeq", reflectType: reflect.TypeOf((*LazySeq)(nil))}
	TYPES["MappingSeq"] = &Type{name: "MappingSeq", reflectType: reflect.TypeOf((*MappingSeq)(nil))}
	TYPES["VectorSeq"] = &Type{name: "VectorSeq", reflectType: reflect.TypeOf((*VectorSeq)(nil))}
	TYPES["VectorRSeq"] = &Type{name: "VectorRSeq", reflectType: reflect.TypeOf((*VectorRSeq)(nil))}
	TYPES["Seq"] = &Type{name: "Seq", reflectType: reflect.TypeOf((*Seq)(nil)).Elem()}
	TYPES["Seqable"] = &Type{name: "Seqable", reflectType: reflect.TypeOf((*Seqable)(nil)).Elem()}
	TYPES["Number"] = &Type{name: "Number", reflectType: reflect.TypeOf((*Number)(nil)).Elem()}
	TYPES["Sequential"] = &Type{name: "Sequential", reflectType: reflect.TypeOf((*Sequential)(nil)).Elem()}
	TYPES["Comparable"] = &Type{name: "Comparable", reflectType: reflect.TypeOf((*Comparable)(nil)).Elem()}
	TYPES["Indexed"] = &Type{name: "Indexed", reflectType: reflect.TypeOf((*Indexed)(nil)).Elem()}
	TYPES["Stack"] = &Type{name: "Stack", reflectType: reflect.TypeOf((*Stack)(nil)).Elem()}
	TYPES["Map"] = &Type{name: "Map", reflectType: reflect.TypeOf((*Map)(nil)).Elem()}
	TYPES["Set"] = &Type{name: "Set", reflectType: reflect.TypeOf((*Set)(nil)).Elem()}
	TYPES["Named"] = &Type{name: "Named", reflectType: reflect.TypeOf((*Named)(nil)).Elem()}
	TYPES["Namespace"] = &Type{name: "Namespace", reflectType: reflect.TypeOf((*Namespace)(nil)).Elem()}
	TYPES["Comparator"] = &Type{name: "Comparator", reflectType: reflect.TypeOf((*Comparator)(nil)).Elem()}
	TYPES["EvalError"] = &Type{name: "EvalError", reflectType: reflect.TypeOf((*EvalError)(nil)).Elem()}
}

var hasher hash.Hash32 = fnv.New32a()

func newIteratorError() error {
	return errors.New("Iterator reached the end of collection")
}

func uint32ToBytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, i)
	return b
}

func getHash() hash.Hash32 {
	hasher.Reset()
	return hasher
}

func hashSymbol(ns, name *string) uint32 {
	h := getHash()
	b := make([]byte, 16)
	binary.LittleEndian.PutUint64(b, uint64(uintptr(unsafe.Pointer(name))))
	binary.LittleEndian.PutUint64(b[8:], uint64(uintptr(unsafe.Pointer(ns))))
	h.Write(b)
	return h.Sum32()
}

func MakeSymbol(nsname string) Symbol {
	index := strings.IndexRune(nsname, '/')
	if index == -1 || nsname == "/" {
		return Symbol{
			ns:   nil,
			name: STRINGS.Intern(nsname),
		}
	}
	return Symbol{
		ns:   STRINGS.Intern(nsname[0:index]),
		name: STRINGS.Intern(nsname[index+1 : len(nsname)]),
	}
}

func MakeKeyword(nsname string) Keyword {
	index := strings.IndexRune(nsname, '/')
	if index == -1 || nsname == "/" {
		name := STRINGS.Intern(nsname)
		return Keyword{
			ns:   nil,
			name: name,
			hash: hashSymbol(nil, name),
		}
	}
	ns := STRINGS.Intern(nsname[0:index])
	name := STRINGS.Intern(nsname[index+1 : len(nsname)])
	return Keyword{
		ns:   ns,
		name: name,
		hash: hashSymbol(ns, name),
	}
}

func panicArity(n int) {
	name := RT.currentExpr.(Traceable).Name()
	panic(RT.NewError(fmt.Sprintf("Wrong number of args (%d) passed to %s", n, name)))
}

func checkArity(args []Object, min int, max int) {
	n := len(args)
	if n < min || n > max {
		panicArity(n)
	}
}

func (s SortableSlice) Len() int {
	return len(s.s)
}

func (s SortableSlice) Swap(i, j int) {
	s.s[i], s.s[j] = s.s[j], s.s[i]
}

func (s SortableSlice) Less(i, j int) bool {
	return s.cmp.Compare(s.s[i], s.s[j]) == -1
}

func hashPtr(ptr uintptr) uint32 {
	h := getHash()
	b := make([]byte, unsafe.Sizeof(ptr))
	b[0] = byte(ptr)
	b[1] = byte(ptr >> 8)
	b[2] = byte(ptr >> 16)
	b[3] = byte(ptr >> 24)
	if unsafe.Sizeof(ptr) == 8 {
		b[4] = byte(ptr >> 32)
		b[5] = byte(ptr >> 40)
		b[6] = byte(ptr >> 48)
		b[7] = byte(ptr >> 56)
	}
	h.Write(b)
	return h.Sum32()
}

func hashGobEncoder(e gob.GobEncoder) uint32 {
	h := getHash()
	b, err := e.GobEncode()
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	h.Write(b)
	return h.Sum32()
}

func (d *Delay) ToString(escape bool) string {
	return "#object[Delay]"
}

func (d *Delay) Equals(other interface{}) bool {
	return d == other
}

func (d *Delay) GetInfo() *ObjectInfo {
	return nil
}

func (d *Delay) GetType() *Type {
	return TYPES["Delay"]
}

func (d *Delay) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(d)))
}

func (d *Delay) WithInfo(info *ObjectInfo) Object {
	return d
}

func (d *Delay) Force() Object {
	if d.value == nil {
		d.value = d.fn.Call([]Object{})
	}
	return d.value
}

func (t *Type) ToString(escape bool) string {
	return t.name
}

func (t *Type) Equals(other interface{}) bool {
	return t == other
}

func (t *Type) GetInfo() *ObjectInfo {
	return nil
}

func (t *Type) GetType() *Type {
	return TYPES["Type"]
}

func (t *Type) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(t)))
}

func (rb RecurBindings) ToString(escape bool) string {
	return "#object[RecurBindings]"
}

func (rb RecurBindings) Equals(other interface{}) bool {
	return false
}

func (rb RecurBindings) GetInfo() *ObjectInfo {
	return nil
}

func (rb RecurBindings) GetType() *Type {
	return TYPES["RecurBindings"]
}

func (rb RecurBindings) Hash() uint32 {
	return 0
}

func (exInfo *ExInfo) ToString(escape bool) string {
	return exInfo.msg.ToString(escape)
}

func (exInfo *ExInfo) Type() Symbol {
	return MakeSymbol("ExInfo")
}

func (exInfo *ExInfo) Equals(other interface{}) bool {
	return exInfo == other
}

func (exInfo *ExInfo) GetType() *Type {
	return TYPES["ExInfo"]
}

func (exInfo *ExInfo) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(exInfo)))
}

func (exInfo *ExInfo) Error() string {
	var pos Position
	ok, form := exInfo.data.Get(MakeKeyword("form"))
	if ok {
		if form.GetInfo() != nil {
			pos = form.GetInfo().Pos()
		}
	}
	if len(exInfo.rt.callstack.frames) > 0 {
		return fmt.Sprintf("stdin:%d:%d: Exception: %s\nStacktrace:\n%s", pos.line, pos.column, exInfo.msg.S, exInfo.rt.stacktrace())
	} else {
		return fmt.Sprintf("stdin:%d:%d: Exception: %s", pos.line, pos.column, exInfo.msg.S)
	}
}

func (fn *Fn) ToString(escape bool) string {
	return "#object[Fn]"
}

func (fn *Fn) Equals(other interface{}) bool {
	switch other := other.(type) {
	case *Fn:
		return fn == other
	default:
		return false
	}
}

func (fn *Fn) WithMeta(meta Map) Object {
	res := *fn
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (fn *Fn) GetType() *Type {
	return TYPES["Fn"]
}

func (fn *Fn) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(fn)))
}

func (fn *Fn) Call(args []Object) Object {
	for _, arity := range fn.fnExpr.arities {
		if len(arity.args) == len(args) {
			RT.pushFrame()
			defer RT.popFrame()
			return evalLoop(arity.body, fn.env.addFrame(args))
		}
	}
	v := fn.fnExpr.variadic
	if v == nil || len(args) < len(v.args)-1 {
		panicArity(len(args))
	}
	var restArgs Object = NIL
	if len(v.args)-1 < len(args) {
		restArgs = &ArraySeq{arr: args, index: len(v.args) - 1}
	}
	vargs := make([]Object, len(v.args))
	for i := 0; i < len(vargs)-1; i++ {
		vargs[i] = args[i]
	}
	vargs[len(vargs)-1] = restArgs
	RT.pushFrame()
	defer RT.popFrame()
	return evalLoop(v.body, fn.env.addFrame(vargs))
}

func compare(c Callable, a, b Object) int {
	switch r := c.Call([]Object{a, b}).(type) {
	case Bool:
		if r.B {
			return -1
		}
		if AssertBool(c.Call([]Object{b, a}), "").B {
			return 1
		}
		return 0
	default:
		return AssertNumber(r, "Function is not a comparator since it returned a non-integer value").Int().I
	}
}

func (fn *Fn) Compare(a, b Object) int {
	return compare(fn, a, b)
}

func (p Proc) Call(args []Object) Object {
	return p(args)
}

func (p Proc) Compare(a, b Object) int {
	return compare(p, a, b)
}

func (p Proc) ToString(escape bool) string {
	return "#object[Proc]"
}

func (p Proc) Equals(other interface{}) bool {
	return reflect.ValueOf(p).Pointer() == reflect.ValueOf(other).Pointer()
}

func (p Proc) GetInfo() *ObjectInfo {
	return nil
}

func (p Proc) WithInfo(*ObjectInfo) Object {
	return p
}

func (p Proc) GetType() *Type {
	return TYPES["Proc"]
}

func (p Proc) Hash() uint32 {
	return hashPtr(reflect.ValueOf(p).Pointer())
}

func (i InfoHolder) GetInfo() *ObjectInfo {
	return i.info
}

func (m MetaHolder) GetMeta() Map {
	return m.meta
}

func (sym Symbol) WithMeta(meta Map) Object {
	res := sym
	res.meta = SafeMerge(res.meta, meta)
	return res
}

func (v *Var) ToString(escape bool) string {
	return "#'" + v.ns.Name.ToString(false) + "/" + v.name.ToString(false)
}

func (v *Var) Equals(other interface{}) bool {
	// TODO: revisit this
	return v == other
}

func (v *Var) WithMeta(meta Map) Object {
	res := *v
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (v *Var) GetType() *Type {
	return TYPES["Var"]
}

func (v *Var) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(v)))
}

func (v *Var) Call(args []Object) Object {
	return AssertCallable(
		v.Value,
		"Var "+v.ToString(false)+" resolves to "+v.Value.ToString(false)+", which is not a Fn").Call(args)
}

func (n Nil) ToString(escape bool) string {
	return "nil"
}

func (n Nil) Equals(other interface{}) bool {
	switch other.(type) {
	case Nil:
		return true
	default:
		return false
	}
}

func (n Nil) GetType() *Type {
	return TYPES["Nil"]
}

func (n Nil) Hash() uint32 {
	return 0
}

func (n Nil) Seq() Seq {
	return n
}

func (n Nil) First() Object {
	return NIL
}

func (n Nil) Rest() Seq {
	return NIL
}

func (n Nil) IsEmpty() bool {
	return true
}

func (n Nil) Cons(obj Object) Seq {
	return NewListFrom(obj)
}

func (n Nil) Conj(obj Object) Conjable {
	return NewListFrom(obj)
}

func (n Nil) Without(key Object) Map {
	return n
}

func (n Nil) Count() int {
	return 0
}

func (n Nil) Iter() MapIterator {
	return emptyMapIterator
}

func (n Nil) Merge(other Map) Map {
	return other
}

func (n Nil) Assoc(key, value Object) Associative {
	return EmptyArrayMap().Assoc(key, value)
}

func (n Nil) EntryAt(key Object) *Vector {
	return nil
}

func (n Nil) Get(key Object) (bool, Object) {
	return false, NIL
}

func (n Nil) Disjoin(key Object) Set {
	return n
}

func (n Nil) Keys() Seq {
	return NIL
}

func (n Nil) Vals() Seq {
	return NIL
}

func (rat *Ratio) ToString(escape bool) string {
	return rat.r.String()
}

func (rat *Ratio) Equals(other interface{}) bool {
	if rat == other {
		return true
	}
	switch r := other.(type) {
	case *Ratio:
		return rat.r.Cmp(&r.r) == 0
	case *BigInt:
		var otherRat big.Rat
		otherRat.SetInt(&r.b)
		return rat.r.Cmp(&otherRat) == 0
	case Int:
		var otherRat big.Rat
		otherRat.SetInt64(int64(r.I))
		return rat.r.Cmp(&otherRat) == 0
	}
	return false
}

func (rat *Ratio) GetType() *Type {
	return TYPES["Ratio"]
}

func (rat *Ratio) Hash() uint32 {
	return hashGobEncoder(&rat.r)
}

func (rat *Ratio) Compare(other Object) int {
	return CompareNumbers(rat, AssertNumber(other, "Cannot compare Ratio and "+other.GetType().ToString(false)))
}

func (bi *BigInt) ToString(escape bool) string {
	return bi.b.String() + "N"
}

func (bi *BigInt) Equals(other interface{}) bool {
	if bi == other {
		return true
	}
	switch b := other.(type) {
	case *BigInt:
		return bi.b.Cmp(&b.b) == 0
	case Int:
		bi2 := big.NewInt(int64(b.I))
		return bi.b.Cmp(bi2) == 0
	}
	return false
}

func (bi *BigInt) GetType() *Type {
	return TYPES["BigInt"]
}

func (bi *BigInt) Hash() uint32 {
	return hashGobEncoder(&bi.b)
}

func (bi *BigInt) Compare(other Object) int {
	return CompareNumbers(bi, AssertNumber(other, "Cannot compare BigInt and "+other.GetType().ToString(false)))
}

func (bf *BigFloat) ToString(escape bool) string {
	return bf.b.Text('g', 256) + "M"
}

func (bf *BigFloat) Equals(other interface{}) bool {
	if bf == other {
		return true
	}
	switch b := other.(type) {
	case *BigFloat:
		return bf.b.Cmp(&b.b) == 0
	case Double:
		bf2 := big.NewFloat(b.D)
		return bf.b.Cmp(bf2) == 0
	}
	return false
}

func (bf *BigFloat) GetType() *Type {
	return TYPES["BigFloat"]
}

func (bf *BigFloat) Hash() uint32 {
	return hashGobEncoder(&bf.b)
}

func (bf *BigFloat) Compare(other Object) int {
	return CompareNumbers(bf, AssertNumber(other, "Cannot compare BigFloat and "+other.GetType().ToString(false)))
}

func (c Char) ToString(escape bool) string {
	if escape {
		return escapeRune(c.ch)
	}
	return string(c.ch)
}

func (c Char) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Char:
		return c.ch == other.ch
	default:
		return false
	}
}

func (c Char) GetType() *Type {
	return TYPES["Char"]
}

func (c Char) Hash() uint32 {
	h := getHash()
	h.Write([]byte(string(c.ch)))
	return h.Sum32()
}

func (c Char) Compare(other Object) int {
	c2 := AssertChar(other, "Cannot compare Char and "+other.GetType().ToString(false))
	if c.ch < c2.ch {
		return -1
	}
	if c2.ch < c.ch {
		return 1
	}
	return 0
}

func MakeDouble(d float64) Double {
	return Double{D: d}
}

func (d Double) ToString(escape bool) string {
	return fmt.Sprintf("%f", d.D)
}

func (d Double) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Double:
		return d.D == other.D
	default:
		return false
	}
}

func (d Double) GetType() *Type {
	return TYPES["Double"]
}

func (d Double) Hash() uint32 {
	h := getHash()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, math.Float64bits(d.D))
	h.Write(b)
	return h.Sum32()
}

func (d Double) Compare(other Object) int {
	return CompareNumbers(d, AssertNumber(other, "Cannot compare Double and "+other.GetType().ToString(false)))
}

func (i Int) ToString(escape bool) string {
	return fmt.Sprintf("%d", i.I)
}

func (i Int) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Int:
		return i.I == other.I
	default:
		return false
	}
}

func (i Int) GetType() *Type {
	return TYPES["Int"]
}

func (i Int) Hash() uint32 {
	h := getHash()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i.I))
	h.Write(b)
	return h.Sum32()
}

func (i Int) Compare(other Object) int {
	return CompareNumbers(i, AssertNumber(other, "Cannot compare Int and "+other.GetType().ToString(false)))
}

func (b Bool) ToString(escape bool) string {
	return fmt.Sprintf("%t", b.B)
}

func (b Bool) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Bool:
		return b.B == other.B
	default:
		return false
	}
}

func (b Bool) GetType() *Type {
	return TYPES["Bool"]
}

func (b Bool) Hash() uint32 {
	h := getHash()
	var bs = make([]byte, 1)
	if b.B {
		bs[0] = 1
	} else {
		bs[0] = 0
	}
	h.Write(bs)
	return h.Sum32()
}

func (b Bool) Compare(other Object) int {
	b2 := AssertBool(other, "Cannot compare Bool and "+other.GetType().ToString(false))
	if b.B == b2.B {
		return 0
	}
	if b.B {
		return 1
	}
	return -1
}

func (k Keyword) ToString(escape bool) string {
	return ":" + k.Name()
}

func (k Keyword) Name() string {
	if k.ns != nil {
		return *k.ns + "/" + *k.name
	}
	return *k.name
}

func (k Keyword) Namespace() string {
	if k.ns != nil {
		return *k.ns
	}
	return ""
}

func (k Keyword) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Keyword:
		return k.ns == other.ns && k.name == other.name
	default:
		return false
	}
}

func (k Keyword) GetType() *Type {
	return TYPES["Keyword"]
}

func (k Keyword) Hash() uint32 {
	return k.hash
}

func (k Keyword) Compare(other Object) int {
	k2 := AssertKeyword(other, "Cannot compare Keyword and "+other.GetType().ToString(false))
	return strings.Compare(k.ToString(false), k2.ToString(false))
}

func (k Keyword) Call(args []Object) Object {
	checkArity(args, 1, 2)
	switch m := args[0].(type) {
	case *ArrayMap:
		ok, v := m.Get(k)
		if ok {
			return v
		}
	}
	if len(args) == 2 {
		return args[1]
	}
	return NIL
}

func (rx Regex) ToString(escape bool) string {
	if escape {
		return "#" + escapeString(rx.R)
	}
	return "#" + rx.R
}

func (rx Regex) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Regex:
		return rx.R == other.R
	default:
		return false
	}
}

func (rx Regex) GetType() *Type {
	return TYPES["Regex"]
}

func (rx Regex) Hash() uint32 {
	h := getHash()
	h.Write([]byte(rx.R))
	return h.Sum32()
}

func (s Symbol) ToString(escape bool) string {
	return s.Name()
}

func (s Symbol) Name() string {
	if s.ns != nil {
		return *s.ns + "/" + *s.name
	}
	return *s.name
}

func (s Symbol) Namespace() string {
	if s.ns != nil {
		return *s.ns
	}
	return ""
}

func (s Symbol) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Symbol:
		return s.ns == other.ns && s.name == other.name
	default:
		return false
	}
}

func (s Symbol) GetType() *Type {
	return TYPES["Symbol"]
}

func (s Symbol) Hash() uint32 {
	return hashSymbol(s.ns, s.name) + 0x9e3779b9
}

func (s Symbol) Compare(other Object) int {
	s2 := AssertSymbol(other, "Cannot compare Symbol and "+other.GetType().ToString(false))
	return strings.Compare(s.ToString(false), s2.ToString(false))
}

func (s String) ToString(escape bool) string {
	if escape {
		return escapeString(s.S)
	}
	return s.S
}

func MakeString(s string) String {
	return String{S: s}
}

func (s String) Equals(other interface{}) bool {
	switch other := other.(type) {
	case String:
		return s.S == other.S
	default:
		return false
	}
}

func (s String) GetType() *Type {
	return TYPES["String"]
}

func (s String) Hash() uint32 {
	h := getHash()
	h.Write([]byte(s.S))
	return h.Sum32()
}

func (s String) Count() int {
	return len(s.S)
}

func (s String) Seq() Seq {
	runes := make([]Object, 0, len(s.S))
	for _, r := range s.S {
		runes = append(runes, Char{ch: r})
	}
	return &ArraySeq{arr: runes}
}

func (s String) Nth(i int) Object {
	if i < 0 {
		panic(RT.NewError(fmt.Sprintf("Negative index: %d", i)))
	}
	j, r := 0, 't'
	for j, r = range s.S {
		if i == j {
			return Char{ch: r}
		}
	}
	panic(RT.NewError(fmt.Sprintf("Index %d exceeds string's length %d", i, j+1)))
}

func (s String) TryNth(i int, d Object) Object {
	if i < 0 {
		return d
	}
	for j, r := range s.S {
		if i == j {
			return Char{ch: r}
		}
	}
	return d
}

func (s String) Compare(other Object) int {
	s2 := AssertString(other, "Cannot compare String and "+other.GetType().ToString(false))
	return strings.Compare(s.S, s2.S)
}

func IsSymbol(obj Object) bool {
	switch obj.(type) {
	case Symbol:
		return true
	default:
		return false
	}
}

func IsVector(obj Object) bool {
	switch obj.(type) {
	case *Vector:
		return true
	default:
		return false
	}
}

func IsSeq(obj Object) bool {
	switch obj.(type) {
	case Seq:
		return true
	default:
		return false
	}
}

func (x *Type) WithInfo(info *ObjectInfo) Object {
	return x
}

func (x RecurBindings) WithInfo(info *ObjectInfo) Object {
	return x
}

func IsInstance(t *Type, obj Object) bool {
	if obj.Equals(NIL) {
		return false
	}
	if t.reflectType.Kind() == reflect.Interface {
		return obj.GetType().reflectType.Implements(t.reflectType)
	} else {
		return obj.GetType().reflectType == t.reflectType
	}
}
