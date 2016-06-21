package main

import (
	"bufio"
	"fmt"
	"gopkg.in/readline.v1"
	"io"
	"os"
	// "runtime"
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

func processReader(reader *Reader, phase Phase) {
	parseContext := &ParseContext{globalEnv: GLOBAL_ENV}
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
	processReader(reader, phase)
}

func skipRestOfLine(reader *Reader) {
	for {
		switch reader.Get() {
		case EOF, '\n':
			return
		}
	}
}

func processReplCommand(reader *Reader, phase Phase, parseContext *ParseContext) (exit bool) {

	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case *ParseError:
				fmt.Fprintln(os.Stderr, r)
			case *EvalError:
				fmt.Fprintln(os.Stderr, r)
			case Error:
				fmt.Fprintln(os.Stderr, r)
			// case *runtime.TypeAssertionError:
			// 	fmt.Fprintln(os.Stderr, r)
			default:
				panic(r)
			}
		}
	}()

	obj, err := TryRead(reader)
	if err == io.EOF {
		return true
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		skipRestOfLine(reader)
		return
	}

	if phase == READ {
		fmt.Println(obj.ToString(true))
		return false
	}

	expr := parse(obj, parseContext)
	if phase == PARSE {
		fmt.Println(expr)
		return false
	}

	res := eval(expr, nil)
	fmt.Println(res.ToString(true))
	return false
}

func repl(phase Phase) {
	fmt.Println("Welcome to gclojure. Use ctrl-c to exit.")
	parseContext := &ParseContext{globalEnv: GLOBAL_ENV}

	rl, err := readline.New("")
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer rl.Close()

	reader := NewReader(NewLineRuneReader(rl))

	for {
		rl.SetPrompt(GLOBAL_ENV.currentNamespace.name.ToString(false) + "=> ")
		if processReplCommand(reader, phase, parseContext) {
			return
		}
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
	GLOBAL_ENV.namespaces[MakeSymbol("user").name].ReferAll(GLOBAL_ENV.namespaces[MakeSymbol("gclojure.core").name])
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
