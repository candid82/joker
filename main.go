package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type (
	Phase int
)

const (
	READ Phase = iota
	PARSE
	EVAL
)

func processFile(filename string, phase Phase) {
	var reader *Reader
	if filename == "--" {
		reader = NewReader(bufio.NewReader(os.Stdin))
	} else {
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: ", err)
			return
		}
		reader = NewReader(bufio.NewReader(f))
	}
	for {
		obj, err := TryRead(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if phase == READ {
			continue
		}
		expr, err := TryParse(obj)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if phase == PARSE {
			continue
		}
		_, err = TryEval(expr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
}

func skipRestOfLine(reader *Reader) {
	for {
		switch reader.Get() {
		case EOF, '\n':
			return
		}
	}
}

func repl(phase Phase) {
	fmt.Println("Welcome to gclojure. Use ctrl-c to exit.")
	reader := NewReader(bufio.NewReader(os.Stdin))
	for {
		fmt.Print("> ")
		obj, err := TryRead(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			skipRestOfLine(reader)
			continue
		}
		if phase == READ {
			fmt.Println(obj.ToString(true))
			continue
		}
		expr, err := TryParse(obj)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if phase == PARSE {
			fmt.Println(expr)
			continue
		}
		res, err := TryEval(expr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Println(res.ToString(true))
	}
}

func parsePhase(s string) Phase {
	switch s {
	case "--read":
		return READ
	case "--parse":
		return PARSE
	default:
		return EVAL
	}
}

func main() {
	if len(os.Args) > 1 {
		if len(os.Args) > 2 {
			processFile(os.Args[2], parsePhase(os.Args[1]))
		} else {
			processFile(os.Args[1], EVAL)
		}
	} else {
		repl(EVAL)
	}
}
