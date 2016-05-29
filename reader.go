package main

import (
	"io"
)

type (
	Reader struct {
		runeReader     io.RuneReader
		rw             *RuneWindow
		line           int
		prevLineLength int
		column         int
		isEof          bool
		rewind         int
	}
)

func NewReader(runeReader io.RuneReader) *Reader {
	return &Reader{line: 1, runeReader: runeReader, rw: &RuneWindow{}, rewind: -1}
}

func (reader *Reader) Get() rune {
	if reader.isEof {
		return EOF
	}
	if reader.rewind > -1 {
		r := top(reader.rw, reader.rewind)
		reader.rewind--
		if r == '\n' {
			reader.line++
			reader.prevLineLength = reader.column
			reader.column = 0
		} else {
			reader.column++
		}
		return r
	}
	r, _, err := reader.runeReader.ReadRune()
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
		add(reader.rw, r)
		return r
	default:
		reader.column++
		add(reader.rw, r)
		return r
	}
}

func (reader *Reader) Unget() {
	if reader.isEof {
		return
	}
	reader.rewind++
	if reader.column == 0 {
		reader.line--
		reader.column = reader.prevLineLength
	} else {
		reader.column--
	}
}

func (reader *Reader) Peek() rune {
	if reader.isEof {
		return EOF
	}
	r := reader.Get()
	reader.Unget()
	return r
}
