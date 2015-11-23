package main

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
)

type (
	Object    interface{}
	Keyword   string
	Symbol    string
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
	return r, nil
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
	return r, nil
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
		return float64(sign) * (float64(n) + fraction), nil
	}
	return sign * n, nil
}

func isSymbolInitial(r rune) bool {
	switch r {
	case '*', '+', '!', '-', '_', '?', ':', '=', '<', '>', '&':
		return true
	}
	return unicode.IsLetter(r)
}

func isSymbolRune(r rune) bool {
	return isSymbolInitial(r) || unicode.IsDigit(r) || r == '#'
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
	if lastAdded == ':' {
		return nil, MakeReadError(reader, "Invalid use of ':' in symbol name")
	}
	reader.Unget()
	str := b.String()
	switch {
	case str == "nil":
		return nil, nil
	case str == "true":
		return true, nil
	case str == "false":
		return false, nil
	case first == ':':
		return Keyword(str), nil
	default:
		return Symbol(str), nil
	}
}

func readString(reader *Reader) (Object, error) {
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
	return b.String(), nil
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
	case isSymbolInitial(r):
		return readSymbol(reader, r)
	case r == '"':
		return readString(reader)
	}
	return nil, MakeReadError(reader, fmt.Sprintf("Unexpected %v", r))
}

func TryRead(reader *Reader) (Object, error) {
	eatWhitespace(reader)
	if reader.Peek() == EOF {
		return nil, io.EOF
	}
	return Read(reader)
}
