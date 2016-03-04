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

var LINTER_MODE bool = false

func processFile(filename string, phase Phase) {
	parseContext := &ParseContext{globalEnv: GLOBAL_ENV}
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
		expr, err := TryParse(obj, parseContext)
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
	GLOBAL_ENV.namespaces[MakeSymbol("user")].ReferAll(GLOBAL_ENV.namespaces[MakeSymbol("gclojure.core")])
	parseContext := &ParseContext{globalEnv: GLOBAL_ENV}
	reader := NewReader(bufio.NewReader(os.Stdin))
	for {
		fmt.Print(GLOBAL_ENV.currentNamespace.name.ToString(false) + "=> ")
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
		expr, err := TryParse(obj, parseContext)
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
			LINTER_MODE = true
			processFile(os.Args[2], parsePhase(os.Args[1]))
		} else {
			processFile(os.Args[1], EVAL)
		}
	} else {
		repl(EVAL)
	}
}
