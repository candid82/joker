package core

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/rand"
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

var (
	LINTER_MODE   bool = false
	FORMAT_MODE   bool = false
	PROBLEM_COUNT      = 0
	DIALECT       Dialect
	LINTER_CONFIG *Var
	SUPPRESS_READ bool = false
)

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

func (err ReadError) Message() Object {
	return MakeString(err.msg)
}

func (err ReadError) Error() string {
	return fmt.Sprintf("%s:%d:%d: Read error: %s", filename(err.filename), err.line, err.column, err.msg)
}

func isDelimiter(r rune) bool {
	switch r {
	case '(', ')', '[', ']', '{', '}', '"', ';', EOF, '\\':
		return true
	}
	return isWhitespace(r)
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

func isJavaSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r': // Listed here purely for speed of common cases
		return true
	case 0xa0 /*&nbsp;*/, 0x85 /*NEL*/, 0x2007 /*&numsp;*/, 0x202f /*narrow non-break space*/ :
		return false
	case 0x1c /*FS*/, 0x1d /*GS*/, 0x1e /*RS*/, 0x1f /*US*/ :
		return true
	default:
		if r > unicode.MaxLatin1 && unicode.In(r, unicode.Zl, unicode.Zp, unicode.Zs) {
			return true
		}
	}
	return unicode.IsSpace(r)
}

func isWhitespace(r rune) bool {
	return isJavaSpace(r) || r == ','
}

func readComment(reader *Reader) Object {
	var b bytes.Buffer
	r := reader.Peek()
	for r != '\n' && r != EOF {
		b.WriteRune(r)
		reader.Get()
		r = reader.Peek()
	}
	return MakeReadObject(reader, Comment{C: b.String()})
}

func eatWhitespace(reader *Reader) {
	r := reader.Get()
	for r != EOF {
		if FORMAT_MODE && r == ',' {
			reader.Unget()
			break
		}
		if isWhitespace(r) {
			r = reader.Get()
			continue
		}
		if (r == ';' || (r == '#' && reader.Peek() == '!')) && !FORMAT_MODE {
			for r != '\n' && r != EOF {
				r = reader.Get()
			}
			r = reader.Get()
			continue
		}
		if r == '#' && reader.Peek() == '_' && !FORMAT_MODE {
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
			return readUnicodeCharacter(reader, 3, 8)
		}
	}
	peekExpectedDelimiter(reader)
	return MakeReadObject(reader, Char{Ch: r})
}

func invalidNumberError(reader *Reader, str string) error {
	return MakeReadError(reader, fmt.Sprintf("Invalid number: %s", str))
}

func scanBigInt(orig, str string, base int, reader *Reader) Object {
	var bi big.Int
	if _, ok := bi.SetString(str, base); !ok {
		panic(invalidNumberError(reader, str))
	}
	res := BigInt{b: bi, Original: orig}
	return MakeReadObject(reader, &res)
}

func scanRatio(str string, reader *Reader) Object {
	var rat big.Rat
	if _, ok := rat.SetString(str); !ok {
		panic(invalidNumberError(reader, str))
	}
	return MakeReadObject(reader, ratioOrIntWithOriginal(str, &rat))
}

func scanBigFloat(orig, str string, reader *Reader) Object {
	var bf big.Float
	if _, ok := bf.SetPrec(256).SetString(str); !ok {
		panic(invalidNumberError(reader, str))
	}
	res := BigFloat{b: bf, Original: orig}
	return MakeReadObject(reader, &res)
}

func scanInt(orig, str string, base int, reader *Reader) Object {
	i, e := strconv.ParseInt(str, base, 0)
	if e != nil {
		return scanBigInt(orig, str, base, reader)
	}
	// TODO: 32-bit issue
	return MakeReadObject(reader, Int{I: int(i), Original: orig})
}

func scanFloat(str string, reader *Reader) Object {
	dbl, e := strconv.ParseFloat(str, 64)
	if e != nil {
		panic(invalidNumberError(reader, str))
	}
	return MakeReadObject(reader, Double{D: dbl, Original: str})
}

func readNumber(reader *Reader) Object {
	var b bytes.Buffer
	isDouble, isHex, isExp, isRatio, baseLen, nonDigits := false, false, false, false, 0, 0
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
			if baseLen == 0 {
				baseLen = b.Len()
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
	if baseLen != 0 {
		baseInt, err := strconv.ParseInt(str[0:baseLen], 0, 0)
		if err != nil {
			panic(invalidNumberError(reader, str))
		}
		negative := false
		if baseInt < 0 {
			baseInt = -baseInt
			negative = true
		}
		if baseInt < 2 || baseInt > 36 {
			panic(invalidNumberError(reader, str))
		}
		var number string
		if negative {
			number = "-" + str[baseLen+1:]
		} else {
			number = str[baseLen+1:] // Avoid an expensive catenation in positive/zero case
		}
		return scanInt(str, number, int(baseInt), reader)
	}
	if isRatio {
		if nonDigits > 2 || nonDigits > 1 && str[0] != '-' && str[0] != '+' {
			panic(invalidNumberError(reader, str))
		}
		return scanRatio(str, reader)
	}
	if last == 'N' {
		return scanBigInt(str, str[:b.Len()-1], 0, reader)
	}
	if last == 'M' {
		return scanBigFloat(str, str[:b.Len()-1], reader)
	}
	if isDouble || (!isHex && isExp) {
		return scanFloat(str, reader)
	}
	return scanInt(str, str, 0, reader)
}

/* Returns whether the rune may be a non-initial character in a symbol
/* name. */
func isSymbolRune(r rune) bool {
	switch r {
	case '"', ';', '@', '^', '`', '~', '(', ')', '[', ']', '{', '}', '\\', ',', ' ', '\t', '\n', '\r', EOF:
		// Whitespace listed above (' ', '\t', '\n', '\r') purely for speed of common cases

		return false
	}
	return !isJavaSpace(r)
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
			if FORMAT_MODE {
				return MakeReadObject(reader, MakeKeyword(str))
			}
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
			ns.isGloballyUsed = true
			return MakeReadObject(reader, MakeKeyword(*ns.Name.name+"/"+*sym.name))
		}
		return MakeReadObject(reader, MakeKeyword(str))
	case str == "nil":
		return MakeReadObject(reader, NIL)
	case str == "true":
		return MakeReadObject(reader, Boolean{B: true})
	case str == "false":
		return MakeReadObject(reader, Boolean{B: false})
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
	s := b.String()
	regex, err := regexp.Compile(s)
	if err != nil {
		if LINTER_MODE {
			return MakeReadObject(reader, &Regex{})
		}
		if FORMAT_MODE {
			res := MakeReadObject(reader, MakeString(s))
			addPrefix(res, "#")
			return res
		}
		panic(MakeReadError(reader, "Invalid regex: "+err.Error()))
	}
	return MakeReadObject(reader, &Regex{R: regex})
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
			if FORMAT_MODE {
				b.WriteRune('\\')
			} else {
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
		}
		if r == EOF {
			panic(MakeReadError(reader, "Non-terminated string literal"))
		}
		b.WriteRune(r)
		r = reader.Get()
	}
	return MakeReadObject(reader, String{S: b.String()})
}

func readMulti(reader *Reader, previouslyRead []Object) (Object, []Object) {
	if len(previouslyRead) == 0 {
		obj, multi := Read(reader)
		if multi {
			v := obj.(*Vector)
			for i := 0; i < v.Count(); i++ {
				previouslyRead = append(previouslyRead, v.at(i))
			}
		} else {
			return obj, previouslyRead
		}
	}
	obj := previouslyRead[len(previouslyRead)-1]
	previouslyRead = previouslyRead[0 : len(previouslyRead)-1]
	return obj, previouslyRead
}

func readError(reader *Reader, msg string) {
	if LINTER_MODE {
		printReadError(reader, msg)
	} else {
		panic(MakeReadError(reader, msg))
	}
}

func readCondList(reader *Reader) Object {
	previousSuppressRead := SUPPRESS_READ
	defer func() {
		SUPPRESS_READ = previousSuppressRead
	}()

	var forms []Object
	eatWhitespace(reader)
	r := reader.Peek()
	var res Object = nil
	for r != ')' || len(forms) != 0 {
		if res == nil {
			feature, forms := readMulti(reader, forms)
			if feature.Equals(KEYWORDS.none) || feature.Equals(KEYWORDS.else_) {
				panic(MakeReadError(reader, "Feature name "+feature.ToString(false)+" is reserved"))
			}
			if !IsKeyword(feature) {
				panic(MakeReadError(reader, "Feature should be a keyword"))
			}
			eatWhitespace(reader)
			if len(forms) == 0 && reader.Peek() == ')' {
				reader.Get()
				readError(reader, "Reader conditional requires an even number of forms")
				return feature
			}
			if ok, _ := GLOBAL_ENV.Features.Get(feature); ok {
				res, forms = readMulti(reader, forms)
			} else {
				SUPPRESS_READ = true
				_, forms = readMulti(reader, forms)
				SUPPRESS_READ = false
			}
		} else {
			SUPPRESS_READ = true
			_, forms = readMulti(reader, forms)
			SUPPRESS_READ = false
		}
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	return res
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
	res := EmptyVector()
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

func appendMapElement(objs []Object, obj Object) []Object {
	objs = append(objs, obj)
	if FORMAT_MODE {
		if isComment(obj) {
			// Add surrogate object to always have even number of elements in the map.
			// Use rand to avoid duplicate keys.
			objs = append(objs, MakeDouble(rand.Float64()))
		}
	}
	return objs
}

func readMapWithNamespace(reader *Reader, nsname string) Object {
	eatWhitespace(reader)
	r := reader.Peek()
	objs := []Object{}
	for r != '}' {
		obj, multi := Read(reader)
		if !multi {
			objs = appendMapElement(objs, obj)
		} else {
			v := obj.(*Vector)
			for i := 0; i < v.Count(); i++ {
				objs = appendMapElement(objs, v.at(i))
			}
		}
		eatWhitespace(reader)
		r = reader.Peek()
	}
	reader.Get()
	if len(objs)%2 != 0 {
		panic(MakeReadError(reader, "Map literal must contain an even number of forms"))
	}
	if int64(len(objs)) >= HASHMAP_THRESHOLD {
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
		return &ArrayMap{arr: []Object{obj, DeriveReadObject(obj, Boolean{B: true})}}
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
	argVector := EmptyVector()
	for _, v := range a {
		argVector = argVector.Conjoin(v)
	}
	if LINTER_MODE {
		if meta, ok := body.(Meta); ok {
			m := EmptyArrayMap().Plus(MakeKeyword("skip-redundant-do"), Boolean{B: true})
			body = meta.WithMeta(m)
		}
	}
	return DeriveReadObject(body, NewListFrom(MakeSymbol("joker.core/fn"), argVector, body))
}

func isTerminatingMacro(r rune) bool {
	switch r {
	case '"', ';', '@', '^', '`', '~', '(', ')', '[', ']', '{', '}', '\\':
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
	if isWhitespace(r) || isTerminatingMacro(r) {
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
	case Boolean, Double, Int, Char, Keyword, String:
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
	if SUPPRESS_READ {
		return readFirst(reader)
	}
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
	if FORMAT_MODE {
		addPrefix(obj, "#")
		return obj
	}
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
		return EnsureObjectIsVar(readFunc, "").Call([]Object{readFirst(reader)})
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
	if FORMAT_MODE {
		cond := readList(reader).(*List)
		if isSplicing {
			addPrefix(cond, "#?@")
		} else {
			addPrefix(cond, "#?")
		}
		return cond, false
	}
	v := readCondList(reader)
	if v == nil {
		return EmptyVector(), true
	}
	if isSplicing {
		s, ok := v.(Seqable)
		if !ok {
			readError(reader, "Spliced form in reader conditional must be Seqable, got "+v.GetType().ToString(false))
			return EmptyVector(), true
		}
		return DeriveReadObject(v, NewVectorFromSeq(s.Seq())), true
	}
	return v, false
}

func namespacedMapPrefix(auto bool, nsSym Object) string {
	res := "#:"
	if auto {
		res += ":"
	}
	if nsSym != nil {
		res += nsSym.ToString(false)
	}
	return res
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
	if FORMAT_MODE {
		obj := readMap(reader)
		addPrefix(obj, namespacedMapPrefix(auto, sym))
		return obj
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
			ns.isGloballyUsed = true
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

var specials = map[string]float64{
	"Inf":  math.Inf(1),
	"-Inf": math.Inf(-1),
	"NaN":  math.NaN(),
}

func readSymbolicValue(reader *Reader) Object {
	obj := readFirst(reader)
	switch o := obj.(type) {
	case Symbol:
		if v, found := specials[o.ToString(false)]; found {
			return Double{D: v}
		}
		panic(MakeReadError(reader, "Unknown symbolic value: ##"+o.ToString(false)))
	default:
		panic(MakeReadError(reader, "Invalid token: ##"+o.ToString(false)))
	}
}

func readDispatch(reader *Reader) (Object, bool) {
	r := reader.Get()
	switch r {
	case '"':
		return readRegex(reader), false
	case '\'':
		popPos()
		nextObj := readFirst(reader)
		if FORMAT_MODE {
			addPrefix(nextObj, "#'")
			return nextObj, false
		}
		return DeriveReadObject(nextObj, NewListFrom(DeriveReadObject(nextObj, SYMBOLS._var), nextObj)), false
	case '_':
		// Only possible in FORMAT mode, otherwise
		// eatWhitespaces eats #_
		popPos()
		nextObj := readFirst(reader)
		addPrefix(nextObj, "#_")
		return nextObj, false
	case '^':
		popPos()
		if FORMAT_MODE {
			nextObj := readFirst(reader)
			addPrefix(nextObj, "#^")
			return nextObj, false
		}
		return readWithMeta(reader), false
	case '{':
		return readSet(reader), false
	case '(':
		popPos()
		reader.Unget()
		if FORMAT_MODE {
			nextObj := readFirst(reader)
			addPrefix(nextObj, "#")
			return nextObj, false
		}
		ARGS = make(map[int]Symbol)
		fn := readFirst(reader)
		res := makeFnForm(ARGS, fn)
		ARGS = nil
		return res, false
	case '?':
		return readConditional(reader)
	case ':':
		return readNamespacedMap(reader), false
	case '#':
		return readSymbolicValue(reader), false
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

func addPrefix(obj Object, prefix string) {
	obj.GetInfo().prefix = prefix + obj.GetInfo().prefix
}

func Read(reader *Reader) (Object, bool) {
	eatWhitespace(reader)
	r := reader.Get()
	pushPos(reader)
	// This is only possible in format mode, otherwise
	// eatWhitespace eats comments.
	if r == ',' {
		return MakeReadObject(reader, Comment{C: ","}), false
	}
	if r == ';' || (r == '#' && reader.Peek() == '!') {
		reader.Unget()
		return readComment(reader), false
	}

	switch {
	case r == '\\':
		return readCharacter(reader), false
	case unicode.IsDigit(r):
		reader.Unget()
		return readNumber(reader), false
	case r == '.':
		if DIALECT == CLJS && unicode.IsDigit(reader.Peek()) {
			reader.Unget()
			return readNumber(reader), false
		}
		return readSymbol(reader, r), false
	case r == '-' || r == '+':
		if unicode.IsDigit(reader.Peek()) {
			reader.Unget()
			return readNumber(reader), false
		}
		return readSymbol(reader, r), false
	case r == '%' && ARGS != nil:
		if FORMAT_MODE {
			return readSymbol(reader, r), false
		}
		return readArgSymbol(reader), false
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
		if FORMAT_MODE {
			addPrefix(nextObj, "'")
			return nextObj, false
		}
		return makeQuote(nextObj, SYMBOLS.quote), false
	case r == '@':
		popPos()
		nextObj := readFirst(reader)
		if FORMAT_MODE {
			addPrefix(nextObj, "@")
			return nextObj, false
		}
		return DeriveReadObject(nextObj, NewListFrom(DeriveReadObject(nextObj, SYMBOLS.deref), nextObj)), false
	case r == '~':
		popPos()
		if reader.Peek() == '@' {
			reader.Get()
			nextObj := readFirst(reader)
			if FORMAT_MODE {
				addPrefix(nextObj, "~@")
				return nextObj, false
			}
			return makeQuote(nextObj, SYMBOLS.unquoteSplicing), false
		}
		nextObj := readFirst(reader)
		if FORMAT_MODE {
			addPrefix(nextObj, "~")
			return nextObj, false
		}
		return makeQuote(nextObj, SYMBOLS.unquote), false
	case r == '`':
		popPos()
		nextObj := readFirst(reader)
		if FORMAT_MODE {
			addPrefix(nextObj, "`")
			return nextObj, false
		}
		return makeSyntaxQuote(nextObj, make(map[*string]Symbol), reader), false
	case r == '^':
		popPos()
		if FORMAT_MODE {
			nextObj := readFirst(reader)
			addPrefix(nextObj, "^")
			return nextObj, false
		}
		return readWithMeta(reader), false
	case r == '#':
		return readDispatch(reader)
	case r == EOF:
		panic(MakeReadError(reader, "Unexpected end of file"))
	default:
		return readSymbol(reader, r), false
	}
}

func TryRead(reader *Reader) (obj Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			PROBLEM_COUNT++
			err = r.(error)
		}
	}()
	for {
		eatWhitespace(reader)
		if reader.Peek() == EOF {
			return NIL, io.EOF
		}
		obj, multi := Read(reader)
		if !multi {
			return obj, nil
		}
		// Check for obj's info to distinguish between
		// legitimate empty vector as read from the source
		// and surrogate value that means "no object was read".
		if obj.GetInfo() != nil {
			PROBLEM_COUNT++
			return NIL, MakeReadError(reader, "Reader conditional splicing not allowed at the top level.")
		}
	}
}
