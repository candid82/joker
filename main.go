package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"

	. "github.com/candid82/joker/core"
	_ "github.com/candid82/joker/std/base64"
	_ "github.com/candid82/joker/std/go/net"
	_ "github.com/candid82/joker/std/html"
	_ "github.com/candid82/joker/std/http"
	_ "github.com/candid82/joker/std/json"
	_ "github.com/candid82/joker/std/math"
	_ "github.com/candid82/joker/std/os"
	_ "github.com/candid82/joker/std/string"
	_ "github.com/candid82/joker/std/time"
	_ "github.com/candid82/joker/std/url"
	_ "github.com/candid82/joker/std/yaml"
	"github.com/chzyer/readline"
	"github.com/pkg/profile"
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
	if filename == "-" || filename == "--" {
		if filename == "--" {
			fmt.Fprintln(os.Stderr, "Warning: '--' indicating standard input (stdin) to Joker is deprecated; please use '-' instead")
		}
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
	fmt.Printf("Welcome to joker %s. Use EOF (Ctrl-D) or SIGINT (Ctrl-C) to exit.\n", VERSION)
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	replContext := NewReplContext(parseContext.GlobalEnv)

	var runeReader io.RuneReader
	var rl *readline.Instance
	var err error
	if noReadline {
		runeReader = bufio.NewReader(os.Stdin)
	} else {
		rl, err = readline.New("")
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}
		defer rl.Close()
		runeReader = NewLineRuneReader(rl)
	}

	reader := NewReader(runeReader, "<repl>")

	for {
		if noReadline {
			print(GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		} else {
			rl.SetPrompt(GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		}
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
	ProcessLinterFiles(dialect, filename, workingDir)
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

func usage(out *os.File) {
	fmt.Fprintf(out, "Joker - %s\n\n", VERSION)
	fmt.Fprintln(out, "Usage: joker [args]                                 starts a repl")
	fmt.Fprintln(out, "   or: joker [args] --repl [-- <repl-args>]         starts a repl with args")
	fmt.Fprintln(out, "   or: joker [args] --eval <expr> [-- <expr-args>]  evaluate <expr>, print if non-nil")
	fmt.Fprintln(out, "   or: joker [args] <filename> [<script-args>]      input from file")
	fmt.Fprintln(out, "   or: joker [args] --lint <filename>               lint the code in file")
	fmt.Fprintln(out, "\nNotes:")
	fmt.Fprintln(out, "  -e is a synonym for --eval.")
	fmt.Fprintln(out, "  '-' for <filename> means read from standard input (stdin).")
	fmt.Fprintln(out, "  Evaluating '(println (str *command-line-args*))' prints the arguments")
	fmt.Fprintln(out, "    in <repl-args>, <expr-args>, or <script-args> (TBD).")
	fmt.Fprintln(out, "\nOptions (<args>):")
	fmt.Fprintln(out, "  --help, -h")
	fmt.Fprintln(out, "    Print this help message and exit.")
	fmt.Fprintln(out, "  --version, -v")
	fmt.Fprintln(out, "    Print version number and exit.")
	fmt.Fprintln(out, "  --read")
	fmt.Fprintln(out, "    Read, but do not parse nor evaluate, the input.")
	fmt.Fprintln(out, "  --parse")
	fmt.Fprintln(out, "    Read and parse, but do not evaluate, the input.")
	fmt.Fprintln(out, "  --evaluate")
	fmt.Fprintln(out, "    Read, parse, and evaluate the input (default unless --lint in effect).")
	fmt.Fprintln(out, "  --no-readline")
	fmt.Fprintln(out, "    Disable readline functionality in the repl. Useful if joker is called with rlwrap.")
	fmt.Fprintln(out, "  --working-dir <directory>")
	fmt.Fprintln(out, "    Specify working directory for lint configuration (requires --lint).")
	fmt.Fprintln(out, "  --dialect <dialect>")
	fmt.Fprintln(out, "    Set input dialect (\"clj\", \"cljs\", \"joker\", \"edn\") for linting;")
	fmt.Fprintln(out, "    default is inferred from <filename> suffix, if any.")
	fmt.Fprintln(out, "  --hashmap-threshold <n>")
	fmt.Fprintln(out, "    Set HASHMAP_THRESHOLD accordingly (internal magic of some sort).")
	fmt.Fprintln(out, "  --profiler <type>")
	fmt.Fprintln(out, "    Specify type of profiler to use (default 'runtime/pprof' or 'pkg/profile').")
	fmt.Fprintln(out, "  --cpuprofile <name>")
	fmt.Fprintln(out, "    Write CPU profile to specified file or directory (depending on")
	fmt.Fprintln(out, "    profiler chosen).")
	fmt.Fprintln(out, "  --cpuprofile-rate <rate>")
	fmt.Fprintln(out, "    Specify rate (hz, aka samples per second) for the 'runtime/pprof' CPU")
	fmt.Fprintln(out, "    profiler to use.")
	fmt.Fprintln(out, "  --memprofile <name>")
	fmt.Fprintln(out, "    Write memory profile to specified file.")
	fmt.Fprintln(out, "  --memprofile-rate <rate>")
	fmt.Fprintln(out, "    Specify rate (one sample per <rate>) for the memory profiler to use.")
}

var (
	debug              bool // Hidden option
	helpFlag           bool
	versionFlag        bool
	phase              Phase = EVAL // --read, --parse, --evaluate
	workingDir         string
	lintFlag           bool
	dialect            Dialect = UNKNOWN
	eval               string
	replFlag           bool
	filename           string
	remainingArgs      []string
	profilerType       string = "runtime/pprof"
	cpuProfileName     string
	cpuProfileRate     int
	cpuProfileRateFlag bool
	memProfileName     string
	noReadline         bool
)

func notOption(arg string) bool {
	return arg == "-" || !strings.HasPrefix(arg, "-")
}

func parseArgs(args []string) {
	length := len(args)
	stop := false
	missing := false
	noFileFlag := false
	var i int
	for i = 1; i < length; i++ { // shift
		if debug {
			fmt.Fprintf(os.Stderr, "arg[%d]=%s\n", i, args[i])
		}
		switch args[i] {
		case "--", "-":
			stop = true // "-" is stdin. "--" is stdin for now; later will formally end options processing
		case "--debug":
			debug = true
		case "--help", "-h":
			helpFlag = true
			return // don't bother parsing anything else
		case "--version", "-v":
			versionFlag = true
		case "--read":
			phase = READ
		case "--parse":
			phase = PARSE
		case "--evaluate":
			phase = EVAL
		case "--working-dir":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				workingDir = args[i]
			} else {
				missing = true
			}
		case "--lint":
			lintFlag = true
		case "--lintclj":
			lintFlag = true
			dialect = CLJ
		case "--lintcljs":
			lintFlag = true
			dialect = CLJS
		case "--lintjoker":
			lintFlag = true
			dialect = JOKER
		case "--lintedn":
			lintFlag = true
			dialect = EDN
		case "--dialect":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				dialect = dialectFromArg(args[i])
			} else {
				missing = true
			}
		case "--hashmap-threshold":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				thresh, err := strconv.Atoi(args[i])
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: ", err)
					return
				}
				if thresh < 0 {
					HASHMAP_THRESHOLD = math.MaxInt64
				} else {
					HASHMAP_THRESHOLD = thresh
				}
			} else {
				missing = true
			}
		case "-e", "--eval":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				eval = args[i]
				phase = PRINT_IF_NOT_NIL
				if i < length-1 && args[i+1] == "--" {
					i += 2 // shift 2
					noFileFlag = true
					stop = true
				}
			} else {
				missing = true
			}
		case "--repl":
			replFlag = true
			if i < length-1 && args[i+1] == "--" {
				i += 2 // shift 2
				noFileFlag = true
				stop = true
			}
		case "--no-readline":
			noReadline = true
		case "--profiler":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				profilerType = args[i]
			} else {
				missing = true
			}
		case "--cpuprofile":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				cpuProfileName = args[i]
			} else {
				missing = true
			}
		case "--cpuprofile-rate":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				rate, err := strconv.Atoi(args[i])
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: ", err)
					return
				}
				if rate > 0 {
					cpuProfileRate = rate
					cpuProfileRateFlag = true
				}
			} else {
				missing = true
			}
		case "--memprofile":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				memProfileName = args[i]
			} else {
				missing = true
			}
		case "--memprofile-rate":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				rate, err := strconv.Atoi(args[i])
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: ", err)
					return
				}
				if rate > 0 {
					runtime.MemProfileRate = rate
				}
			} else {
				missing = true
			}
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "Error: Unrecognized option '%s'\n", args[i])
				ExitJoker(2)
			}
			stop = true
		}
		if stop || missing {
			break
		}
	}
	if missing {
		fmt.Fprintf(os.Stderr, "Error: Missing argument for '%s' option\n", args[i])
		ExitJoker(3)
	}
	if i < length && !noFileFlag {
		if debug {
			fmt.Fprintf(os.Stderr, "filename=%s\n", args[i])
		}
		filename = args[i]
		i += 1 // shift
	}
	if i < length {
		if debug {
			fmt.Fprintf(os.Stderr, "remaining=%v\n", args[i:])
		}
		remainingArgs = args[i:]
	}
}

var runningProfile interface {
	Stop()
}

func main() {
	ProcessCoreData()

	SetExitJoker(func(code int) {
		finish()
		os.Exit(code)
	})
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.CoreNamespace)

	if len(os.Args) > 1 && os.Args[1] == "--debug" {
		debug = true
	} // peek to see if it's the first arg

	parseArgs(os.Args)
	GLOBAL_ENV.SetEnvArgs(remainingArgs)

	if debug {
		fmt.Fprintf(os.Stderr, "debug=%v\n", debug)
		fmt.Fprintf(os.Stderr, "helpFlag=%v\n", helpFlag)
		fmt.Fprintf(os.Stderr, "versionFlag=%v\n", versionFlag)
		fmt.Fprintf(os.Stderr, "phase=%v\n", phase)
		fmt.Fprintf(os.Stderr, "lintFlag=%v\n", lintFlag)
		fmt.Fprintf(os.Stderr, "dialect=%v\n", dialect)
		fmt.Fprintf(os.Stderr, "workingDir=%v\n", workingDir)
		fmt.Fprintf(os.Stderr, "HASHMAP_THRESHOLD=%v\n", HASHMAP_THRESHOLD)
		fmt.Fprintf(os.Stderr, "eval=%v\n", eval)
		fmt.Fprintf(os.Stderr, "replFlag=%v\n", replFlag)
		fmt.Fprintf(os.Stderr, "noReadline=%v\n", noReadline)
		fmt.Fprintf(os.Stderr, "filename=%v\n", filename)
		fmt.Fprintf(os.Stderr, "remainingArgs=%v\n", remainingArgs)
	}

	if helpFlag {
		usage(os.Stdout)
		return
	}

	if versionFlag {
		println(VERSION)
		return
	}

	if len(remainingArgs) > 0 {
		if lintFlag {
			fmt.Fprintf(os.Stderr, "Error: Cannot provide arguments to code while linting it.\n")
			ExitJoker(4)
		}
		if phase != EVAL && phase != PRINT_IF_NOT_NIL {
			fmt.Fprintf(os.Stderr, "Error: Cannot provide arguments to code without evaluating it.\n")
			ExitJoker(5)
		}
	}

	/* Set up profiling. */

	if cpuProfileName != "" {
		switch profilerType {
		case "pkg/profile":
			runningProfile = profile.Start(profile.ProfilePath(cpuProfileName))
			defer finish()
		case "runtime/pprof":
			f, err := os.Create(cpuProfileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Could not create CPU profile `%s': %v\n",
					cpuProfileName, err)
				cpuProfileName = ""
				ExitJoker(96)
			}
			if cpuProfileRateFlag {
				runtime.SetCPUProfileRate(cpuProfileRate)
			}
			pprof.StartCPUProfile(f)
			fmt.Fprintf(os.Stderr, "Profiling started at rate=%d. See file `%s'.\n",
				cpuProfileRate, cpuProfileName)
			defer finish()
		default:
			fmt.Fprintf(os.Stderr,
				"Unrecognized profiler: %s\n  Use 'pkg/profile' or 'runtime/pprof'.\n",
				profilerType)
			ExitJoker(96)
		}
	} else if memProfileName != "" {
		defer finish()
	}

	if eval != "" {
		if lintFlag {
			fmt.Fprintf(os.Stderr, "Error: Cannot combine --eval/-e and --lint.\n")
			ExitJoker(6)
		}
		if replFlag {
			fmt.Fprintf(os.Stderr, "Error: Cannot combine --eval/-e and --repl.\n")
			ExitJoker(7)
		}
		if workingDir != "" {
			fmt.Fprintf(os.Stderr, "Error: Cannot combine --eval/-e and --working-dir.\n")
			ExitJoker(8)
		}
		if filename != "" {
			fmt.Fprintf(os.Stderr, "Error: Cannot combine --eval/-e and a <filename> argument.\n")
			ExitJoker(9)
		}
		reader := NewReader(strings.NewReader(eval), "<expr>")
		ProcessReader(reader, "", phase)
		return
	}

	if lintFlag {
		if replFlag {
			fmt.Fprintf(os.Stderr, "Error: Cannot combine --lint and --repl.\n")
			ExitJoker(10)
		}
		if dialect == UNKNOWN {
			dialect = detectDialect(filename)
		}
		lintFile(filename, dialect, workingDir)
		if PROBLEM_COUNT > 0 {
			ExitJoker(1)
		}
		return
	}

	if workingDir != "" {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify --working-dir option when not linting.\n")
		ExitJoker(11)
	}

	if filename != "" {
		processFile(filename, phase)
		return
	}

	repl(phase)
	return
}

func finish() {
	if runningProfile != nil {
		runningProfile.Stop()
		runningProfile = nil
	} else if cpuProfileName != "" {
		pprof.StopCPUProfile()
		fmt.Fprintf(os.Stderr, "Profiling stopped. See file `%s'.\n", cpuProfileName)
		cpuProfileName = ""
	}

	if memProfileName != "" {
		f, err := os.Create(memProfileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not create memory profile `%s': %v\n",
				memProfileName, err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not write memory profile `%s': %v\n",
				memProfileName, err)
		}
		f.Close()
		fmt.Fprintf(os.Stderr, "Memory profile rate=%d written to `%s'.\n",
			runtime.MemProfileRate, memProfileName)
		memProfileName = ""
	}
}
