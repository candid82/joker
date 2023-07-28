//go:generate go run gen/gen_types.go assert Comparable Vec Char String Symbol Keyword *Regex Boolean Time Number Seqable Callable *Type Meta Int Double Stack Map Set Associative Reversible Named Comparator *Ratio *BigFloat *BigInt *Namespace *Var Error *Fn Deref *Atom Ref KVReduce Pending *File io.Reader io.Writer StringReader io.RuneReader *Channel CountedIndexed
//go:generate go run gen/gen_types.go info *List *ArrayMapSeq *ArrayMap *HashMap *ExInfo *Fn *Var Nil *Ratio *BigInt *BigFloat Char Double Int Boolean Time Keyword *Regex Symbol String Comment *LazySeq *MappingSeq *ArraySeq *ConsSeq *NodeSeq *ArrayNodeSeq *MapSet *Vector *ArrayVector *VectorSeq *VectorRSeq
//go:generate go run -tags gen_code gen_code/gen_code.go

package core

import (
	"bytes"
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
	"unicode/utf8"
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
		MetaHolder
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
	CountedIndexed interface {
		Counted
		At(int) Object
	}
	Error interface {
		error
		Object
		Message() Object
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
		prefix string
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
		D        float64
		Original string
	}
	Int struct {
		InfoHolder
		I        int
		Original string
	}
	BigInt struct {
		InfoHolder
		b        *big.Int
		Original string
	}
	BigFloat struct {
		InfoHolder
		b        *big.Float
		Original string
	}
	Ratio struct {
		InfoHolder
		r        *big.Rat
		Original string
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
	Comment struct {
		InfoHolder
		C string
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
		ns             *Namespace
		name           Symbol
		Value          Object
		expr           Expr
		isMacro        bool
		isPrivate      bool
		isDynamic      bool
		isUsed         bool
		isGloballyUsed bool
		isFake         bool
		taggedType     *Type
	}
	ProcFn func([]Object) Object
	Proc   struct {
		Fn      ProcFn
		Name    string
		Package string // "" for core (this package), else e.g. "std/string"
	}
	Fn struct {
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
		EntryAt(key Object) *ArrayVector
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
	Formatter interface {
		Format(writer io.Writer, indent int) int
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
		CountedIndexed *Type
		Deref          *Type
		Channel        *Type
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
		ProcFn         *Type
		Ratio          *Type
		RecurBindings  *Type
		Regex          *Type
		String         *Type
		Symbol         *Type
		Type           *Type
		Var            *Type
		Vector         *Type
		Vec            *Type
		ArrayVector    *Type
		VectorRSeq     *Type
		VectorSeq      *Type
	}
)

func (pos Position) Filename() string {
	if pos.filename == nil {
		return "<file>"
	}
	return *pos.filename
}

func newIteratorError() error {
	return errors.New("Iterator reached the end of collection")
}

func uint32ToBytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, i)
	return b
}

func getHash() hash.Hash32 {
	return fnv.New32a()
}

func hashSymbol(ns, name *string) uint32 {
	h := getHash()
	if ns != nil {
		h.Write([]byte(*ns))
	}
	h.Write([]byte("/" + *name))
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

type BySymbolName []Symbol

func (s BySymbolName) Len() int {
	return len(s)
}
func (s BySymbolName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s BySymbolName) Less(i, j int) bool {
	return s[i].ToString(false) < s[j].ToString(false)
}

const KeywordHashMask uint32 = 0x7334c790

func MakeKeyword(nsname string) Keyword {
	index := strings.IndexRune(nsname, '/')
	if index == -1 || nsname == "/" {
		name := STRINGS.Intern(nsname)
		return Keyword{
			ns:   nil,
			name: name,
			hash: hashSymbol(nil, name) ^ KeywordHashMask,
		}
	}
	ns := STRINGS.Intern(nsname[0:index])
	name := STRINGS.Intern(nsname[index+1 : len(nsname)])
	return Keyword{
		ns:   ns,
		name: name,
		hash: hashSymbol(ns, name) ^ KeywordHashMask,
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

func getMap(k Object, args []Object) Object {
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

func (s SortableSlice) Len() int {
	return len(s.s)
}

func (s SortableSlice) Swap(i, j int) {
	s.s[i], s.s[j] = s.s[j], s.s[i]
}

func (s SortableSlice) Less(i, j int) bool {
	return s.cmp.Compare(s.s[i], s.s[j]) == -1
}

func HashPtr(ptr uintptr) uint32 {
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
	return HashPtr(uintptr(unsafe.Pointer(a)))
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
	return HashPtr(uintptr(unsafe.Pointer(d)))
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
	return HashPtr(uintptr(unsafe.Pointer(t)))
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
	return HashPtr(uintptr(unsafe.Pointer(exInfo)))
}

func (exInfo *ExInfo) Message() Object {
	if ok, res := exInfo.Get(KEYWORDS.message); ok {
		return res
	}
	return NIL
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
	return HashPtr(uintptr(unsafe.Pointer(fn)))
}

func (fn *Fn) Call(args []Object) Object {
	min := math.MaxInt32
	max := -1
	for _, arity := range fn.fnExpr.arities {
		a := len(arity.args)
		if a == len(args) {
			RT.pushFrame()
			defer RT.popFrame()
			return evalLoop(arity.body, fn.env.addFrame(args))
		}
		if min > a {
			min = a
		}
		if max < a {
			max = a
		}
	}
	v := fn.fnExpr.variadic
	if v == nil || len(args) < len(v.args)-1 {
		if v != nil {
			min = len(v.args)
			max = math.MaxInt32
		}
		c := len(args)
		if fn.isMacro {
			c -= 2
			min -= 2
			if max != math.MaxInt32 {
				max -= 2
			}
		}
		PanicArityMinMax(c, min, max)
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
		if EnsureObjectIsBoolean(c.Call([]Object{b, a}), "").B {
			return 1
		}
		return 0
	default:
		return EnsureObjectIsNumber(r, "Function is not a comparator since it returned a non-integer value%.s").Int().I
	}
}

func (fn *Fn) Compare(a, b Object) int {
	return compare(fn, a, b)
}

func (p Proc) Call(args []Object) Object {
	return p.Fn(args)
}

func (p Proc) Compare(a, b Object) int {
	return compare(p, a, b)
}

func (p Proc) ToString(escape bool) string {
	pkg := p.Package
	if pkg != "" {
		pkg += "."
	}
	return fmt.Sprintf("#object[Proc:%s%s]", pkg, p.Name)
}

func (p Proc) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Proc:
		return reflect.ValueOf(p.Fn).Pointer() == reflect.ValueOf(other.Fn).Pointer()
	}
	return false
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
	return HashPtr(reflect.ValueOf(p.Fn).Pointer())
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
	m.meta = EnsureObjectIsMap(fn.Call(fargs), "")
	return m.meta
}

func (sym Symbol) WithMeta(meta Map) Object {
	res := sym
	res.meta = SafeMerge(res.meta, meta)
	return res
}

func (v *Var) Name() string {
	return v.ns.Name.ToString(false) + "/" + v.name.ToString(false)
}

func (v *Var) ToString(escape bool) string {
	return "#'" + v.Name()
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
	return HashPtr(uintptr(unsafe.Pointer(v)))
}

func (v *Var) Resolve() Object {
	if v.Value == nil {
		return NIL
	}
	return v.Value
}

func (v *Var) Call(args []Object) Object {
	vl := v.Resolve()
	return EnsureObjectIsCallable(
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

func (n Nil) EntryAt(key Object) *ArrayVector {
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
	return hashGobEncoder(rat.r)
}

func (rat *Ratio) Compare(other Object) int {
	return CompareNumbers(rat, EnsureObjectIsNumber(other, "Cannot compare Ratio: %s"))
}

func MakeBigInt(b *big.Int) *BigInt {
	return &BigInt{b: b}
}

// Helper function that returns a math/big.Int given an int.
func MakeMathBigIntFromInt(i int) *big.Int {
	return MakeMathBigIntFromInt64(int64(i))
}

// Helper function that returns a math/big.Int given an int64.
func MakeMathBigIntFromInt64(i int64) *big.Int {
	return big.NewInt(i)
}

// Helper function that returns a math/big.Int given a uint.
func MakeMathBigIntFromUint(b uint) *big.Int {
	return MakeMathBigIntFromUint64(uint64(b))
}

// Helper function that returns a math/big.Int given a uint64.
func MakeMathBigIntFromUint64(b uint64) *big.Int {
	bigint := big.NewInt(0)
	bigint.SetUint64(b)
	return bigint
}

func (bi *BigInt) ToString(escape bool) string {
	if FORMAT_MODE && bi.Original != "" {
		return bi.Original
	}
	return bi.b.String() + "N"
}

func (bi *BigInt) Equals(other interface{}) bool {
	return equalsNumbers(bi, other)
}

func (bi *BigInt) GetType() *Type {
	return TYPE.BigInt
}

func (bi *BigInt) Hash() uint32 {
	return hashGobEncoder(bi.b)
}

func (bi *BigInt) Compare(other Object) int {
	return CompareNumbers(bi, EnsureObjectIsNumber(other, "Cannot compare BigInt: %s"))
}

// Determine the precision for a float-point constant, with float64
// precision (53) being the minimum value. No need to be as strict
// when parsing it as is math/big.(Float).Parse().
func computePrecision(s string) (prec uint) {
	prec = 53 // Default to precision for float64
	if s == "" {
		return
	}
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}
	if s == "Inf" || s == "inf" || s == "NaN" {
		return
	}

	bitsNeeded := 0.

	// Assume base 10 at first.
	bitsPerDigit := 3.33 // (joker.math/log-2 10) => 3.32192809488736
	exponentUpper, exponentLower := 'E', 'e'

	if len(s) > 2 && s[0] == '0' && strings.ContainsAny(s[1:2], "bBoOxX") {
		switch s[1] {
		case 'b', 'B':
			bitsPerDigit = 1
		case 'o', 'O':
			bitsPerDigit = 3
		case 'x', 'X':
			bitsPerDigit = 4
		default:
			panic(fmt.Sprintf("internal error examining %q", s))
		}
		exponentUpper, exponentLower = 'P', 'p'
		s = s[2:]
	}

	for _, c := range s {
		if c == exponentUpper || c == exponentLower {
			break
		}
		if ('0' <= c && c <= '9') || ('A' <= c && c <= 'F') || ('a' <= c && c <= 'f') {
			bitsNeeded += bitsPerDigit
		}
	}

	bitsNeeded = math.Max(float64(prec), math.Ceil(bitsNeeded)) // Round up, then return >= 53
	return uint(bitsNeeded)
}

func MakeBigFloat(b *big.Float) *BigFloat {
	return &BigFloat{b: b}
}

// Helper function that returns a BigFloat given a string, remembering
// any original string provided, and true if the string had the proper
// format; nil and false otherwise.
func MakeBigFloatWithOrig(s, orig string) (*BigFloat, bool) {
	prec := computePrecision(s)
	f := new(big.Float)
	f.SetPrec(uint(prec))

	if _, ok := f.SetString(s); ok {
		return &BigFloat{b: f, Original: orig}, true
	}

	return nil, false
}

func (bf *BigFloat) ToString(escape bool) string {
	if FORMAT_MODE && bf.Original != "" {
		return bf.Original
	}
	b := bf.b
	if b.IsInf() {
		if b.Signbit() {
			return "##-Inf"
		}
		return "##Inf"
	}
	return b.Text('g', -1) + "M"
}

func (bf *BigFloat) Equals(other interface{}) bool {
	return equalsNumbers(bf, other)
}

func (bf *BigFloat) GetType() *Type {
	return TYPE.BigFloat
}

func (bf *BigFloat) Hash() uint32 {
	return hashGobEncoder(bf.b)
}

func (bf *BigFloat) Compare(other Object) int {
	return CompareNumbers(bf, EnsureObjectIsNumber(other, "Cannot compare BigFloat: %s"))
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
	c2 := EnsureObjectIsChar(other, "Cannot compare Char: %s")
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
	if FORMAT_MODE && d.Original != "" {
		return d.Original
	}
	dbl := d.D
	if math.IsInf(dbl, 1) {
		return "##Inf"
	}
	if math.IsInf(dbl, -1) {
		return "##-Inf"
	}
	if math.IsNaN(dbl) {
		return "##NaN"
	}
	res := fmt.Sprintf("%g", dbl)
	if strings.ContainsAny(res, ".e") {
		return res
	}
	return res + ".0"
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
	return CompareNumbers(d, EnsureObjectIsNumber(other, "Cannot compare Double: %s"))
}

func (i Int) ToString(escape bool) string {
	if FORMAT_MODE && i.Original != "" {
		return i.Original
	}
	return fmt.Sprintf("%d", i.I)
}

func MakeInt(i int) Int {
	return Int{I: i}
}

func MakeIntVector(ii []int) *ArrayVector {
	res := EmptyArrayVector()
	for _, i := range ii {
		res.Append(MakeInt(i))
	}
	return res
}

func MakeIntWithOriginal(orig string, i int) Int {
	return Int{I: i, Original: orig}
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
	return CompareNumbers(i, EnsureObjectIsNumber(other, "Cannot compare Int: %s"))
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
	b2 := EnsureObjectIsBoolean(other, "Cannot compare Boolean: %s")
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
	t2 := EnsureObjectIsTime(other, "Cannot compare Time: %s")
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
	k2 := EnsureObjectIsKeyword(other, "Cannot compare Keyword: %s")
	return strings.Compare(k.ToString(false), k2.ToString(false))
}

func (k Keyword) Call(args []Object) Object {
	return getMap(k, args)
}

func MakeRegex(r *regexp.Regexp) *Regex {
	return &Regex{R: r}
}

func (rx *Regex) ToString(escape bool) string {
	if escape {
		return "#\"" + rx.R.String() + "\""
	}
	return rx.R.String()
}

func (rx *Regex) Print(w io.Writer, printReadably bool) {
	fmt.Fprint(w, rx.ToString(true))
}

func (rx *Regex) Equals(other interface{}) bool {
	switch other := other.(type) {
	case *Regex:
		return rx.R == other.R
	default:
		return false
	}
}

func (rx *Regex) GetType() *Type {
	return TYPE.Regex
}

func (rx *Regex) Hash() uint32 {
	return HashPtr(uintptr(unsafe.Pointer(rx.R)))
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
	s2 := EnsureObjectIsSymbol(other, "Cannot compare Symbol: %s")
	return strings.Compare(s.ToString(false), s2.ToString(false))
}

func (s Symbol) Call(args []Object) Object {
	return getMap(s, args)
}

func (c Comment) ToString(escape bool) string {
	return c.C
}

func (c Comment) Equals(other interface{}) bool {
	return false
}

func (c Comment) GetType() *Type {
	// Comments don't deserve their own type
	// since they are only used in FORMAT mode.
	return TYPE.String
}

func (c Comment) Hash() uint32 {
	h := getHash()
	h.Write([]byte(c.C))
	return h.Sum32()
}

func (s String) ToString(escape bool) string {
	if escape {
		return escapeString(s.S)
	}
	return s.S
}

func (s String) Format(w io.Writer, indent int) int {
	fmt.Fprint(w, "\"", s.S, "\"")
	return indent + utf8.RuneCountInString(s.S) + 2
}

func MakeString(s string) String {
	return String{S: s}
}

func MakeStringVector(ss []string) *ArrayVector {
	res := EmptyArrayVector()
	for _, s := range ss {
		res.Append(MakeString(s))
	}
	return res
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
	return utf8.RuneCountInString(s.S)
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
	s2 := EnsureObjectIsString(other, "Cannot compare String: %s")
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

func IsKeyword(obj Object) bool {
	_, ok := obj.(Keyword)
	return ok
}

func IsVector(obj Object) bool {
	switch obj.(type) {
	case Vec:
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

func RegRefType(name string, inst interface{}, doc string) *Type {
	if doc != "" {
		doc = "\n  " + doc
	}
	meta := MakeMeta(nil, "(Concrete reference type)"+doc, "1.0")
	meta.Add(KEYWORDS.name, MakeString(name))
	t := &Type{MetaHolder{meta}, name, reflect.TypeOf(inst)}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func RegType(name string, inst interface{}, doc string) *Type {
	if doc != "" {
		doc = "\n  " + doc
	}
	meta := MakeMeta(nil, "(Concrete type)"+doc, "1.0")
	meta.Add(KEYWORDS.name, MakeString(name))
	t := &Type{MetaHolder{meta}, name, reflect.TypeOf(inst).Elem()}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func RegInterface(name string, inst interface{}, doc string) *Type {
	if doc != "" {
		doc = "\n  " + doc
	}
	meta := MakeMeta(nil, "(Interface type)"+doc, "1.0")
	meta.Add(KEYWORDS.name, MakeString(name))
	t := &Type{MetaHolder{meta}, name, reflect.TypeOf(inst).Elem()}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func CountedIndexedToString(v CountedIndexed, escape bool) string {
	var b bytes.Buffer
	b.WriteRune('[')
	cnt := v.Count()
	if cnt > 0 {
		for i := 0; i < cnt-1; i++ {
			b.WriteString(v.At(i).ToString(escape))
			b.WriteRune(' ')
		}
		b.WriteString(v.At(cnt - 1).ToString(escape))
	}
	b.WriteRune(']')
	return b.String()
}

func AreCountedIndexedEqual(v1, v2 CountedIndexed) bool {
	if v1.Count() != v2.Count() {
		return false
	}
	for i := 0; i < v1.Count(); i++ {
		if !v1.At(i).Equals(v2.At(i)) {
			return false
		}
	}
	return true
}

func CountedIndexedHash(v CountedIndexed) uint32 {
	h := getHash()
	for i := 0; i < v.Count(); i++ {
		h.Write(uint32ToBytes(v.At(i).Hash()))
	}
	return h.Sum32()
}

func CountedIndexedGet(v CountedIndexed, key Object) (bool, Object) {
	switch key := key.(type) {
	case Int:
		if key.I >= 0 && key.I < v.Count() {
			return true, v.At(key.I)
		}
	}
	return false, nil
}

func CountedIndexedCompare(v1, v2 CountedIndexed) int {
	if v1.Count() > v2.Count() {
		return 1
	}
	if v1.Count() < v2.Count() {
		return -1
	}
	for i := 0; i < v1.Count(); i++ {
		c := EnsureObjectIsComparable(v1.At(i), "").Compare(v2.At(i))
		if c != 0 {
			return c
		}
	}
	return 0
}

func CountedIndexedKvreduce(v CountedIndexed, c Callable, init Object) Object {
	res := init
	for i := 0; i < v.Count(); i++ {
		res = c.Call([]Object{res, Int{I: i}, v.At(i)})
	}
	return res
}

func CountedIndexedPprint(v CountedIndexed, w io.Writer, indent int) int {
	ind := indent + 1
	fmt.Fprint(w, "[")
	if v.Count() > 0 {
		for i := 0; i < v.Count()-1; i++ {
			pprintObject(v.At(i), indent+1, w)
			fmt.Fprint(w, "\n")
			writeIndent(w, indent+1)
		}
		ind = pprintObject(v.At(v.Count()-1), indent+1, w)
	}
	fmt.Fprint(w, "]")
	return ind + 1
}

func CountedIndexedFormat(v CountedIndexed, w io.Writer, indent int) int {
	ind := indent + 1
	fmt.Fprint(w, "[")
	if v.Count() > 0 {
		for i := 0; i < v.Count()-1; i++ {
			ind = formatObject(v.At(i), ind, w)

			ind = maybeNewLine(w, v.At(i), v.At(i+1), indent+1, ind)
		}
		ind = formatObject(v.At(v.Count()-1), ind, w)
	}
	if v.Count() > 0 {
		if isComment(v.At(v.Count() - 1)) {
			fmt.Fprint(w, "\n")
			writeIndent(w, indent+1)
			ind = indent + 1
		}
	}
	fmt.Fprint(w, "]")
	return ind + 1
}
