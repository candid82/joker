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
var PROBLEM_COUNT = 0
var DIALECT Dialect
var LINTER_CONFIG *Var

var (
	ARGS   map[int]Symbol
	GENSYM int
)

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

func isWhitespace(r rune) bool {
	return unicode.IsSpace(r) || r == ','
}

func eatWhitespace(reader *Reader) {
	r := reader.Get()
	for r != EOF {
		if isWhitespace(r) {
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
	return MakeReadObject(reader, ratioOrInt(&rat))
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
		if nonDigits > 2 || nonDigits > 1 && str[0] != '-' && str[0] != '+' {
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
	return unicode.IsLetter(r) || r > 255
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
	case str == "":
		panic(MakeReadError(reader, "Invalid keyword: :"))
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

func readUnicodeCharacterInString(reader *Reader, initial rune, length, base int, exactLength bool) rune {
	n := initial
	var b bytes.Buffer
	for i := 0; i < length && n != '"'; i++ {
		b.WriteRune(n)
		n = reader.Get()
	}
	reader.Unget()
	str := b.String()
	if exactLength && len(str) != length {
		panic(MakeReadError(reader, fmt.Sprintf("Invalid character length: %d, should be: %d", len(str), length)))
	}
	i, err := strconv.ParseInt(str, base, 32)
	if err != nil {
		panic(MakeReadError(reader, "Invalid unicode code: "+str))
	}
	return rune(i)
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
				n := reader.Get()
				r = readUnicodeCharacterInString(reader, n, 4, 16, true)
			default:
				if unicode.IsDigit(r) {
					r = readUnicodeCharacterInString(reader, r, 3, 8, false)
				} else {
					panic(MakeReadError(reader, "Unsupported escape character: \\"+string(r)))
				}
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
		obj, multi := Read(reader)
		if multi {
			v := obj.(*Vector)
			for i := 0; i < v.Count(); i++ {
				s = append(s, v.at(i))
			}
		} else {
			s = append(s, obj)
		}
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
	res := EmptyVector
	eatWhitespace(reader)
	r := reader.Peek()
	for r != ']' {
		obj, multi := Read(reader)
		if multi {
			v := obj.(*Vector)
			for i := 0; i < v.Count(); i++ {
				res = res.Conjoin(v.at(i))
			}
		} else {
			res = res.Conjoin(obj)
		}
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return MakeReadObject(reader, res)
}

func resolveKey(key Object, nsname string) Object {
	if nsname == "" {
		return key
	}
	switch key := key.(type) {
	case Keyword:
		if key.ns == nil {
			return DeriveReadObject(key, MakeKeyword(nsname+"/"+key.Name()))
		}
		if key.Namespace() == "_" {
			return DeriveReadObject(key, MakeKeyword(key.Name()))
		}
	case Symbol:
		if key.ns == nil {
			return DeriveReadObject(key, MakeSymbol(nsname+"/"+key.Name()))
		}
		if key.Namespace() == "_" {
			return DeriveReadObject(key, MakeSymbol(key.Name()))
		}
	}
	return key
}

func readMap(reader *Reader) Object {
	return readMapWithNamespace(reader, "")
}

func readMapWithNamespace(reader *Reader, nsname string) Object {
	eatWhitespace(reader)
	r := reader.Peek()
	objs := []Object{}
	for r != '}' {
		obj, multi := Read(reader)
		if !multi {
			objs = append(objs, obj)
		} else {
			v := obj.(*Vector)
			for i := 0; i < v.Count(); i++ {
				objs = append(objs, v.at(i))
			}
		}
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	if len(objs)%2 != 0 {
		panic(MakeReadError(reader, "Map literal must contain an even number of forms"))
	}
	if len(objs) > HASHMAP_THRESHOLD {
		hashMap := NewHashMap()
		for i := 0; i < len(objs); i += 2 {
			key := resolveKey(objs[i], nsname)
			if hashMap.containsKey(key) {
				panic(MakeReadError(reader, "Duplicate key "+key.ToString(false)))
			}
			hashMap = hashMap.Assoc(key, objs[i+1]).(*HashMap)
		}
		return MakeReadObject(reader, hashMap)
	}
	m := EmptyArrayMap()
	for i := 0; i < len(objs); i += 2 {
		key := resolveKey(objs[i], nsname)
		if !m.Add(key, objs[i+1]) {
			panic(MakeReadError(reader, "Duplicate key "+key.ToString(false)))
		}
	}
	return MakeReadObject(reader, m)
}

func readSet(reader *Reader) Object {
	set := EmptySet()
	eatWhitespace(reader)
	r := reader.Peek()
	for r != '}' {
		obj, multi := Read(reader)
		if !multi {
			if !set.Add(obj) {
				panic(MakeReadError(reader, "Duplicate set element "+obj.ToString(false)))
			}
		} else {
			v := obj.(*Vector)
			for i := 0; i < v.Count(); i++ {
				if !set.Add(v.at(i)) {
					panic(MakeReadError(reader, "Duplicate set element "+v.at(i).ToString(false)))
				}
			}
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
	obj := readFirst(reader)
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
	obj := readFirst(reader)
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
		return readFirst(reader)
	}
	panic(MakeReadError(reader, "No reader function for tag "+s.ToString(false)))
}

func readTagged(reader *Reader) Object {
	obj := readFirst(reader)
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
		return AssertVar(readFunc, "").Call([]Object{readFirst(reader)})
	default:
		panic(MakeReadError(reader, "Reader tag must be a symbol"))
	}
}

func readConditional(reader *Reader) (Object, bool) {
	isSplicing := false
	if reader.Peek() == '@' {
		reader.Get()
		isSplicing = true
	}
	eatWhitespace(reader)
	r := reader.Get()
	if r != '(' {
		panic(MakeReadError(reader, "Reader conditional body must be a list"))
	}
	cond := readList(reader).(*List)
	if cond.count%2 != 0 {
		if LINTER_MODE {
			printReadError(reader, "Reader conditional requires an even number of forms")
		} else {
			panic(MakeReadError(reader, "Reader conditional requires an even number of forms"))
		}
	}
	for cond.count > 0 {
		if ok, _ := GLOBAL_ENV.Features.Get(cond.first); ok {
			v := Second(cond)
			if isSplicing {
				s, ok := v.(Seqable)
				if !ok {
					msg := "Spliced form in reader conditional must be Seqable, got " + v.GetType().ToString(false)
					if LINTER_MODE {
						printReadError(reader, msg)
						return EmptyVector, true
					} else {
						panic(MakeReadError(reader, msg))
					}
				}
				return NewVectorFromSeq(s.Seq()), true
			}
			return v, false
		}
		cond = cond.rest.rest
	}
	return EmptyVector, true
}

func readNamespacedMap(reader *Reader) Object {
	auto := reader.Get() == ':'
	if !auto {
		reader.Unget()
	}
	var sym Object
	r := reader.Get()
	if isWhitespace(r) {
		if !auto {
			reader.Unget()
			panic(MakeReadError(reader, "Namespaced map must specify a namespace"))
		}
		for isWhitespace(r) {
			r = reader.Get()
		}
		if r != '{' {
			reader.Unget()
			panic(MakeReadError(reader, "Namespaced map must specify a namespace"))
		}
	} else if r != '{' {
		reader.Unget()
		sym, _ = Read(reader)
		r = reader.Get()
		for isWhitespace(r) {
			r = reader.Get()
		}
	}
	if r != '{' {
		panic(MakeReadError(reader, "Namespaced map must specify a map"))
	}
	var nsname string
	if auto {
		if sym == nil {
			nsname = GLOBAL_ENV.CurrentNamespace().Name.Name()
		} else {
			sym, ok := sym.(Symbol)
			if !ok || sym.ns != nil {
				panic(MakeReadError(reader, "Namespaced map must specify a valid namespace: "+sym.ToString(false)))
			}
			ns := GLOBAL_ENV.CurrentNamespace().aliases[sym.name]
			if ns == nil {
				ns = GLOBAL_ENV.Namespaces[sym.name]
			}
			if ns == nil {
				panic(MakeReadError(reader, "Unknown auto-resolved namespace alias: "+sym.ToString(false)))
			}
			ns.isUsed = true
			nsname = ns.Name.Name()
		}
	} else {
		if sym == nil {
			panic(MakeReadError(reader, "Namespaced map must specify a valid namespace"))
		}
		sym, ok := sym.(Symbol)
		if !ok || sym.ns != nil {
			panic(MakeReadError(reader, "Namespaced map must specify a valid namespace: "+sym.ToString(false)))
		}
		nsname = sym.Name()
	}
	return readMapWithNamespace(reader, nsname)
}

func readDispatch(reader *Reader) (Object, bool) {
	r := reader.Get()
	switch r {
	case '"':
		return readRegex(reader), false
	case '\'':
		popPos()
		nextObj := readFirst(reader)
		return DeriveReadObject(nextObj, NewListFrom(DeriveReadObject(nextObj, SYMBOLS._var), nextObj)), false
	case '^':
		popPos()
		return readWithMeta(reader), false
	case '{':
		return readSet(reader), false
	case '(':
		popPos()
		reader.Unget()
		ARGS = make(map[int]Symbol)
		fn := readFirst(reader)
		res := makeFnForm(ARGS, fn)
		ARGS = nil
		return res, false
	case '?':
		return readConditional(reader)
	case ':':
		return readNamespacedMap(reader), false
	}
	popPos()
	reader.Unget()
	return readTagged(reader), false
}

func readWithMeta(reader *Reader) Object {
	meta := readMeta(reader)
	nextObj := readFirst(reader)
	switch v := nextObj.(type) {
	case Meta:
		return DeriveReadObject(nextObj, v.WithMeta(meta))
	default:
		panic(MakeReadError(reader, "Metadata cannot be applied to "+v.ToString(false)))
	}
}

func readFirst(reader *Reader) Object {
	obj, multi := Read(reader)
	if !multi {
		return obj
	}
	v := obj.(*Vector)
	if v.Count() == 0 {
		return readFirst(reader)
	}
	return v.at(0)
}

func Read(reader *Reader) (Object, bool) {
	eatWhitespace(reader)
	r := reader.Get()
	pushPos(reader)
	switch {
	case r == '\\':
		return readCharacter(reader), false
	case unicode.IsDigit(r):
		reader.Unget()
		return readNumber(reader), false
	case r == '-' || r == '+':
		if unicode.IsDigit(reader.Peek()) {
			reader.Unget()
			return readNumber(reader), false
		}
		return readSymbol(reader, r), false
	case r == '%' && ARGS != nil:
		return readArgSymbol(reader), false
	case isSymbolInitial(r):
		return readSymbol(reader, r), false
	case r == '"':
		return readString(reader), false
	case r == '(':
		return readList(reader), false
	case r == '[':
		return readVector(reader), false
	case r == '{':
		return readMap(reader), false
	case r == '/' && isDelimiter(reader.Peek()):
		return MakeReadObject(reader, SYMBOLS.backslash), false
	case r == '\'':
		popPos()
		nextObj := readFirst(reader)
		return makeQuote(nextObj, SYMBOLS.quote), false
	case r == '@':
		popPos()
		nextObj := readFirst(reader)
		return DeriveReadObject(nextObj, NewListFrom(DeriveReadObject(nextObj, SYMBOLS.deref), nextObj)), false
	case r == '~':
		popPos()
		if reader.Peek() == '@' {
			reader.Get()
			nextObj := readFirst(reader)
			return makeQuote(nextObj, SYMBOLS.unquoteSplicing), false
		}
		nextObj := readFirst(reader)
		return makeQuote(nextObj, SYMBOLS.unquote), false
	case r == '`':
		popPos()
		nextObj := readFirst(reader)
		return makeSyntaxQuote(nextObj, make(map[*string]Symbol), reader), false
	case r == '^':
		popPos()
		return readWithMeta(reader), false
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
			PROBLEM_COUNT++
			err = r.(error)
		}
	}()
	eatWhitespace(reader)
	if reader.Peek() == EOF {
		return NIL, io.EOF
	}
	for {
		obj, multi := Read(reader)
		if !multi {
			return obj, nil
		}
		if obj.(*Vector).Count() > 0 {
			PROBLEM_COUNT++
			return NIL, MakeReadError(reader, "Reader conditional splicing not allowed at the top level.")
		}
	}
}
