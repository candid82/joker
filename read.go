package main

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

type (
	Equality interface {
		Equals(interface{}) bool
	}
	Object interface {
		Equality
		ToString(escape bool) string
	}
	Char      rune
	Double    float64
	Int       int
	Bool      bool
	Keyword   string
	Symbol    string
	String    string
	Regex     string
	ReadError struct {
		msg    string
		line   int
		column int
	}
	Reader struct {
		scanner        io.RuneScanner
		line           int
		prevLineLength int
		column         int
		isEof          bool
	}
)

const EOF = -1

var (
	ARGS   map[int]Symbol
	GENSYM int
)

func (c Char) ToString(escape bool) string {
	if escape {
		return escapeRune(rune(c))
	}
	return string(c)
}

func (c Char) Equals(other interface{}) bool {
	return c == other
}

func (d Double) ToString(escape bool) string {
	return fmt.Sprintf("%f", float64(d))
}

func (d Double) Equals(other interface{}) bool {
	return d == other
}

func (i Int) ToString(escape bool) string {
	return fmt.Sprintf("%d", int(i))
}

func (i Int) Equals(other interface{}) bool {
	return i == other
}

func (b Bool) ToString(escape bool) string {
	return fmt.Sprintf("%t", bool(b))
}

func (b Bool) Equals(other interface{}) bool {
	return b == other
}

func (k Keyword) ToString(escape bool) string {
	return string(k)
}

func (k Keyword) Equals(other interface{}) bool {
	return k == other
}

func (rx Regex) ToString(escape bool) string {
	if escape {
		return "#" + escapeString(string(rx))
	}
	return "#" + string(rx)
}

func (rx Regex) Equals(other interface{}) bool {
	return rx == other
}

func (s Symbol) ToString(escape bool) string {
	return string(s)
}

func (s Symbol) Equals(other interface{}) bool {
	return s == other
}

func (s String) ToString(escape bool) string {
	if escape {
		return escapeString(string(s))
	}
	return string(s)
}

func (s String) Equals(other interface{}) bool {
	return s == other
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

func NewReader(scanner io.RuneScanner) *Reader {
	return &Reader{line: 1, scanner: scanner}
}

func (reader *Reader) Get() rune {
	r, _, err := reader.scanner.ReadRune()
	switch {
	case err == io.EOF:
		reader.isEof = true
		return EOF
	case err != nil:
		panic(err)
	case r == '\n':
		reader.line++
		reader.prevLineLength = reader.column
		reader.column = 0
		return r
	default:
		reader.column++
		return r
	}
}

func (reader *Reader) Unget() {
	if reader.isEof {
		return
	}
	if err := reader.scanner.UnreadRune(); err != nil {
		panic(err)
	}
	if reader.column == 0 {
		reader.line--
		reader.column = reader.prevLineLength
	} else {
		reader.column--
	}
}

func (reader *Reader) Peek() rune {
	r := reader.Get()
	reader.Unget()
	return r
}

func (err ReadError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: %s", err.line, err.column, err.msg)
}

func isDelimiter(r rune) bool {
	switch r {
	case '(', ')', '[', ']', '{', '}', '"', ';', EOF:
		return true
	}
	return unicode.IsSpace(r)
}

func eatString(reader *Reader, str string) error {
	for _, sr := range str {
		if r := reader.Get(); r != sr {
			return MakeReadError(reader, fmt.Sprintf("Unexpected character %U", r))
		}
	}
	return nil
}

func peekExpectedDelimiter(reader *Reader) error {
	r := reader.Peek()
	if !isDelimiter(r) {
		return MakeReadError(reader, "Character not followed by delimiter")
	}
	return nil
}

func readSpecialCharacter(reader *Reader, ending string, r rune) (Object, error) {
	if err := eatString(reader, ending); err != nil {
		return nil, err
	}
	if err := peekExpectedDelimiter(reader); err != nil {
		return nil, err
	}
	return Char(r), nil
}

func eatWhitespace(reader *Reader) {
	r := reader.Get()
	for r != EOF {
		if unicode.IsSpace(r) {
			r = reader.Get()
			continue
		}
		if r == ';' {
			for r != '\n' && r != EOF {
				r = reader.Get()
			}
			r = reader.Get()
			continue
		}
		reader.Unget()
		break
	}
}

func readCharacter(reader *Reader) (Object, error) {
	r := reader.Get()
	if r == EOF {
		return nil, MakeReadError(reader, "Incomplete character literal")
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
	}
	if err := peekExpectedDelimiter(reader); err != nil {
		return nil, err
	}
	return Char(r), nil
}

func readNumber(reader *Reader, sign int) (Object, error) {
	n, fraction, isDouble := 0, 0.0, false
	d := reader.Get()
	for unicode.IsDigit(d) {
		n = n*10 + int(d-'0')
		d = reader.Get()
	}
	if d == '.' {
		isDouble = true
		weight := 10.0
		d = reader.Get()
		for unicode.IsDigit(d) {
			fraction += float64(d-'0') / weight
			weight *= 10
			d = reader.Get()
		}
	}
	if !isDelimiter(d) {
		return nil, MakeReadError(reader, "Number not followed by delimiter")
	}
	reader.Unget()
	if isDouble {
		return Double(float64(sign) * (float64(n) + fraction)), nil
	}
	return Int(sign * n), nil
}

func isSymbolInitial(r rune) bool {
	switch r {
	case '*', '+', '!', '-', '_', '?', ':', '=', '<', '>', '&', '.', '%':
		return true
	}
	return unicode.IsLetter(r)
}

func isSymbolRune(r rune) bool {
	return isSymbolInitial(r) || unicode.IsDigit(r) || r == '#' || r == '/'
}

func readSymbol(reader *Reader, first rune) (Object, error) {
	var b bytes.Buffer
	b.WriteRune(first)
	var lastAdded rune
	r := reader.Get()
	for isSymbolRune(r) {
		if r == ':' {
			if b.Len() > 1 && lastAdded == ':' {
				return nil, MakeReadError(reader, "Invalid use of ':' in symbol name")
			}
		}
		b.WriteRune(r)
		lastAdded = r
		r = reader.Get()
	}
	if lastAdded == ':' || lastAdded == '/' {
		return nil, MakeReadError(reader, fmt.Sprintf("Invalid use of %c in symbol name", lastAdded))
	}
	reader.Unget()
	str := b.String()
	switch {
	case str == "nil":
		return nil, nil
	case str == "true":
		return Bool(true), nil
	case str == "false":
		return Bool(false), nil
	case first == ':':
		return Keyword(str), nil
	default:
		return Symbol(str), nil
	}
}

func readString(reader *Reader, isRegex bool) (Object, error) {
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
			}
		}
		if r == EOF {
			return nil, MakeReadError(reader, "Non-terminated string literal")
		}
		b.WriteRune(r)
		r = reader.Get()
	}
	if isRegex {
		return Regex(b.String()), nil
	}
	return String(b.String()), nil
}

func readList(reader *Reader) (Object, error) {
	s := make([]Object, 0, 10)
	eatWhitespace(reader)
	r := reader.Peek()
	for r != ')' {
		obj, err := Read(reader)
		if err != nil {
			return nil, err
		}
		s = append(s, obj)
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	list := EmptyList
	for i := len(s) - 1; i >= 0; i-- {
		list = list.Conj(s[i])
	}
	return list, nil
}

func readVector(reader *Reader) (Object, error) {
	result := EmptyVector
	eatWhitespace(reader)
	r := reader.Peek()
	for r != ']' {
		obj, err := Read(reader)
		if err != nil {
			return nil, err
		}
		result = result.conj(obj)
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return result, nil
}

func readMap(reader *Reader) (Object, error) {
	m := EmptyArrayMap()
	eatWhitespace(reader)
	r := reader.Peek()
	for r != '}' {
		key, err := Read(reader)
		if err != nil {
			return nil, err
		}
		value, err := Read(reader)
		if err != nil {
			return nil, err
		}
		if !m.Add(key, value) {
			return nil, MakeReadError(reader, "Duplicate key "+key.ToString(false))
		}
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return m, nil
}

func readSet(reader *Reader) (Object, error) {
	set := EmptySet()
	eatWhitespace(reader)
	r := reader.Peek()
	for r != '}' {
		obj, err := Read(reader)
		if err != nil {
			return nil, err
		}
		if !set.Add(obj) {
			return nil, MakeReadError(reader, "Duplicate key "+obj.ToString(false))
		}
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return set, nil
}

func makeQuote(obj Object, quote Symbol) Object {
	return EmptyList.Cons(obj).Cons(quote)
}

func readMeta(reader *Reader) (*ArrayMap, error) {
	obj, err := Read(reader)
	if err != nil {
		return nil, err
	}
	switch m := obj.(type) {
	case *ArrayMap:
		return m, nil
	case String, Symbol:
		return &ArrayMap{arr: []Object{Keyword(":tag"), obj}}, nil
	case Keyword:
		return &ArrayMap{arr: []Object{obj, Bool(true)}}, nil
	default:
		return nil, MakeReadError(reader, "Metadata must be Symbol, Keyword, String or Map")
	}
}

func makeWithMeta(obj Object, meta *ArrayMap) Object {
	return EmptyList.Cons(meta).Cons(obj).Cons(Symbol("with-meta"))
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
			args[i] = makeSymbol("p")
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
		a[len(args)-1] = Symbol("&")
		a = append(a, v)
	}
	argVector := EmptyVector
	for _, v := range a {
		argVector = argVector.conj(v)
	}
	return EmptyList.Cons(body).Cons(argVector).Cons(Symbol("fn"))
}

func isTerminatingMacro(r rune) bool {
	switch r {
	case '"', ';', '@', '^', '`', '~', '(', ')', '[', ']', '{', '}', '\\', '%':
		return true
	default:
		return false
	}
}

func makeSymbol(prefix string) Symbol {
	GENSYM++
	return Symbol(fmt.Sprintf("%s__%d#", prefix, GENSYM))
}

func registerArg(index int) Symbol {
	if s, ok := ARGS[index]; ok {
		return s
	}
	ARGS[index] = makeSymbol("p")
	return ARGS[index]
}

func readArgSymbol(reader *Reader) (Object, error) {
	r := reader.Peek()
	if unicode.IsSpace(r) || isTerminatingMacro(r) {
		return registerArg(1), nil
	}
	obj, err := Read(reader)
	if err != nil {
		return nil, err
	}
	if obj.Equals(Symbol("&")) {
		return registerArg(-1), nil
	}
	switch n := obj.(type) {
	case Int:
		return registerArg(int(n)), nil
	default:
		return nil, MakeReadError(reader, "Arg literal must be %, %& or %integer")
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

func syntaxQuoteSeq(seq Seq, env map[Symbol]Symbol, reader *Reader) (Seq, error) {
	res := make([]Object, 0)
	for iter := iter(seq); iter.HasNext(); {
		obj := iter.Next()
		if isCall(obj, Symbol("unquote-splicing")) {
			res = append(res, obj.(Seq).Rest().First())
		} else {
			q, err := makeSyntaxQuote(obj, env, reader)
			if err != nil {
				return nil, err
			}
			res = append(res, NewListFrom(Symbol("list"), q))
		}
	}
	return &ArraySeq{arr: res}, nil
}

func syntaxQuoteColl(seq Seq, env map[Symbol]Symbol, reader *Reader, ctor Symbol) (Object, error) {
	q, err := syntaxQuoteSeq(seq, env, reader)
	if err != nil {
		return nil, err
	}
	concat := q.Cons(Symbol("concat"))
	seqList := NewListFrom(Symbol("seq"), concat)
	if ctor == Symbol("") {
		return seqList, nil
	}
	return NewListFrom(ctor, seqList).Cons(Symbol("apply")), nil
}

func makeSyntaxQuote(obj Object, env map[Symbol]Symbol, reader *Reader) (Object, error) {
	if isSelfEvaluating(obj) {
		return obj, nil
	}
	switch s := obj.(type) {
	case Symbol:
		str := string(s)
		if r, _ := utf8.DecodeLastRuneInString(str); r == '#' {
			sym, ok := env[s]
			if !ok {
				sym = makeSymbol(str[:len(str)-1])
				env[s] = sym
			}
			obj = sym
		}
		return makeQuote(obj, Symbol("quote")), nil
	case Seq:
		if isCall(obj, Symbol("unquote")) {
			return Second(s), nil
		}
		if isCall(obj, Symbol("unquote-splicing")) {
			return nil, MakeReadError(reader, "Splice not in list")
		}
		return syntaxQuoteColl(s, env, reader, Symbol(""))
	case *Vector:
		return syntaxQuoteColl(s.Seq(), env, reader, Symbol("vector"))
	case *ArrayMap:
		return syntaxQuoteColl(ArraySeqFromArrayMap(s), env, reader, Symbol("hash-map"))
	case *Set:
		return syntaxQuoteColl(s.Seq(), env, reader, Symbol("hash-set"))
	default:
		return obj, nil
	}
}

func Read(reader *Reader) (Object, error) {
	eatWhitespace(reader)
	r := reader.Get()
	switch {
	case r == '\\':
		return readCharacter(reader)
	case unicode.IsDigit(r):
		reader.Unget()
		return readNumber(reader, 1)
	case r == '-':
		if unicode.IsDigit(reader.Peek()) {
			return readNumber(reader, -1)
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
	case r == '#' && reader.Peek() == '"':
		reader.Get()
		return readString(reader, true)
	case r == '#' && reader.Peek() == '{':
		reader.Get()
		return readSet(reader)
	case r == '/' && isDelimiter(reader.Peek()):
		return Symbol("/"), nil
	case r == '\'':
		nextObj, err := Read(reader)
		if err != nil {
			return nil, err
		}
		return makeQuote(nextObj, Symbol("quote")), nil
	case r == '@':
		nextObj, err := Read(reader)
		if err != nil {
			return nil, err
		}
		return NewListFrom(Symbol("deref"), nextObj), nil
	case r == '~':
		if reader.Peek() == '@' {
			reader.Get()
			nextObj, err := Read(reader)
			if err != nil {
				return nil, err
			}
			return makeQuote(nextObj, Symbol("unquote-splicing")), nil
		}
		nextObj, err := Read(reader)
		if err != nil {
			return nil, err
		}
		return makeQuote(nextObj, Symbol("unquote")), nil
	case r == '`':
		nextObj, err := Read(reader)
		if err != nil {
			return nil, err
		}
		return makeSyntaxQuote(nextObj, make(map[Symbol]Symbol), reader)
	case r == '^':
		meta, err := readMeta(reader)
		if err != nil {
			return nil, err
		}
		nextObj, err := Read(reader)
		if err != nil {
			return nil, err
		}
		return makeWithMeta(nextObj, meta), nil
	case r == '#' && reader.Peek() == '_':
		reader.Get()
		Read(reader)
		return Read(reader)
	case r == '#' && reader.Peek() == '(':
		ARGS = make(map[int]Symbol)
		fn, err := Read(reader)
		if err != nil {
			return nil, err
		}
		res := makeFnForm(ARGS, fn)
		ARGS = nil
		return res, nil
	}
	return nil, MakeReadError(reader, fmt.Sprintf("Unexpected %c", r))
}

func TryRead(reader *Reader) (Object, error) {
	eatWhitespace(reader)
	if reader.Peek() == EOF {
		return nil, io.EOF
	}
	return Read(reader)
}
