package main

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type (
	ReadError struct {
		line   int
		column int
		msg    string
	}
	ReadFunc func(reader *Reader) Object
)

const EOF = -1

var (
	ARGS   map[int]Symbol
	GENSYM int
)

func readStub(reader *Reader) Object {
	return Read(reader)
}

var DATA_READERS = map[*string]ReadFunc{}
var NIL = Nil{}

func init() {
	DATA_READERS[MakeSymbol("inst").name] = readStub
	DATA_READERS[MakeSymbol("uuid").name] = readStub
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
		line:   reader.line,
		column: reader.column,
		msg:    msg,
	}
}

func MakeReadObject(reader *Reader, obj Object) Object {
	return obj.WithInfo(&ObjectInfo{Position: Position{line: reader.line, column: reader.column}})
}

func DeriveReadObject(base Object, obj Object) Object {
	baseInfo := base.GetInfo()
	if baseInfo != nil {
		return obj.WithInfo(&ObjectInfo{Position: Position{line: baseInfo.line, column: baseInfo.column}})
	}
	return obj
}

func (err ReadError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: Read error: %s", err.line, err.column, err.msg)
}

func (err ReadError) Type() Symbol {
	return MakeSymbol("ReadError")
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
	return MakeReadObject(reader, Char{ch: r})
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
	return MakeReadObject(reader, Char{ch: rune(i)})
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
	return MakeReadObject(reader, Char{ch: r})
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
	return MakeReadObject(reader, Int{i: int(i)})
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
		return MakeReadObject(reader, Double{d: dbl})
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
		return MakeReadObject(reader, MakeKeyword(str))
	case str == "nil":
		return MakeReadObject(reader, NIL)
	case str == "true":
		return MakeReadObject(reader, Bool{b: true})
	case str == "false":
		return MakeReadObject(reader, Bool{b: false})
	default:
		return MakeReadObject(reader, MakeSymbol(str))
	}
}

func readString(reader *Reader, isRegex bool) Object {
	var b bytes.Buffer
	r := reader.Get()
	for r != '"' {
		if r == '\\' {
			r = reader.Get()
			switch r {
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
			}
		}
		if r == EOF {
			panic(MakeReadError(reader, "Non-terminated string literal"))
		}
		b.WriteRune(r)
		r = reader.Get()
	}
	if isRegex {
		return MakeReadObject(reader, Regex{r: b.String()})
	}
	return MakeReadObject(reader, String{s: b.String()})
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
		result = result.conj(obj)
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return MakeReadObject(reader, result)
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
		return &ArrayMap{arr: []Object{DeriveReadObject(obj, MakeKeyword("tag")), obj}}
	case Keyword:
		return &ArrayMap{arr: []Object{obj, DeriveReadObject(obj, Bool{b: true})}}
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
		a[len(args)-1] = MakeSymbol("&")
		a = append(a, v)
	}
	argVector := EmptyVector
	for _, v := range a {
		argVector = argVector.conj(v)
	}
	return DeriveReadObject(body, NewListFrom(MakeSymbol("fn"), argVector, body))
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
	if obj.Equals(MakeSymbol("&")) {
		return MakeReadObject(reader, registerArg(-1))
	}
	switch n := obj.(type) {
	case Int:
		return MakeReadObject(reader, registerArg(n.i))
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
		if isCall(obj, MakeSymbol("unquote-splicing")) {
			res = append(res, (obj).(Seq).Rest().First())
		} else {
			q := makeSyntaxQuote(obj, env, reader)
			res = append(res, DeriveReadObject(q, NewListFrom(MakeSymbol("list"), q)))
		}
	}
	return &ArraySeq{arr: res}
}

func syntaxQuoteColl(seq Seq, env map[*string]Symbol, reader *Reader, ctor Symbol, info *ObjectInfo) Object {
	q := syntaxQuoteSeq(seq, env, reader)
	concat := q.Cons(MakeSymbol("concat"))
	seqList := NewListFrom(MakeSymbol("seq"), concat)
	var res Object = seqList
	if ctor != MakeSymbol("") {
		res = NewListFrom(ctor, seqList).Cons(MakeSymbol("apply"))
	}
	return res.WithInfo(info)
}

func makeSyntaxQuote(obj Object, env map[*string]Symbol, reader *Reader) Object {
	if isSelfEvaluating(obj) {
		return obj
	}
	info := obj.GetInfo()
	switch s := obj.(type) {
	case Symbol:
		str := *s.name
		if r, _ := utf8.DecodeLastRuneInString(str); r == '#' {
			sym, ok := env[s.name]
			if !ok {
				sym = generateSymbol(str[:len(str)-1] + "__")
				env[s.name] = sym
			}
			obj = DeriveReadObject(obj, sym)
		}
		return makeQuote(obj, MakeSymbol("quote"))
	case Seq:
		if isCall(obj, MakeSymbol("unquote")) {
			return Second(s)
		}
		if isCall(obj, MakeSymbol("unquote-splicing")) {
			panic(MakeReadError(reader, "Splice not in list"))
		}
		return syntaxQuoteColl(s, env, reader, MakeSymbol(""), info)
	case *Vector:
		return syntaxQuoteColl(s.Seq(), env, reader, MakeSymbol("vector"), info)
	case *ArrayMap:
		return syntaxQuoteColl(ArraySeqFromArrayMap(s), env, reader, MakeSymbol("hash-map"), info)
	case *ArraySet:
		return syntaxQuoteColl(s.Seq(), env, reader, MakeSymbol("hash-set"), info)
	default:
		return obj
	}
}

func readTagged(reader *Reader) Object {
	obj := Read(reader)
	switch s := obj.(type) {
	case Symbol:
		readFunc := DATA_READERS[s.name]
		if readFunc == nil {
			fmt.Fprintf(os.Stderr, "stdin:%d:%d: Read warning: No reader function for tag %s\n", reader.line, reader.column, s.ToString(false))
			return Read(reader)
		}
		return readFunc(reader)
	default:
		panic(MakeReadError(reader, "Reader tag must be a symbol"))
	}
}

func readDispatch(reader *Reader) Object {
	r := reader.Get()
	switch r {
	case '"':
		return readString(reader, true)
	case '\'':
		nextObj := Read(reader)
		return DeriveReadObject(nextObj, NewListFrom(DeriveReadObject(nextObj, MakeSymbol("var")), nextObj))
	case '^':
		return readWithMeta(reader)
	case '{':
		return readSet(reader)
	case '(':
		reader.Unget()
		ARGS = make(map[int]Symbol)
		fn := Read(reader)
		res := makeFnForm(ARGS, fn)
		ARGS = nil
		return res
	}
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
		return readString(reader, false)
	case r == '(':
		return readList(reader)
	case r == '[':
		return readVector(reader)
	case r == '{':
		return readMap(reader)
	case r == '/' && isDelimiter(reader.Peek()):
		return MakeReadObject(reader, MakeSymbol("/"))
	case r == '\'':
		nextObj := Read(reader)
		return makeQuote(nextObj, MakeSymbol("quote"))
	case r == '@':
		nextObj := Read(reader)
		return DeriveReadObject(nextObj, NewListFrom(DeriveReadObject(nextObj, MakeSymbol("deref")), nextObj))
	case r == '~':
		if reader.Peek() == '@' {
			reader.Get()
			nextObj := Read(reader)
			return makeQuote(nextObj, MakeSymbol("unquote-splicing"))
		}
		nextObj := Read(reader)
		return makeQuote(nextObj, MakeSymbol("unquote"))
	case r == '`':
		nextObj := Read(reader)
		return makeSyntaxQuote(nextObj, make(map[*string]Symbol), reader)
	case r == '^':
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
			err = r.(ReadError)
		}
	}()
	eatWhitespace(reader)
	if reader.Peek() == EOF {
		return NIL, io.EOF
	}
	return Read(reader), nil
}
