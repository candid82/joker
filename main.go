package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	_ "github.com/candid82/joker/base64"
	. "github.com/candid82/joker/core"
	_ "github.com/candid82/joker/json"
	_ "github.com/candid82/joker/os"
	_ "github.com/candid82/joker/string"
	"gopkg.in/readline.v1"
)

type (
	ReplContext struct {
		first  *Var
		second *Var
		third  *Var
		exc    *Var
	}
)

const VERSION = "v0.6.1"

func NewReplContext(env *Env) *ReplContext {
	first, _ := env.Resolve(MakeSymbol("core/*1"))
	second, _ := env.Resolve(MakeSymbol("core/*2"))
	third, _ := env.Resolve(MakeSymbol("core/*3"))
	exc, _ := env.Resolve(MakeSymbol("core/*e"))
	first.Value = NIL
	second.Value = NIL
	third.Value = NIL
	exc.Value = NIL
	return &ReplContext{
		first:  first,
		second: second,
		third:  third,
		exc:    exc,
	}
}

func (ctx *ReplContext) PushValue(obj Object) {
	ctx.third.Value = ctx.second.Value
	ctx.second.Value = ctx.first.Value
	ctx.first.Value = obj
}

func (ctx *ReplContext) PushException(exc Object) {
	ctx.exc.Value = exc
}

func processFile(filename string, phase Phase) {
	var reader *Reader
	if filename == "--" {
		reader = NewReader(bufio.NewReader(os.Stdin), "<stdin>")
		filename = ""
	} else {
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: ", err)
			return
		}
		reader = NewReader(bufio.NewReader(f), filename)
	}
	ProcessReader(reader, filename, phase)
}

func skipRestOfLine(reader *Reader) {
	for {
		switch reader.Get() {
		case EOF, '\n':
			return
		}
	}
}

func processReplCommand(reader *Reader, phase Phase, parseContext *ParseContext, replContext *ReplContext) (exit bool) {

	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case *ParseError:
				replContext.PushException(r)
				fmt.Fprintln(os.Stderr, r)
			case *EvalError:
				replContext.PushException(r)
				fmt.Fprintln(os.Stderr, r)
			case Error:
				replContext.PushException(r)
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
	replContext.PushValue(res)
	fmt.Println(res.ToString(true))
	return false
}

func repl(phase Phase) {
	fmt.Printf("Welcome to joker %s. Use ctrl-c to exit.\n", VERSION)
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	replContext := NewReplContext(parseContext.GlobalEnv)

	rl, err := readline.New("")
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer rl.Close()

	reader := NewReader(NewLineRuneReader(rl), "<repl>")

	for {
		rl.SetPrompt(GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		if processReplCommand(reader, phase, parseContext, replContext) {
			return
		}
	}
}

func configureLinterMode(dialect Dialect) {
	LINTER_MODE = true
	DIALECT = dialect
	lm, _ := GLOBAL_ENV.Resolve(MakeSymbol("core/*linter-mode*"))
	lm.Value = Bool{B: true}
	ProcessLinterData(dialect)
}

func main() {
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.FindNamespace(MakeSymbol("core")))
	if len(os.Args) == 1 {
		repl(EVAL)
		return
	}
	if len(os.Args) == 2 {
		if os.Args[1] == "-v" || os.Args[1] == "--version" {
			println(VERSION)
			return
		}
		processFile(os.Args[1], EVAL)
		return
	}
	switch os.Args[1] {
	case "--read":
		processFile(os.Args[2], READ)
	case "--parse":
		processFile(os.Args[2], PARSE)
	case "--lint":
		fallthrough
	case "--lintclj":
		configureLinterMode(CLJ)
		processFile(os.Args[2], PARSE)
	case "--lintcljs":
		configureLinterMode(CLJS)
		processFile(os.Args[2], PARSE)
	case "--lintjoker":
		configureLinterMode(JOKER)
		processFile(os.Args[2], PARSE)
	case "--lintedn":
		configureLinterMode(EDN)
		processFile(os.Args[2], READ)
	default:
		processFile(os.Args[1], EVAL)
	}
}
