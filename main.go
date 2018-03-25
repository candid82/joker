package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	. "github.com/candid82/joker/core"
	_ "github.com/candid82/joker/std/base64"
	_ "github.com/candid82/joker/std/html"
	_ "github.com/candid82/joker/std/http"
	_ "github.com/candid82/joker/std/json"
	_ "github.com/candid82/joker/std/math"
	_ "github.com/candid82/joker/std/os"
	_ "github.com/candid82/joker/std/string"
	_ "github.com/candid82/joker/std/time"
	_ "github.com/candid82/joker/std/yaml"
	"github.com/chzyer/readline"
)

type (
	ReplContext struct {
		first  *Var
		second *Var
		third  *Var
		exc    *Var
	}
)

func NewReplContext(env *Env) *ReplContext {
	first, _ := env.Resolve(MakeSymbol("joker.core/*1"))
	second, _ := env.Resolve(MakeSymbol("joker.core/*2"))
	third, _ := env.Resolve(MakeSymbol("joker.core/*3"))
	exc, _ := env.Resolve(MakeSymbol("joker.core/*e"))
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

func processFile(filename string, phase Phase) error {
	var reader *Reader
	if filename == "--" {
		reader = NewReader(bufio.NewReader(os.Stdin), "<stdin>")
		filename = ""
	} else {
		var err error
		reader, err = NewReaderFromFile(filename)
		if err != nil {
			return err
		}
	}
	return ProcessReader(reader, filename, phase)
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

func makeDialectKeyword(dialect Dialect) Keyword {
	switch dialect {
	case EDN:
		return MakeKeyword("clj")
	case CLJ:
		return MakeKeyword("clj")
	case CLJS:
		return MakeKeyword("cljs")
	default:
		return MakeKeyword("joker ")
	}
}

func configureLinterMode(dialect Dialect, filename string, workingDir string) {
	LINTER_MODE = true
	DIALECT = dialect
	lm, _ := GLOBAL_ENV.Resolve(MakeSymbol("joker.core/*linter-mode*"))
	lm.Value = Bool{B: true}
	GLOBAL_ENV.Features = GLOBAL_ENV.Features.Disjoin(MakeKeyword("joker")).Conj(makeDialectKeyword(dialect)).(Set)
	ProcessLinterData(dialect)
	ProcessLinterFiles(dialect, filename, workingDir)
}

func detectDialect(filename string) Dialect {
	switch {
	case strings.HasSuffix(filename, ".edn"):
		return EDN
	case strings.HasSuffix(filename, ".cljs"):
		return CLJS
	case strings.HasSuffix(filename, ".joke"):
		return JOKER
	}
	return CLJ
}

func lintFile(filename string, dialect Dialect, workingDir string) {
	phase := PARSE
	if dialect == EDN {
		phase = READ
	}
	ReadConfig(filename, workingDir)
	configureLinterMode(dialect, filename, workingDir)
	if processFile(filename, phase) == nil {
		WarnOnUnusedNamespaces()
		WarnOnUnusedVars()
	}
}

func dialectFromArg(arg string) Dialect {
	switch strings.ToLower(arg) {
	case "clj":
		return CLJ
	case "cljs":
		return CLJS
	case "joker":
		return JOKER
	case "edn":
		return EDN
	}
	return UNKNOWN
}

func main() {
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.CoreNamespace)
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
	workingDir := ""
	phase := EVAL
	lint := false
	dialect := UNKNOWN
	expr := ""
	length := len(os.Args) - 1
	for i := 1; i < length; i++ {
		switch os.Args[i] {
		case "--read":
			phase = READ
		case "--parse":
			phase = PARSE
		case "--working-dir":
			if i < length-1 {
				workingDir = os.Args[i+1]
			}
		case "--lint":
			lint = true
		case "--lintclj":
			lint = true
			dialect = CLJ
		case "--lintcljs":
			lint = true
			dialect = CLJS
		case "--lintjoker":
			lint = true
			dialect = JOKER
		case "--lintedn":
			lint = true
			dialect = EDN
		case "--dialect":
			if i < length-1 {
				dialect = dialectFromArg(os.Args[i+1])
			}
		case "-e":
			if i < length {
				expr = os.Args[i+1]
			}
		case "--hashmap-threshold":
			if i < length {
				i, err := strconv.Atoi(os.Args[i+1])
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: ", err)
					return
				}
				if i < 0 {
					HASHMAP_THRESHOLD = math.MaxInt64
				} else {
					HASHMAP_THRESHOLD = i
				}
			}
		}
	}
	filename := os.Args[length]
	if lint {
		if dialect == UNKNOWN {
			dialect = detectDialect(filename)
		}
		lintFile(filename, dialect, workingDir)
		return
	}
	if phase == EVAL {
		if expr == "" {
			// First argument is a filename, subsequent arguments are script arguments.
			processFile(os.Args[1], phase)
		} else {
			reader := NewReader(strings.NewReader(expr), "<expr>")
			ProcessReader(reader, "", phase)
		}
	} else {
		processFile(filename, phase)
	}
}
