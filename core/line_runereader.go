// +build !plan9

package core

import (
	"io"
	"unicode/utf8"

	"github.com/chzyer/readline"
)

type (
	LineRuneReader struct {
		rl     *readline.Instance
		buffer []rune
		i      int
	}
)

func NewLineRuneReader(rl *readline.Instance) *LineRuneReader {
	return &LineRuneReader{rl: rl}
}

func (lrr *LineRuneReader) ReadRune() (rune, int, error) {
	if lrr.buffer != nil && lrr.i < len(lrr.buffer) {
		r := lrr.buffer[lrr.i]
		lrr.i++
		return r, utf8.RuneLen(r), nil
	}
	line, err := lrr.rl.Readline()
	if err != nil {
		return EOF, 0, io.EOF
	}
	lrr.buffer = make([]rune, 0, len(line)+1)
	for _, r := range line {
		lrr.buffer = append(lrr.buffer, r)
	}
	lrr.buffer = append(lrr.buffer, '\n')
	lrr.i = 1
	return lrr.buffer[0], utf8.RuneLen(lrr.buffer[0]), nil
}
