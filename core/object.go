//go:generate go run gen_data/gen_data.go
//go:generate go run gen/gen_types.go assert Comparable *Vector Char String Symbol Keyword Regex Boolean Time Number Seqable Callable *Type Meta Int Double Stack Map Set Associative Reversible Named Comparator *Ratio *Namespace *Var Error *Fn Deref *Atom Ref KVReduce Pending
//go:generate go run gen/gen_types.go info *List *ArrayMapSeq *ArrayMap *HashMap *ExInfo *Fn *Var Nil *Ratio *BigInt *BigFloat Char Double Int Boolean Time Keyword Regex Symbol String *LazySeq *MappingSeq *ArraySeq *ConsSeq *NodeSeq *ArrayNodeSeq *MapSet *Vector *VectorSeq *VectorRSeq

package core

import (
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type (
	Position struct {
		endLine     int
		endColumn   int
		startLine   int
		startColumn int
		filename    *string
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
	Ref interface {
		AlterMeta(fn *Fn, args []Object) Map
		ResetMeta(m Map) Map
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
		Ch rune
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
	Boolean struct {
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
		R *regexp.Regexp
	}
	Time struct {
		InfoHolder
		T time.Time
	}
	Var struct {
		InfoHolder
		MetaHolder
		ns         *Namespace
		name       Symbol
		Value      Object
		expr       Expr
		isMacro    bool
		isPrivate  bool
		isDynamic  bool
		isUsed     bool
		taggedType *Type
	}
	Proc func([]Object) Object
	Fn   struct {
		InfoHolder
		MetaHolder
		isMacro bool
		fnExpr  *FnExpr
		env     *LocalEnv
	}
	ExInfo struct {
		ArrayMap
		rt *Runtime
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
	Printer interface {
		Print(writer io.Writer, printReadably bool)
	}
	Pprinter interface {
		Pprint(writer io.Writer, indent int) int
	}
	Collection interface {
		Object
		Counted
		Seqable
		Empty() Collection
	}
	Atom struct {
		MetaHolder
		value Object
	}
	Deref interface {
		Deref() Object
	}
	Native interface {
		Native() interface{}
	}
	KVReduce interface {
		kvreduce(c Callable, init Object) Object
	}
	Pending interface {
		IsRealized() bool
	}
	Types struct {
		Associative    *Type
		Callable       *Type
		Collection     *Type
		Comparable     *Type
		Comparator     *Type
		Counted        *Type
		Deref          *Type
		Error          *Type
		Gettable       *Type
		Indexed        *Type
		IOReader       *Type
		IOWriter       *Type
		KVReduce       *Type
		Map            *Type
		Meta           *Type
		Named          *Type
		Number         *Type
		Pending        *Type
		Ref            *Type
		Reversible     *Type
		Seq            *Type
		Seqable        *Type
		Sequential     *Type
		Set            *Type
		Stack          *Type
		ArrayMap       *Type
		ArrayMapSeq    *Type
		ArrayNodeSeq   *Type
		ArraySeq       *Type
		MapSet         *Type
		Atom           *Type
		BigFloat       *Type
		BigInt         *Type
		Boolean        *Type
		Time           *Type
		Buffer         *Type
		Char           *Type
		ConsSeq        *Type
		Delay          *Type
		Double         *Type
		EvalError      *Type
		ExInfo         *Type
		Fn             *Type
		File           *Type
		BufferedReader *Type
		HashMap        *Type
		Int            *Type
		Keyword        *Type
		LazySeq        *Type
		List           *Type
		MappingSeq     *Type
		Namespace      *Type
		Nil            *Type
		NodeSeq        *Type
		ParseError     *Type
		Proc           *Type
		Ratio          *Type
		RecurBindings  *Type
		Regex          *Type
		String         *Type
		Symbol         *Type
		Type           *Type
		Var            *Type
		Vector         *Type
		VectorRSeq     *Type
		VectorSeq      *Type
	}
)

var TYPES = map[*string]*Type{}
var TYPE Types

func regRefType(name string, inst interface{}) *Type {
	t := &Type{name: name, reflectType: reflect.TypeOf(inst)}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func regType(name string, inst interface{}) *Type {
	t := &Type{name: name, reflectType: reflect.TypeOf(inst).Elem()}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func regInterface(name string, inst interface{}) *Type {
	t := &Type{name: name, reflectType: reflect.TypeOf(inst).Elem()}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func init() {
	TYPE = Types{
		Associative:    regInterface("Associative", (*Associative)(nil)),
		Callable:       regInterface("Callable", (*Callable)(nil)),
		Collection:     regInterface("Collection", (*Collection)(nil)),
		Comparable:     regInterface("Comparable", (*Comparable)(nil)),
		Comparator:     regInterface("Comparator", (*Comparator)(nil)),
		Counted:        regInterface("Counted", (*Counted)(nil)),
		Deref:          regInterface("Deref", (*Deref)(nil)),
		Error:          regInterface("Error", (*Error)(nil)),
		Gettable:       regInterface("Gettable", (*Gettable)(nil)),
		Indexed:        regInterface("Indexed", (*Indexed)(nil)),
		IOReader:       regInterface("IOReader", (*io.Reader)(nil)),
		IOWriter:       regInterface("IOWriter", (*io.Writer)(nil)),
		KVReduce:       regInterface("KVReduce", (*KVReduce)(nil)),
		Map:            regInterface("Map", (*Map)(nil)),
		Meta:           regInterface("Meta", (*Meta)(nil)),
		Named:          regInterface("Named", (*Named)(nil)),
		Number:         regInterface("Number", (*Number)(nil)),
		Pending:        regInterface("Pending", (*Pending)(nil)),
		Ref:            regInterface("Ref", (*Ref)(nil)),
		Reversible:     regInterface("Reversible", (*Reversible)(nil)),
		Seq:            regInterface("Seq", (*Seq)(nil)),
		Seqable:        regInterface("Seqable", (*Seqable)(nil)),
		Sequential:     regInterface("Sequential", (*Sequential)(nil)),
		Set:            regInterface("Set", (*Set)(nil)),
		Stack:          regInterface("Stack", (*Stack)(nil)),
		ArrayMap:       regRefType("ArrayMap", (*ArrayMap)(nil)),
		ArrayMapSeq:    regRefType("ArrayMapSeq", (*ArrayMapSeq)(nil)),
		ArrayNodeSeq:   regRefType("ArrayNodeSeq", (*ArrayNodeSeq)(nil)),
		ArraySeq:       regRefType("ArraySeq", (*ArraySeq)(nil)),
		MapSet:         regRefType("MapSet", (*MapSet)(nil)),
		Atom:           regRefType("Atom", (*Atom)(nil)),
		BigFloat:       regRefType("BigFloat", (*BigFloat)(nil)),
		BigInt:         regRefType("BigInt", (*BigInt)(nil)),
		Boolean:        regType("Boolean", (*Boolean)(nil)),
		Time:           regType("Time", (*Time)(nil)),
		Buffer:         regRefType("Buffer", (*Buffer)(nil)),
		Char:           regType("Char", (*Char)(nil)),
		ConsSeq:        regRefType("ConsSeq", (*ConsSeq)(nil)),
		Delay:          regRefType("Delay", (*Delay)(nil)),
		Double:         regType("Double", (*Double)(nil)),
		EvalError:      regRefType("EvalError", (*EvalError)(nil)),
		ExInfo:         regRefType("ExInfo", (*ExInfo)(nil)),
		Fn:             regRefType("Fn", (*Fn)(nil)),
		File:           regRefType("File", (*File)(nil)),
		BufferedReader: regRefType("BufferedReader", (*BufferedReader)(nil)),
		HashMap:        regRefType("HashMap", (*HashMap)(nil)),
		Int:            regType("Int", (*Int)(nil)),
		Keyword:        regType("Keyword", (*Keyword)(nil)),
		LazySeq:        regRefType("LazySeq", (*LazySeq)(nil)),
		List:           regRefType("List", (*List)(nil)),
		MappingSeq:     regRefType("MappingSeq", (*MappingSeq)(nil)),
		Namespace:      regRefType("Namespace", (*Namespace)(nil)),
		Nil:            regType("Nil", (*Nil)(nil)),
		NodeSeq:        regRefType("NodeSeq", (*NodeSeq)(nil)),
		ParseError:     regRefType("ParseError", (*ParseError)(nil)),
		Proc:           regRefType("Proc", (*Proc)(nil)),
		Ratio:          regRefType("Ratio", (*Ratio)(nil)),
		RecurBindings:  regRefType("RecurBindings", (*RecurBindings)(nil)),
		Regex:          regType("Regex", (*Regex)(nil)),
		String:         regType("String", (*String)(nil)),
		Symbol:         regType("Symbol", (*Symbol)(nil)),
		Type:           regRefType("Type", (*Type)(nil)),
		Var:            regRefType("Var", (*Var)(nil)),
		Vector:         regRefType("Vector", (*Vector)(nil)),
		VectorRSeq:     regRefType("VectorRSeq", (*VectorRSeq)(nil)),
		VectorSeq:      regRefType("VectorSeq", (*VectorSeq)(nil)),
	}
}

func (pos Position) Filename() string {
	if pos.filename == nil {
		return "<file>"
	}
	return *pos.filename
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

func PanicArity(n int) {
	name := RT.currentExpr.(Traceable).Name()
	panic(RT.NewError(fmt.Sprintf("Wrong number of args (%d) passed to %s", n, name)))
}

func rangeString(min, max int) string {
	if min == max {
		return strconv.Itoa(min)
	}
	if min+1 == max {
		return strconv.Itoa(min) + " or " + strconv.Itoa(max)
	}
	if min+2 == max {
		return strconv.Itoa(min) + ", " + strconv.Itoa(min+1) + ", or " + strconv.Itoa(max)
	}
	if max >= 999 {
		return "at least " + strconv.Itoa(min)
	}
	return "between " + strconv.Itoa(min) + " and " + strconv.Itoa(max) + ", inclusive"
}

func PanicArityMinMax(n, min, max int) {
	name := RT.currentExpr.(Traceable).Name()
	panic(RT.NewError(fmt.Sprintf("Wrong number of args (%d) passed to %s; expects %s", n, name, rangeString(min, max))))
}

func CheckArity(args []Object, min int, max int) {
	n := len(args)
	if n < min || n > max {
		PanicArityMinMax(n, min, max)
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
	PanicOnErr(err)
	h.Write(b)
	return h.Sum32()
}

func equalsNumbers(x Number, y interface{}) bool {
	switch y := y.(type) {
	case Number:
		return category(x) == category(y) && numbersEq(x, y)
	default:
		return false
	}
}

func (a *Atom) ToString(escape bool) string {
	return "#object[Atom {:val " + a.value.ToString(escape) + "}]"
}

func (a *Atom) Equals(other interface{}) bool {
	return a == other
}

func (a *Atom) GetInfo() *ObjectInfo {
	return nil
}

func (a *Atom) GetType() *Type {
	return TYPE.Atom
}

func (a *Atom) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(a)))
}

func (a *Atom) WithInfo(info *ObjectInfo) Object {
	return a
}

func (a *Atom) WithMeta(meta Map) Object {
	res := *a
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (a *Atom) ResetMeta(newMeta Map) Map {
	a.meta = newMeta
	return a.meta
}

func (a *Atom) AlterMeta(fn *Fn, args []Object) Map {
	return AlterMeta(&a.MetaHolder, fn, args)
}

func (a *Atom) Deref() Object {
	return a.value
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
	return TYPE.Delay
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

func (d *Delay) Deref() Object {
	return d.Force()
}

func (d *Delay) IsRealized() bool {
	return d.value != nil
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
	return TYPE.Type
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
	return TYPE.RecurBindings
}

func (rb RecurBindings) Hash() uint32 {
	return 0
}

func (exInfo *ExInfo) ToString(escape bool) string {
	return exInfo.Error()
}

func (exInfo *ExInfo) Equals(other interface{}) bool {
	return exInfo == other
}

func (exInfo *ExInfo) GetType() *Type {
	return TYPE.ExInfo
}

func (exInfo *ExInfo) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(exInfo)))
}

func (exInfo *ExInfo) Error() string {
	var pos Position
	_, data := exInfo.Get(KEYWORDS.data)
	ok, form := data.(Map).Get(KEYWORDS.form)
	if ok {
		if form.GetInfo() != nil {
			pos = form.GetInfo().Pos()
		}
	}
	prefix := "Exception"
	if ok, pr := data.(Map).Get(KEYWORDS._prefix); ok {
		prefix = pr.ToString(false)
	}
	_, msg := exInfo.Get(KEYWORDS.message)
	if len(exInfo.rt.callstack.frames) > 0 && !LINTER_MODE {
		return fmt.Sprintf("%s:%d:%d: %s: %s\nStacktrace:\n%s", pos.Filename(), pos.startLine, pos.startColumn, prefix, msg.(String).S, exInfo.rt.stacktrace())
	} else {
		return fmt.Sprintf("%s:%d:%d: %s: %s", pos.Filename(), pos.startLine, pos.startColumn, prefix, msg.(String).S)
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
	return TYPE.Fn
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
		c := len(args)
		if fn.isMacro {
			c -= 2
		}
		PanicArity(c)
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
	case Boolean:
		if r.B {
			return -1
		}
		if AssertBoolean(c.Call([]Object{b, a}), "").B {
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
	return TYPE.Proc
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

func AlterMeta(m *MetaHolder, fn *Fn, args []Object) Map {
	meta := m.meta
	if meta == nil {
		meta = NIL
	}
	fargs := append([]Object{meta}, args...)
	m.meta = AssertMap(fn.Call(fargs), "")
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

func (v *Var) ResetMeta(newMeta Map) Map {
	v.meta = newMeta
	return v.meta
}

func (v *Var) AlterMeta(fn *Fn, args []Object) Map {
	return AlterMeta(&v.MetaHolder, fn, args)
}

func (v *Var) GetType() *Type {
	return TYPE.Var
}

func (v *Var) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(v)))
}

func (v *Var) Resolve() Object {
	if v.Value == nil {
		panic(RT.NewError("Unbound var: " + v.ToString(false)))
	}
	return v.Value
}

func (v *Var) Call(args []Object) Object {
	vl := v.Resolve()
	return AssertCallable(
		vl,
		"Var "+v.ToString(false)+" resolves to "+vl.ToString(false)+", which is not a Fn").Call(args)
}

func (v *Var) Deref() Object {
	return v.Resolve()
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
	return TYPE.Nil
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
	return equalsNumbers(rat, other)
}

func (rat *Ratio) GetType() *Type {
	return TYPE.Ratio
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
	return equalsNumbers(bi, other)
}

func (bi *BigInt) GetType() *Type {
	return TYPE.BigInt
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
	return equalsNumbers(bf, other)
}

func (bf *BigFloat) GetType() *Type {
	return TYPE.BigFloat
}

func (bf *BigFloat) Hash() uint32 {
	return hashGobEncoder(&bf.b)
}

func (bf *BigFloat) Compare(other Object) int {
	return CompareNumbers(bf, AssertNumber(other, "Cannot compare BigFloat and "+other.GetType().ToString(false)))
}

func (c Char) ToString(escape bool) string {
	if escape {
		return escapeRune(c.Ch)
	}
	return string(c.Ch)
}

func (c Char) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Char:
		return c.Ch == other.Ch
	default:
		return false
	}
}

func (c Char) GetType() *Type {
	return TYPE.Char
}

func (c Char) Native() interface{} {
	return c.Ch
}

func (c Char) Hash() uint32 {
	h := getHash()
	h.Write([]byte(string(c.Ch)))
	return h.Sum32()
}

func (c Char) Compare(other Object) int {
	c2 := AssertChar(other, "Cannot compare Char and "+other.GetType().ToString(false))
	if c.Ch < c2.Ch {
		return -1
	}
	if c2.Ch < c.Ch {
		return 1
	}
	return 0
}

func MakeBoolean(b bool) Boolean {
	return Boolean{B: b}
}

func MakeTime(t time.Time) Time {
	return Time{T: t}
}

func MakeDouble(d float64) Double {
	return Double{D: d}
}

func (d Double) ToString(escape bool) string {
	return fmt.Sprintf("%f", d.D)
}

func (d Double) Equals(other interface{}) bool {
	return equalsNumbers(d, other)
}

func (d Double) GetType() *Type {
	return TYPE.Double
}

func (d Double) Native() interface{} {
	return d.D
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

func MakeInt(i int) Int {
	return Int{I: i}
}

func (i Int) Equals(other interface{}) bool {
	return equalsNumbers(i, other)
}

func (i Int) GetType() *Type {
	return TYPE.Int
}

func (i Int) Native() interface{} {
	return i.I
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

func (b Boolean) ToString(escape bool) string {
	return fmt.Sprintf("%t", b.B)
}

func (b Boolean) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Boolean:
		return b.B == other.B
	default:
		return false
	}
}

func (b Boolean) GetType() *Type {
	return TYPE.Boolean
}

func (b Boolean) Native() interface{} {
	return b.B
}

func (b Boolean) Hash() uint32 {
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

func (b Boolean) Compare(other Object) int {
	b2 := AssertBoolean(other, "Cannot compare Boolean and "+other.GetType().ToString(false))
	if b.B == b2.B {
		return 0
	}
	if b.B {
		return 1
	}
	return -1
}

func (t Time) ToString(escape bool) string {
	return t.T.String()
}

func (t Time) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Time:
		return t.T.Equal(other.T)
	default:
		return false
	}
}

func (t Time) GetType() *Type {
	return TYPE.Time
}

func (t Time) Native() interface{} {
	return t.T
}

func (t Time) Hash() uint32 {
	return hashGobEncoder(t.T)
}

func (t Time) Compare(other Object) int {
	t2 := AssertTime(other, "Cannot compare Time and "+other.GetType().ToString(false))
	if t.T.Equal(t2.T) {
		return 0
	}
	if t2.T.Before(t.T) {
		return 1
	}
	return -1
}

func (k Keyword) ToString(escape bool) string {
	if k.ns != nil {
		return ":" + *k.ns + "/" + *k.name
	}
	return ":" + *k.name
}

func (k Keyword) Name() string {
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
	return TYPE.Keyword
}

func (k Keyword) Hash() uint32 {
	return k.hash
}

func (k Keyword) Compare(other Object) int {
	k2 := AssertKeyword(other, "Cannot compare Keyword and "+other.GetType().ToString(false))
	return strings.Compare(k.ToString(false), k2.ToString(false))
}

func (k Keyword) Call(args []Object) Object {
	CheckArity(args, 1, 2)
	switch m := args[0].(type) {
	case Map:
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
		return "#\"" + rx.R.String() + "\""
	}
	return rx.R.String()
}

func (rx Regex) Print(w io.Writer, printReadably bool) {
	fmt.Fprint(w, rx.ToString(true))
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
	return TYPE.Regex
}

func (rx Regex) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(rx.R)))
}

func (s Symbol) ToString(escape bool) string {
	if s.ns != nil {
		return *s.ns + "/" + *s.name
	}
	return *s.name
}

func (s Symbol) Name() string {
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
	return TYPE.Symbol
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
	return TYPE.String
}

func (s String) Native() interface{} {
	return s.S
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
		runes = append(runes, Char{Ch: r})
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
			return Char{Ch: r}
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
			return Char{Ch: r}
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

func IsEqualOrImplements(abstractType *Type, concreteType *Type) bool {
	if abstractType.reflectType.Kind() == reflect.Interface {
		return concreteType.reflectType.Implements(abstractType.reflectType)
	} else {
		return concreteType.reflectType == abstractType.reflectType
	}
}

func IsInstance(t *Type, obj Object) bool {
	if obj.Equals(NIL) {
		return false
	}
	return IsEqualOrImplements(t, obj.GetType())
}

func IsSpecialSymbol(obj Object) bool {
	switch obj := obj.(type) {
	case Symbol:
		return obj.ns == nil && SPECIAL_SYMBOLS[obj.name]
	default:
		return false
	}
}

func MakeMeta(arglists Seq, docstring string, added string) *ArrayMap {
	res := EmptyArrayMap()
	if arglists != nil {
		res.Add(KEYWORDS.arglist, arglists)
	}
	res.Add(KEYWORDS.doc, String{S: docstring})
	res.Add(KEYWORDS.added, String{S: added})
	return res
}
