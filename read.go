package main

import (
	"fmt"
	"io"
	"unicode"
)

type (
	Object    interface{}
	ReadError struct {
		msg string
	}
)

func peekRune(s io.RuneScanner) (rune, int, error) {
	r, i, err := s.ReadRune()
	if err == nil {
		s.UnreadRune()
	}
	return r, i, err
}

func (err ReadError) Error() string {
	return err.msg
}

func isDelimiter(r rune) bool {
	switch r {
	case '(', ')', '[', ']', '{', '}', '"', ';':
		return true
	}
	return unicode.IsSpace(r)
}

func eatString(s io.RuneScanner, str string) error {
	for _, sr := range str {
		r, _, err := s.ReadRune()
		if err != nil {
			return err
		}
		if r != sr {
			return ReadError{msg: fmt.Sprintf("Unexpected character %U", r)}
		}
	}
	return nil
}

func peekExpectedDelimiter(s io.RuneScanner) error {
	r, _, err := peekRune(s)
	if err != nil {
		return err
	}
	if !isDelimiter(r) {
		return ReadError{msg: "Character not followed by delimiter"}
	}
	return nil
}

func readSpecialCharacter(s io.RuneScanner, ending string, r rune) (Object, error) {
	err := eatString(s, ending)
	if err != nil {
		return nil, err
	}
	err = peekExpectedDelimiter(s)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func eatWhitespace(s io.RuneScanner) {
	r, _, err := s.ReadRune()
	for err == nil {
		if unicode.IsSpace(r) {
			r, _, err = s.ReadRune()
			continue
		}
		if r == ';' {
			for r != '\n' && err == nil {
				r, _, err = s.ReadRune()
			}
			r, _, err = s.ReadRune()
			continue
		}
		s.UnreadRune()
		break
	}
}

func readCharacter(s io.RuneScanner) (Object, error) {
	r, _, err := s.ReadRune()
	if err != nil {
		return nil, ReadError{msg: "Incomplete character literal"}
	}
	switch r {
	case 's':
		if next, _, err := peekRune(s); err == nil && next == 'p' {
			return readSpecialCharacter(s, "pace", ' ')
		}
	case 'n':
		if next, _, err := peekRune(s); err == nil && next == 'e' {
			return readSpecialCharacter(s, "ewline", '\n')
		}
	case 't':
		if next, _, err := peekRune(s); err == nil && next == 'a' {
			return readSpecialCharacter(s, "ab", '\t')
		}
	case 'f':
		if next, _, err := peekRune(s); err == nil && next == 'o' {
			return readSpecialCharacter(s, "ormfeed", '\f')
		}
	case 'b':
		if next, _, err := peekRune(s); err == nil && next == 'a' {
			return readSpecialCharacter(s, "ackspace", '\b')
		}
	case 'r':
		if next, _, err := peekRune(s); err == nil && next == 'e' {
			return readSpecialCharacter(s, "eturn", '\r')
		}
	}
	if err = peekExpectedDelimiter(s); err != nil {
		return nil, err
	}
	return r, nil
}

func readNumber(s io.RuneScanner) (Object, error) {
	n, sign, fraction, isDouble := 0, 1, 0.0, false
	d, _, err := s.ReadRune()
	if err != nil {
		return nil, err
	}
	if d == '-' {
		sign = -1
	} else {
		s.UnreadRune()
	}
	if d, _, err = s.ReadRune(); err != nil {
		return nil, err
	}
	for unicode.IsDigit(d) {
		n = n*10 + int(d-'0')
		if d, _, err = s.ReadRune(); err != nil {
			return nil, err
		}
	}
	if d == '.' {
		isDouble = true
		weight := 10.0
		if d, _, err = s.ReadRune(); err != nil {
			return nil, err
		}
		for unicode.IsDigit(d) {
			fraction += float64(d-'0') / weight
			weight *= 10
			if d, _, err = s.ReadRune(); err != nil {
				return nil, err
			}
		}
	}
	if !isDelimiter(d) {
		return nil, ReadError{msg: "Number not followed by delimiter"}
	}
	s.UnreadRune()
	if isDouble {
		return float64(sign) * (float64(n) + fraction), nil
	}
	return sign * n, nil
}

func Read(s io.RuneScanner) (Object, error) {
	eatWhitespace(s)
	r, _, err := s.ReadRune()
	if err != nil {
		return nil, err
	}
	switch {
	case r == '\\':
		return readCharacter(s)
	case unicode.IsDigit(r):
		s.UnreadRune()
		return readNumber(s)
	}
	return nil, ReadError{msg: fmt.Sprintf("Unexpected %v", r)}
}
