package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	_ "github.com/candid/joker/base64"
	. "github.com/candid/joker/core"
	_ "github.com/candid/joker/json"
	_ "github.com/candid/joker/os"
	_ "github.com/candid/joker/string"
	"gopkg.in/readline.v1"
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
	ProcessReader(reader, phase)
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

	expr := Parse(obj, parseContext)
	if phase == PARSE {
		fmt.Println(expr)
		return false
	}

	res := Eval(expr, nil)
	fmt.Println(res.ToString(true))
	return false
}

func repl(phase Phase) {
	fmt.Println("Welcome to joker. Use ctrl-c to exit.")
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}

	rl, err := readline.New("")
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer rl.Close()

	reader := NewReader(NewLineRuneReader(rl))

	for {
		rl.SetPrompt(GLOBAL_ENV.CurrentNamespace.Name.ToString(false) + "=> ")
		if processReplCommand(reader, phase, parseContext) {
			return
		}
	}
}

func main() {
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.FindNamespace(MakeSymbol("joker.core")))
	if len(os.Args) == 1 {
		repl(EVAL)
		return
	}
	if len(os.Args) == 2 {
		processFile(os.Args[1], EVAL)
		return
	}
	switch os.Args[1] {
	case "--read":
		LINTER_MODE = true
		processFile(os.Args[2], READ)
	case "--parse":
		LINTER_MODE = true
		processFile(os.Args[2], PARSE)
	default:
		processFile(os.Args[1], EVAL)
	}
}
