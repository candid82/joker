//go:build !plan9
// +build !plan9

package core

import (
	"io"
	"strings"
	"unicode/utf8"

	"github.com/candid82/liner"
)

type (
	LineRuneReader struct {
		rl     *liner.State
		buffer []rune
		i      int
		Prompt string
	}
)

func NewLineRuneReader(rl *liner.State) *LineRuneReader {
	return &LineRuneReader{rl: rl}
}

func (lrr *LineRuneReader) ReadRune() (rune, int, error) {
	if lrr.buffer != nil && lrr.i < len(lrr.buffer) {
		r := lrr.buffer[lrr.i]
		lrr.i++
		return r, utf8.RuneLen(r), nil
	}
	line, err := lrr.rl.Prompt(lrr.Prompt)
	if err != nil {
		return EOF, 0, io.EOF
	}
	if strings.TrimSpace(line) != "" {
		lrr.rl.AppendHistory(line)
	}
	lrr.buffer = make([]rune, 0, len(line)+1)
	for _, r := range line {
		lrr.buffer = append(lrr.buffer, r)
	}
	lrr.buffer = append(lrr.buffer, '\n')
	lrr.i = 1
	return lrr.buffer[0], utf8.RuneLen(lrr.buffer[0]), nil
}
