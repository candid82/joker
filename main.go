package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	. "github.com/candid82/joker/core"
	_ "github.com/candid82/joker/std/base64"
	_ "github.com/candid82/joker/std/json"
	_ "github.com/candid82/joker/std/os"
	_ "github.com/candid82/joker/std/string"
	_ "github.com/candid82/joker/std/time"
	_ "github.com/candid82/joker/std/yaml"
	"github.com/chzyer/readline"
	"github.com/spf13/pflag"
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
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: ", err)
			return err
		}
		reader = NewReader(bufio.NewReader(f), filename)
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

func configureLinterMode(dialect Dialect) {
	LINTER_MODE = true
	DIALECT = dialect
	lm, _ := GLOBAL_ENV.Resolve(MakeSymbol("joker.core/*linter-mode*"))
	lm.Value = Bool{B: true}
	GLOBAL_ENV.Features = GLOBAL_ENV.Features.Disjoin(MakeKeyword("joker")).Conj(makeDialectKeyword(dialect)).(Set)
	ProcessLinterData(dialect)
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
	configureLinterMode(dialect)
	if processFile(filename, phase) == nil {
		WarnOnUnusedNamespaces()
		WarnOnUnusedVars()
	}
}

var versionFlag = pflag.BoolP("version", "v", false, "display version information")
var readFlag = pflag.Bool("read", false, "read the file")
var parseFlag = pflag.Bool("parse", false, "parse the file")
var lintFlag = pflag.Bool("lint", false, "lint the file")

var lintDialect = pflag.String("dialect", "", "dialect to lint as. Valid options are clj, cljs, joker and edn")
var lintWorkingDir = pflag.String("working-dir", "", "override the working directory for the linter")

// these flags are here to support the preexisting `--lint<dialect>` flags:
var lintCljFlag = pflag.Bool("lintclj", false, "lint as clojure")
var lintCljsFlag = pflag.Bool("lintcljs", false, "lint as clojurescript")
var lintJokerFlag = pflag.Bool("lintjoker", false, "lint as joker")
var lintEDNFlag = pflag.Bool("lintedn", false, "lint as edn")

func usage() {
	fmt.Fprintf(os.Stderr, "Joker - %s\n\n", VERSION)
	fmt.Fprintln(os.Stderr, "usage: joker                                   starts a repl")
	fmt.Fprintln(os.Stderr, "   or: joker [arguments] <filename>            execute a script")
	fmt.Fprintln(os.Stderr, "\nArguments:")
	pflag.PrintDefaults()
}

func init() {
	pflag.Usage = usage
	pflag.Parse()

	switch {
	case *lintCljFlag:
		*lintFlag = true
		*lintDialect = "clj"
	case *lintCljsFlag:
		*lintFlag = true
		*lintDialect = "cljs"
	case *lintJokerFlag:
		*lintFlag = true
		*lintDialect = "joker"
	case *lintEDNFlag:
		*lintFlag = true
		*lintDialect = "edn"
	}
}

func main() {
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.CoreNamespace)
	switch {
	case *versionFlag:
		println(VERSION)
	case *readFlag:
		processFile(pflag.Arg(0), READ)
	case *parseFlag:
		processFile(pflag.Arg(0), PARSE)
	case *lintFlag:
		var dialect Dialect
		switch strings.ToLower(*lintDialect) {
		case "clj":
			dialect = CLJ
		case "cljs":
			dialect = CLJS
		case "joker":
			dialect = JOKER
		case "edn":
			dialect = EDN
		default:
			dialect = detectDialect(pflag.Arg(0))
		}
		filename := pflag.Arg(0)
		if filename == "" {
			filename = "--"
		}
		lintFile(filename, dialect, *lintWorkingDir)
	case len(pflag.Args()) > 0:
		processFile(pflag.Arg(0), EVAL)
	default:
		repl(EVAL)
	}
}
