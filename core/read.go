package core

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type (
	ReadError struct {
		line     int
		column   int
		filename *string
		msg      string
	}
	ReadFunc func(reader *Reader) Object
	pos      struct {
		line   int
		column int
	}
)

const EOF = -1

var LINTER_MODE bool = false
var DIALECT Dialect
var LINTER_CONFIG *Var

var (
	ARGS   map[int]Symbol
	GENSYM int
)

func readStub(reader *Reader) Object {
	return Read(reader)
}

var NIL = Nil{}
var posStack = make([]pos, 0, 8)

func pushPos(reader *Reader) {
	posStack = append(posStack, pos{line: reader.line, column: reader.column})
}

func popPos() pos {
	p := posStack[len(posStack)-1]
	posStack = posStack[:len(posStack)-1]
	return p
}

func escapeRune(r rune) string {
	switch r {
	case ' ':
		return "\\space"
	case '\n':
		return "\\newline"
	case '\t':
		return "\\tab"
	case '\r':
		return "\\return"
	case '\b':
		return "\\backspace"
	case '\f':
		return "\\formfeed"
	default:
		return "\\" + string(r)
	}
}

func escapeString(str string) string {
	var b bytes.Buffer
	b.WriteRune('"')
	for _, r := range str {
		switch r {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\t':
			b.WriteString("\\t")
		case '\r':
			b.WriteString("\\r")
		case '\n':
			b.WriteString("\\n")
		case '\f':
			b.WriteString("\\f")
		case '\b':
			b.WriteString("\\b")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteRune('"')
	return b.String()
}

func MakeReadError(reader *Reader, msg string) ReadError {
	return ReadError{
		line:     reader.line,
		column:   reader.column,
		filename: reader.filename,
		msg:      msg,
	}
}

func MakeReadObject(reader *Reader, obj Object) Object {
	p := popPos()
	return obj.WithInfo(&ObjectInfo{Position: Position{
		startColumn: p.column,
		startLine:   p.line,
		endLine:     reader.line,
		endColumn:   reader.column,
		filename:    reader.filename,
	}})
}

func DeriveReadObject(base Object, obj Object) Object {
	baseInfo := base.GetInfo()
	if baseInfo != nil {
		bi := *baseInfo
		return obj.WithInfo(&bi)
	}
	return obj
}

func (err ReadError) Error() string {
	return fmt.Sprintf("%s:%d:%d: Read error: %s", filename(err.filename), err.line, err.column, err.msg)
}

func isDelimiter(r rune) bool {
	switch r {
	case '(', ')', '[', ']', '{', '}', '"', ';', EOF, ',', '\\':
		return true
	}
	return unicode.IsSpace(r)
}

func eatString(reader *Reader, str string) {
	for _, sr := range str {
		if r := reader.Get(); r != sr {
			panic(MakeReadError(reader, fmt.Sprintf("Unexpected character %U", r)))
		}
	}
}

func peekExpectedDelimiter(reader *Reader) {
	r := reader.Peek()
	if !isDelimiter(r) {
		panic(MakeReadError(reader, "Character not followed by delimiter"))
	}
}

func readSpecialCharacter(reader *Reader, ending string, r rune) Object {
	eatString(reader, ending)
	peekExpectedDelimiter(reader)
	return MakeReadObject(reader, Char{Ch: r})
}

func eatWhitespace(reader *Reader) {
	r := reader.Get()
	for r != EOF {
		if unicode.IsSpace(r) || r == ',' {
			r = reader.Get()
			continue
		}
		if r == ';' || (r == '#' && reader.Peek() == '!') {
			for r != '\n' && r != EOF {
				r = reader.Get()
			}
			r = reader.Get()
			continue
		}
		if r == '#' && reader.Peek() == '_' {
			reader.Get()
			Read(reader)
			r = reader.Get()
			continue
		}
		reader.Unget()
		break
	}
}

func readUnicodeCharacter(reader *Reader, length, base int) Object {
	var b bytes.Buffer
	for n := reader.Get(); !isDelimiter(n); n = reader.Get() {
		b.WriteRune(n)
	}
	reader.Unget()
	str := b.String()
	if len(str) != length {
		panic(MakeReadError(reader, "Invalid unicode character: \\o"+str))
	}
	i, err := strconv.ParseInt(str, base, 32)
	if err != nil {
		panic(MakeReadError(reader, "Invalid unicode character: \\o"+str))
	}
	peekExpectedDelimiter(reader)
	return MakeReadObject(reader, Char{Ch: rune(i)})
}

func readCharacter(reader *Reader) Object {
	r := reader.Get()
	if r == EOF {
		panic(MakeReadError(reader, "Incomplete character literal"))
	}
	switch r {
	case 's':
		if reader.Peek() == 'p' {
			return readSpecialCharacter(reader, "pace", ' ')
		}
	case 'n':
		if reader.Peek() == 'e' {
			return readSpecialCharacter(reader, "ewline", '\n')
		}
	case 't':
		if reader.Peek() == 'a' {
			return readSpecialCharacter(reader, "ab", '\t')
		}
	case 'f':
		if reader.Peek() == 'o' {
			return readSpecialCharacter(reader, "ormfeed", '\f')
		}
	case 'b':
		if reader.Peek() == 'a' {
			return readSpecialCharacter(reader, "ackspace", '\b')
		}
	case 'r':
		if reader.Peek() == 'e' {
			return readSpecialCharacter(reader, "eturn", '\r')
		}
	case 'u':
		if !isDelimiter(reader.Peek()) {
			return readUnicodeCharacter(reader, 4, 16)
		}
	case 'o':
		if !isDelimiter(reader.Peek()) {
			readUnicodeCharacter(reader, 3, 8)
		}
	}
	peekExpectedDelimiter(reader)
	return MakeReadObject(reader, Char{Ch: r})
}

func scanBigInt(str string, base int, err error, reader *Reader) Object {
	var bi big.Int
	if _, ok := bi.SetString(str, base); !ok {
		panic(err)
	}
	res := BigInt{b: bi}
	return MakeReadObject(reader, &res)
}

func scanRatio(str string, err error, reader *Reader) Object {
	var rat big.Rat
	if _, ok := rat.SetString(str); !ok {
		panic(err)
	}
	res := Ratio{r: rat}
	return MakeReadObject(reader, &res)
}

func scanBigFloat(str string, err error, reader *Reader) Object {
	var bf big.Float
	if _, ok := bf.SetPrec(256).SetString(str); !ok {
		panic(err)
	}
	res := BigFloat{b: bf}
	return MakeReadObject(reader, &res)
}

func scanInt(str string, base int, err error, reader *Reader) Object {
	i, e := strconv.ParseInt(str, base, 0)
	if e != nil {
		return scanBigInt(str, base, err, reader)
	}
	return MakeReadObject(reader, Int{I: int(i)})
}

func readNumber(reader *Reader) Object {
	var b bytes.Buffer
	isDouble, isHex, isExp, isRatio, base, nonDigits := false, false, false, false, "", 0
	d := reader.Get()
	last := d
	for !isDelimiter(d) {
		switch d {
		case '.':
			isDouble = true
		case '/':
			isRatio = true
		case 'x', 'X':
			isHex = true
		case 'e', 'E':
			isExp = true
		case 'r', 'R':
			if base == "" {
				base = b.String()
				b.Reset()
				last = d
				d = reader.Get()
				continue
			}
		}
		if !unicode.IsDigit(d) {
			nonDigits++
		}
		b.WriteRune(d)
		last = d
		d = reader.Get()
	}
	reader.Unget()
	str := b.String()
	if base != "" {
		invalidNumberError := MakeReadError(reader, fmt.Sprintf("Invalid number: %s", base+"r"+str))
		baseInt, err := strconv.ParseInt(base, 0, 0)
		if err != nil {
			panic(invalidNumberError)
		}
		if base[0] == '-' {
			baseInt = -baseInt
			str = "-" + str
		}
		if baseInt < 2 || baseInt > 36 {
			panic(invalidNumberError)
		}
		return scanInt(str, int(baseInt), invalidNumberError, reader)
	}
	invalidNumberError := MakeReadError(reader, fmt.Sprintf("Invalid number: %s", str))
	if isRatio {
		if nonDigits > 2 || nonDigits > 1 && str[0] != '-' {
			panic(invalidNumberError)
		}
		return scanRatio(str, invalidNumberError, reader)
	}
	if last == 'N' {
		b.Truncate(b.Len() - 1)
		return scanBigInt(b.String(), 0, invalidNumberError, reader)
	}
	if last == 'M' {
		b.Truncate(b.Len() - 1)
		return scanBigFloat(b.String(), invalidNumberError, reader)
	}
	if isDouble || (!isHex && isExp) {
		dbl, err := strconv.ParseFloat(str, 64)
		if err != nil {
			panic(invalidNumberError)
		}
		return MakeReadObject(reader, Double{D: dbl})
	}
	return scanInt(str, 0, invalidNumberError, reader)
}

func isSymbolInitial(r rune) bool {
	switch r {
	case '*', '+', '!', '-', '_', '?', ':', '=', '<', '>', '&', '.', '%', '$', '|':
		return true
	}
	return unicode.IsLetter(r)
}

func isSymbolRune(r rune) bool {
	return isSymbolInitial(r) || unicode.IsDigit(r) || r == '#' || r == '/' || r == '\''
}

func readSymbol(reader *Reader, first rune) Object {
	var b bytes.Buffer
	if first != ':' {
		b.WriteRune(first)
	}
	var lastAdded rune
	r := reader.Get()
	for isSymbolRune(r) {
		if r == ':' {
			if lastAdded == ':' {
				panic(MakeReadError(reader, "Invalid use of ':' in symbol name"))
			}
		}
		b.WriteRune(r)
		lastAdded = r
		r = reader.Get()
	}
	if lastAdded == ':' || lastAdded == '/' {
		panic(MakeReadError(reader, fmt.Sprintf("Invalid use of %c in symbol name", lastAdded)))
	}
	reader.Unget()
	str := b.String()
	switch {
	case first == ':':
		if str[0] == '/' {
			panic(MakeReadError(reader, "Blank namespaces are not allowed"))
		}
		if str[0] == ':' {
			sym := MakeSymbol(str[1:])
			ns := GLOBAL_ENV.NamespaceFor(GLOBAL_ENV.CurrentNamespace(), sym)
			if ns == nil {
				msg := fmt.Sprintf("Unable to resolve namespace %s in keyword %s", *sym.ns, ":"+str)
				if LINTER_MODE {
					printReadWarning(reader, msg)
					return MakeReadObject(reader, MakeKeyword(*sym.name))
				}
				panic(MakeReadError(reader, msg))
			}
			ns.isUsed = true
			return MakeReadObject(reader, MakeKeyword(*ns.Name.name+"/"+*sym.name))
		}
		return MakeReadObject(reader, MakeKeyword(str))
	case str == "nil":
		return MakeReadObject(reader, NIL)
	case str == "true":
		return MakeReadObject(reader, Bool{B: true})
	case str == "false":
		return MakeReadObject(reader, Bool{B: false})
	default:
		return MakeReadObject(reader, MakeSymbol(str))
	}
}

func readRegex(reader *Reader) Object {
	var b bytes.Buffer
	r := reader.Get()
	for r != '"' {
		if r == EOF {
			panic(MakeReadError(reader, "Non-terminated regex literal"))
		}
		b.WriteRune(r)
		if r == '\\' {
			r = reader.Get()
			if r == EOF {
				panic(MakeReadError(reader, "Non-terminated regex literal"))
			}
			b.WriteRune(r)
		}
		r = reader.Get()
	}
	regex, err := regexp.Compile(b.String())
	if err != nil {
		if LINTER_MODE {
			return MakeReadObject(reader, Regex{})
		}
		panic(MakeReadError(reader, "Invalid regex: "+err.Error()))
	}
	return MakeReadObject(reader, Regex{R: regex})
}

func readString(reader *Reader) Object {
	var b bytes.Buffer
	r := reader.Get()
	for r != '"' {
		if r == '\\' {
			r = reader.Get()
			switch r {
			case '\\':
				r = '\\'
			case '"':
				r = '"'
			case 'n':
				r = '\n'
			case 't':
				r = '\t'
			case 'r':
				r = '\r'
			case 'b':
				r = '\b'
			case 'f':
				r = '\f'
			case 'u':
				var b bytes.Buffer
				n := reader.Get()
				for i := 0; i < 4 && n != '"'; i++ {
					b.WriteRune(n)
					n = reader.Get()
				}
				reader.Unget()
				str := b.String()
				if len(str) != 4 {
					panic(MakeReadError(reader, "Invalid unicode escape: \\u"+str))
				}
				i, err := strconv.ParseInt(str, 16, 32)
				if err != nil {
					panic(MakeReadError(reader, "Invalid unicode escape: \\u"+str))
				}
				r = rune(i)
			default:
				panic(MakeReadError(reader, "Unsupported escape character: \\"+string(r)))
			}
		}
		if r == EOF {
			panic(MakeReadError(reader, "Non-terminated string literal"))
		}
		b.WriteRune(r)
		r = reader.Get()
	}
	return MakeReadObject(reader, String{S: b.String()})
}

func readList(reader *Reader) Object {
	s := make([]Object, 0, 10)
	eatWhitespace(reader)
	r := reader.Peek()
	for r != ')' {
		obj := Read(reader)
		s = append(s, obj)
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	list := EmptyList
	for i := len(s) - 1; i >= 0; i-- {
		list = list.conj(s[i])
	}
	res := MakeReadObject(reader, list)
	return res
}

func readVector(reader *Reader) Object {
	result := EmptyVector
	eatWhitespace(reader)
	r := reader.Peek()
	for r != ']' {
		obj := Read(reader)
		result = result.Conjoin(obj)
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return MakeReadObject(reader, result)
}

func readHashMap(reader *Reader, hashMap *HashMap) Object {
	eatWhitespace(reader)
	r := reader.Peek()
	for r != '}' {
		key := Read(reader)
		value := Read(reader)
		if hashMap.containsKey(key) {
			panic(MakeReadError(reader, "Duplicate key "+key.ToString(false)))
		}
		hashMap = hashMap.Assoc(key, value).(*HashMap)
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return MakeReadObject(reader, hashMap)
}

func readMap(reader *Reader) Object {
	m := EmptyArrayMap()
	eatWhitespace(reader)
	r := reader.Peek()
	for r != '}' {
		key := Read(reader)
		value := Read(reader)
		if !m.Add(key, value) {
			panic(MakeReadError(reader, "Duplicate key "+key.ToString(false)))
		}
		if len(m.arr) > HASHMAP_THRESHOLD {
			hashMap := NewHashMap(m.arr...)
			return readHashMap(reader, hashMap)
		}
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return MakeReadObject(reader, m)
}

func readSet(reader *Reader) Object {
	set := EmptySet()
	eatWhitespace(reader)
	r := reader.Peek()
	for r != '}' {
		obj := Read(reader)
		if !set.Add(obj) {
			panic(MakeReadError(reader, "Duplicate set element "+obj.ToString(false)))
		}
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return MakeReadObject(reader, set)
}

func makeQuote(obj Object, quote Symbol) Object {
	res := NewListFrom(quote, obj)
	return DeriveReadObject(obj, res)
}

func readMeta(reader *Reader) *ArrayMap {
	obj := Read(reader)
	switch v := obj.(type) {
	case *ArrayMap:
		return v
	case String, Symbol:
		return &ArrayMap{arr: []Object{DeriveReadObject(obj, KEYWORDS.tag), obj}}
	case Keyword:
		return &ArrayMap{arr: []Object{obj, DeriveReadObject(obj, Bool{B: true})}}
	default:
		panic(MakeReadError(reader, "Metadata must be Symbol, Keyword, String or Map"))
	}
}

func fillInMissingArgs(args map[int]Symbol) {
	max := 0
	for k := range args {
		if k > max {
			max = k
		}
	}
	for i := 1; i < max; i++ {
		if _, ok := args[i]; !ok {
			args[i] = generateSymbol("p__")
		}
	}
}

func makeFnForm(args map[int]Symbol, body Object) Object {
	fillInMissingArgs(args)
	a := make([]Symbol, len(args))
	for key, value := range args {
		if key != -1 {
			a[key-1] = value
		}
	}
	if v, ok := args[-1]; ok {
		a[len(args)-1] = SYMBOLS.amp
		a = append(a, v)
	}
	argVector := EmptyVector
	for _, v := range a {
		argVector = argVector.Conjoin(v)
	}
	return DeriveReadObject(body, NewListFrom(SYMBOLS.fn, argVector, body))
}

func isTerminatingMacro(r rune) bool {
	switch r {
	case '"', ';', '@', '^', '`', '~', '(', ')', '[', ']', '{', '}', '\\', '%':
		return true
	default:
		return false
	}
}

func genSym(prefix string, postfix string) Symbol {
	GENSYM++
	return MakeSymbol(fmt.Sprintf("%s%d%s", prefix, GENSYM, postfix))
}

func generateSymbol(prefix string) Symbol {
	return genSym(prefix, "#")
}

func registerArg(index int) Symbol {
	if s, ok := ARGS[index]; ok {
		return s
	}
	ARGS[index] = generateSymbol("p__")
	return ARGS[index]
}

func readArgSymbol(reader *Reader) Object {
	r := reader.Peek()
	if unicode.IsSpace(r) || isTerminatingMacro(r) {
		return MakeReadObject(reader, registerArg(1))
	}
	obj := Read(reader)
	if obj.Equals(SYMBOLS.amp) {
		return MakeReadObject(reader, registerArg(-1))
	}
	switch n := obj.(type) {
	case Int:
		return MakeReadObject(reader, registerArg(n.I))
	default:
		panic(MakeReadError(reader, "Arg literal must be %, %& or %integer"))
	}
}

func isSelfEvaluating(obj Object) bool {
	if obj == EmptyList {
		return true
	}
	switch obj.(type) {
	case Bool, Double, Int, Char, Keyword, String:
		return true
	default:
		return false
	}
}

func isCall(obj Object, name Symbol) bool {
	switch seq := obj.(type) {
	case Seq:
		return seq.First().Equals(name)
	default:
		return false
	}
}

func syntaxQuoteSeq(seq Seq, env map[*string]Symbol, reader *Reader) Seq {
	res := make([]Object, 0)
	for iter := iter(seq); iter.HasNext(); {
		obj := iter.Next()
		if isCall(obj, SYMBOLS.unquoteSplicing) {
			res = append(res, (obj).(Seq).Rest().First())
		} else {
			q := makeSyntaxQuote(obj, env, reader)
			res = append(res, DeriveReadObject(q, NewListFrom(SYMBOLS.list, q)))
		}
	}
	return &ArraySeq{arr: res}
}

func syntaxQuoteColl(seq Seq, env map[*string]Symbol, reader *Reader, ctor Symbol, info *ObjectInfo) Object {
	q := syntaxQuoteSeq(seq, env, reader)
	concat := q.Cons(SYMBOLS.concat)
	seqList := NewListFrom(SYMBOLS.seq, concat)
	var res Object = seqList
	if ctor != SYMBOLS.emptySymbol {
		res = NewListFrom(ctor, seqList).Cons(SYMBOLS.apply)
	}
	return res.WithInfo(info)
}

func makeSyntaxQuote(obj Object, env map[*string]Symbol, reader *Reader) Object {
	if isSelfEvaluating(obj) {
		return obj
	}
	if IsSpecialSymbol(obj) {
		return makeQuote(obj, SYMBOLS.quote)
	}
	info := obj.GetInfo()
	switch s := obj.(type) {
	case Symbol:
		str := *s.name
		if r, _ := utf8.DecodeLastRuneInString(str); r == '#' && s.ns == nil {
			sym, ok := env[s.name]
			if !ok {
				sym = generateSymbol(str[:len(str)-1] + "__")
				env[s.name] = sym
			}
			obj = DeriveReadObject(obj, sym)
		} else {
			obj = DeriveReadObject(obj, GLOBAL_ENV.ResolveSymbol(s))
		}
		return makeQuote(obj, SYMBOLS.quote)
	case Seq:
		if isCall(obj, SYMBOLS.unquote) {
			return Second(s)
		}
		if isCall(obj, SYMBOLS.unquoteSplicing) {
			panic(MakeReadError(reader, "Splice not in list"))
		}
		return syntaxQuoteColl(s, env, reader, SYMBOLS.emptySymbol, info)
	case *Vector:
		return syntaxQuoteColl(s.Seq(), env, reader, SYMBOLS.vector, info)
	case *ArrayMap:
		return syntaxQuoteColl(ArraySeqFromArrayMap(s), env, reader, SYMBOLS.hashMap, info)
	case *MapSet:
		return syntaxQuoteColl(s.Seq(), env, reader, SYMBOLS.hashSet, info)
	default:
		return obj
	}
}

func filename(f *string) string {
	if f != nil {
		return *f
	}
	return "<file>"
}

func handleNoReaderError(reader *Reader, s Symbol) Object {
	if LINTER_MODE {
		if DIALECT != EDN {
			printReadWarning(reader, "No reader function for tag "+s.ToString(false))
		}
		return Read(reader)
	}
	panic(MakeReadError(reader, "No reader function for tag "+s.ToString(false)))
}

func readTagged(reader *Reader) Object {
	obj := Read(reader)
	switch s := obj.(type) {
	case Symbol:
		readersVar, ok := GLOBAL_ENV.CoreNamespace.mappings[SYMBOLS.defaultDataReaders.name]
		if !ok {
			return handleNoReaderError(reader, s)
		}
		readersMap, ok := readersVar.Value.(Map)
		if !ok {
			return handleNoReaderError(reader, s)
		}
		ok, readFunc := readersMap.Get(s)
		if !ok {
			return handleNoReaderError(reader, s)
		}
		return AssertVar(readFunc, "").Call([]Object{Read(reader)})
	default:
		panic(MakeReadError(reader, "Reader tag must be a symbol"))
	}
}

func readConditional(reader *Reader) Object {
	if reader.Peek() == '@' {
		// Ignoring splicing for now
		// TODO: implement support for splicing
		reader.Get()
	}
	eatWhitespace(reader)
	r := reader.Get()
	if r != '(' {
		panic(MakeReadError(reader, "Reader conditional body must be a list"))
	}
	cond := readList(reader).(*List)
	if cond.count%2 != 0 {
		if LINTER_MODE {
			printReadWarning(reader, "Reader conditional requires an even number of forms")
		} else {
			panic(MakeReadError(reader, "Reader conditional requires an even number of forms"))
		}
	}
	for cond.count > 0 {
		if ok, _ := GLOBAL_ENV.Features.Get(cond.first); ok {
			return Second(cond)
		}
		cond = cond.rest.rest
	}
	return Read(reader)
}

func readDispatch(reader *Reader) Object {
	r := reader.Get()
	switch r {
	case '"':
		return readRegex(reader)
	case '\'':
		popPos()
		nextObj := Read(reader)
		return DeriveReadObject(nextObj, NewListFrom(DeriveReadObject(nextObj, SYMBOLS._var), nextObj))
	case '^':
		popPos()
		return readWithMeta(reader)
	case '{':
		return readSet(reader)
	case '(':
		popPos()
		reader.Unget()
		ARGS = make(map[int]Symbol)
		fn := Read(reader)
		res := makeFnForm(ARGS, fn)
		ARGS = nil
		return res
	case '?':
		return readConditional(reader)
	}
	popPos()
	reader.Unget()
	return readTagged(reader)
}

func readWithMeta(reader *Reader) Object {
	meta := readMeta(reader)
	nextObj := Read(reader)
	switch v := nextObj.(type) {
	case Meta:
		return DeriveReadObject(nextObj, v.WithMeta(meta))
	default:
		panic(MakeReadError(reader, "Metadata cannot be applied to "+v.ToString(false)))
	}
}

func Read(reader *Reader) Object {
	eatWhitespace(reader)
	r := reader.Get()
	pushPos(reader)
	switch {
	case r == '\\':
		return readCharacter(reader)
	case unicode.IsDigit(r):
		reader.Unget()
		return readNumber(reader)
	case r == '-':
		if unicode.IsDigit(reader.Peek()) {
			reader.Unget()
			return readNumber(reader)
		}
		return readSymbol(reader, '-')
	case r == '%' && ARGS != nil:
		return readArgSymbol(reader)
	case isSymbolInitial(r):
		return readSymbol(reader, r)
	case r == '"':
		return readString(reader)
	case r == '(':
		return readList(reader)
	case r == '[':
		return readVector(reader)
	case r == '{':
		return readMap(reader)
	case r == '/' && isDelimiter(reader.Peek()):
		return MakeReadObject(reader, SYMBOLS.backslash)
	case r == '\'':
		popPos()
		nextObj := Read(reader)
		return makeQuote(nextObj, SYMBOLS.quote)
	case r == '@':
		popPos()
		nextObj := Read(reader)
		return DeriveReadObject(nextObj, NewListFrom(DeriveReadObject(nextObj, SYMBOLS.deref), nextObj))
	case r == '~':
		popPos()
		if reader.Peek() == '@' {
			reader.Get()
			nextObj := Read(reader)
			return makeQuote(nextObj, SYMBOLS.unquoteSplicing)
		}
		nextObj := Read(reader)
		return makeQuote(nextObj, SYMBOLS.unquote)
	case r == '`':
		popPos()
		nextObj := Read(reader)
		return makeSyntaxQuote(nextObj, make(map[*string]Symbol), reader)
	case r == '^':
		popPos()
		return readWithMeta(reader)
	case r == '#':
		return readDispatch(reader)
	case r == EOF:
		panic(MakeReadError(reader, "Unexpected end of file"))
	}
	panic(MakeReadError(reader, fmt.Sprintf("Unexpected %c", r)))
}

func TryRead(reader *Reader) (obj Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case *EvalError:
				err = r.(error)
			case ReadError:
				err = r.(error)
			default:
				panic(r)
			}
		}
	}()
	eatWhitespace(reader)
	if reader.Peek() == EOF {
		return NIL, io.EOF
	}
	return Read(reader), nil
}
