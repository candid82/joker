package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"

	. "github.com/candid82/joker/core"
	_ "github.com/candid82/joker/std/base64"
	_ "github.com/candid82/joker/std/bolt"
	_ "github.com/candid82/joker/std/crypto"
	_ "github.com/candid82/joker/std/csv"
	_ "github.com/candid82/joker/std/filepath"
	_ "github.com/candid82/joker/std/hex"
	_ "github.com/candid82/joker/std/html"
	_ "github.com/candid82/joker/std/http"
	_ "github.com/candid82/joker/std/io"
	_ "github.com/candid82/joker/std/json"
	_ "github.com/candid82/joker/std/math"
	_ "github.com/candid82/joker/std/os"
	_ "github.com/candid82/joker/std/strconv"
	_ "github.com/candid82/joker/std/string"
	_ "github.com/candid82/joker/std/time"
	_ "github.com/candid82/joker/std/url"
	_ "github.com/candid82/joker/std/uuid"
	_ "github.com/candid82/joker/std/yaml"
	"github.com/pkg/profile"
)

var dataRead = []rune{}
var saveForRepl = true

type replayable struct {
	reader *Reader
}

func (r *replayable) ReadRune() (ch rune, size int, err error) {
	ch = r.reader.Get()
	if ch == EOF {
		err = io.EOF
		size = 0
	} else {
		dataRead = append(dataRead, ch)
		size = 1
	}
	return
}

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
	if filename == "-" {
		reader = NewReader(bufio.NewReader(Stdin), "<stdin>")
		filename = ""
	} else {
		var err error
		reader, err = NewReaderFromFile(filename)
		if err != nil {
			return err
		}
	}
	if filename != "" {
		f, err := filepath.Abs(filename)
		PanicOnErr(err)
		GLOBAL_ENV.SetMainFilename(f)
	}
	if saveForRepl {
		reader = NewReader(&replayable{reader}, "<replay>")
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
				fmt.Fprintln(Stderr, r)
			case *EvalError:
				replContext.PushException(r)
				fmt.Fprintln(Stderr, r)
			case Error:
				replContext.PushException(r)
				fmt.Fprintln(Stderr, r)
				// case *runtime.TypeAssertionError:
				// 	fmt.Fprintln(Stderr, r)
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
		fmt.Fprintln(Stderr, err)
		skipRestOfLine(reader)
		return
	}

	if kw, yes := obj.(Keyword); yes {
		if kw.ToString(false) == ":repl/quit" {
			ExitJoker(0)
		}
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
	PrintObject(res, Stdout)
	fmt.Fprintln(Stdout, "")
	return false
}

func srepl(port string, phase Phase) {
	ProcessReplData()
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.FindNamespace(MakeSymbol("joker.repl")))
	l, err := net.Listen("tcp", replSocket)
	if err != nil {
		fmt.Fprintf(Stderr, "Cannot start srepl listening on %s: %s\n",
			replSocket, err.Error())
		ExitJoker(12)
	}
	defer l.Close()

	fmt.Printf("Joker repl listening at %s...\n", l.Addr())
	conn, err := l.Accept() // Wait for a single connection
	if err != nil {
		fmt.Fprintf(Stderr, "Cannot start repl accepting on %s: %s\n",
			l.Addr(), err.Error())
		ExitJoker(13)
	}

	oldStdIn := Stdin
	oldStdOut := Stdout
	oldStdErr := Stderr
	oldStdinValue, oldStdoutValue, oldStderrValue := GLOBAL_ENV.StdIO()
	Stdin = conn
	Stdout = conn
	Stderr = conn
	newIn := MakeBufferedReader(conn)
	newOut := MakeIOWriter(conn)
	GLOBAL_ENV.SetStdIO(newIn, newOut, newOut)
	defer func() {
		conn.Close()
		Stdin = oldStdIn
		Stdout = oldStdOut
		Stderr = oldStdErr
		GLOBAL_ENV.SetStdIO(oldStdinValue, oldStdoutValue, oldStderrValue)
	}()

	fmt.Printf("Joker repl accepting client at %s...\n", conn.RemoteAddr())

	runeReader := bufio.NewReader(conn)

	/* The rest of this code comes from repl(), below: */

	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	replContext := NewReplContext(parseContext.GlobalEnv)

	reader := NewReader(runeReader, "<srepl>")

	fmt.Fprintf(Stdout, "Welcome to joker %s, client at %s. Use ':repl/quit', or close the connection, to exit.\n",
		VERSION, conn.RemoteAddr())

	for {
		fmt.Fprint(Stdout, GLOBAL_ENV.CurrentNamespace().Name.ToString(false)+"=> ")
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
		return MakeKeyword("joker")
	}
}

func configureLinterMode(dialect Dialect, filename string, workingDir string) {
	ProcessLinterFiles(dialect, filename, workingDir)
	LINTER_MODE = true
	DIALECT = dialect
	lm, _ := GLOBAL_ENV.Resolve(MakeSymbol("joker.core/*linter-mode*"))
	lm.Value = Boolean{B: true}
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

func matchesDialect(path string, dialect Dialect) bool {
	ext := ".clj"
	switch dialect {
	case CLJS:
		ext = ".cljs"
	case JOKER:
		ext = ".joke"
	case EDN:
		ext = ".edn"
	}
	return strings.HasSuffix(path, ext)
}

func isIgnored(path string) bool {
	for _, r := range WARNINGS.IgnoredFileRegexes {
		m := r.FindStringSubmatchIndex(path)
		if len(m) > 0 {
			if m[1]-m[0] == len(path) {
				return true
			}
		}
	}
	return false
}

func lintDir(dirname string, dialect Dialect, reportGloballyUnused bool) {
	var processErr error
	phase := PARSE
	if dialect == EDN {
		phase = READ
	}
	ns := GLOBAL_ENV.CurrentNamespace()
	ReadConfig("", dirname)
	configureLinterMode(dialect, "", dirname)
	filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintln(Stderr, "Error: ", err)
			return nil
		}
		if !info.IsDir() && matchesDialect(path, dialect) && !isIgnored(path) {
			GLOBAL_ENV.CoreNamespace.Resolve("*loaded-libs*").Value = EmptySet()
			processErr = processFile(path, phase)
			if processErr == nil {
				WarnOnUnusedNamespaces()
				WarnOnUnusedVars()
			}
			ResetUsage()
			GLOBAL_ENV.SetCurrentNamespace(ns)
		}
		return nil
	})
	if processErr == nil && reportGloballyUnused {
		WarnOnGloballyUnusedNamespaces()
		WarnOnGloballyUnusedVars()
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

func usage(out io.Writer) {
	fmt.Fprintf(out, "Joker - %s\n\n", VERSION)
	fmt.Fprintln(out, "Usage: joker [args] [-- <repl-args>]                starts a repl")
	fmt.Fprintln(out, "   or: joker [args] --repl [<socket>] [-- <repl-args>]")
	fmt.Fprintln(out, "                                                    starts a repl (on optional network socket)")
	fmt.Fprintln(out, "   or: joker [args] --eval <expr> [-- <expr-args>]  evaluate <expr>, print if non-nil")
	fmt.Fprintln(out, "   or: joker [args] [--file] <filename> [<script-args>]")
	fmt.Fprintln(out, "                                                    input from file")
	fmt.Fprintln(out, "   or: joker [args] --lint <filename>               lint the code in file")
	fmt.Fprintln(out, "\nNotes:")
	fmt.Fprintln(out, "  -e is a synonym for --eval.")
	fmt.Fprintln(out, "  '-' for <filename> means read from standard input (stdin).")
	fmt.Fprintln(out, "  Evaluating '(println (str *command-line-args*))' prints the arguments")
	fmt.Fprintln(out, "    in <repl-args>, <expr-args>, or <script-args> (TBD).")
	fmt.Fprintln(out, "  <socket> is passed to Go's net.Listen() function. If multiple --*repl options are specified,")
	fmt.Fprintln(out, "    the final one specified \"wins\".")

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
	fmt.Fprintln(out, "  --exit-to-repl [<socket>]")
	fmt.Fprintln(out, "    After successfully processing --eval or --file, drop into repl instead of exiting.")
	fmt.Fprintln(out, "  --error-to-repl [<socket>]")
	fmt.Fprintln(out, "    After failure processing --eval or --file, drop into repl instead of exiting.")
	fmt.Fprintln(out, "  --no-readline")
	fmt.Fprintln(out, "    Disable readline functionality in the repl. Useful when using rlwrap.")
	fmt.Fprintln(out, "  --working-dir <directory>")
	fmt.Fprintln(out, "    Specify directory to lint or working directory for lint configuration if linting single file (requires --lint).")
	fmt.Fprintln(out, "  --report-globally-unused")
	fmt.Fprintln(out, "    Report globally unused namespaces and public vars when linting directories (requires --lint and --working-dir).")
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
	debugOut                 io.Writer
	helpFlag                 bool
	versionFlag              bool
	phase                    Phase = EVAL // --read, --parse, --evaluate
	workingDir               string
	lintFlag                 bool
	reportGloballyUnusedFlag bool
	dialect                  Dialect = UNKNOWN
	eval                     string
	replFlag                 bool
	replSocket               string
	classPath                string
	filename                 string
	remainingArgs            []string
	profilerType             string = "runtime/pprof"
	cpuProfileName           string
	cpuProfileRate           int
	cpuProfileRateFlag       bool
	memProfileName           string
	noReadline               bool
	exitToRepl               bool
	errorToRepl              bool
)

func isNumber(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func notOption(arg string) bool {
	return arg == "-" || !strings.HasPrefix(arg, "-") || isNumber(arg[1:])
}

func parseArgs(args []string) {
	if len(args) > 1 {
		// peek to see if the first arg is "--debug*"
		switch args[1] {
		case "--debug", "--debug=stderr":
			debugOut = Stderr
		case "--debug=stdout":
			debugOut = Stdout
		}
	}

	length := len(args)
	stop := false
	missing := false
	noFileFlag := false
	if v, ok := os.LookupEnv("JOKER_CLASSPATH"); ok {
		classPath = v
	} else {
		classPath = ""
	}
	var i int
	for i = 1; i < length; i++ { // shift
		if debugOut != nil {
			fmt.Fprintf(debugOut, "arg[%d]=%s\n", i, args[i])
		}
		switch args[i] {
		case "-": // denotes stdin
			stop = true
		case "--": // formally ends options processing
			stop = true
			noFileFlag = true
			i += 1 // do not include "--" in *command-line-args*
		case "--debug":
			debugOut = Stderr
		case "--debug=stderr":
			debugOut = Stderr
		case "--debug=stdout":
			debugOut = Stdout
		case "--verbose":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				verbosity, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					fmt.Fprintln(Stderr, "Error: ", err)
					return
				}
				if verbosity <= 0 {
					VerbosityLevel = 0
				} else {
					VerbosityLevel = int(verbosity)
				}
			} else {
				VerbosityLevel++
			}
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
		case "--report-globally-unused":
			reportGloballyUnusedFlag = true
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
				thresh, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					fmt.Fprintln(Stderr, "Error: ", err)
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
			} else {
				missing = true
			}
		case "--repl":
			replFlag = true
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				replSocket = args[i]
			}
		case "-c", "--classpath":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				classPath = args[i]
			} else {
				missing = true
			}
		case "--no-readline":
			noReadline = true
		case "--exit-to-repl":
			exitToRepl = true
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				replSocket = args[i]
			}
		case "--error-to-repl":
			errorToRepl = true
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				replSocket = args[i]
			}
		case "--file":
			if i < length-1 && notOption(args[i+1]) {
				i += 1 // shift
				filename = args[i]
			}
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
					fmt.Fprintln(Stderr, "Error: ", err)
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
					fmt.Fprintln(Stderr, "Error: ", err)
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
				fmt.Fprintf(Stderr, "Error: Unrecognized option '%s'\n", args[i])
				ExitJoker(2)
			}
			stop = true
		}
		if stop || missing {
			break
		}
	}
	if missing {
		fmt.Fprintf(Stderr, "Error: Missing argument for '%s' option\n", args[i])
		ExitJoker(3)
	}
	if i < length && !noFileFlag && filename == "" {
		if debugOut != nil {
			fmt.Fprintf(debugOut, "filename=%s\n", args[i])
		}
		filename = args[i]
		i += 1 // shift
	}
	if i < length {
		if debugOut != nil {
			fmt.Fprintf(debugOut, "remaining=%v\n", args[i:])
		}
		remainingArgs = args[i:]
	}
}

var runningProfile interface {
	Stop()
}

func main() {
	SetExitJoker(func(code int) {
		finish()
		os.Exit(code)
	})

	GLOBAL_ENV.InitEnv(Stdin, Stdout, Stderr, os.Args[1:])

	parseArgs(os.Args) // Do this early enough so --verbose can show joker.core being processed.

	saveForRepl = saveForRepl && (exitToRepl || errorToRepl) // don't bother saving stuff if no repl

	RT.GIL.Lock()
	ProcessCoreData()

	GLOBAL_ENV.ReferCoreToUser()
	GLOBAL_ENV.SetEnvArgs(remainingArgs)
	GLOBAL_ENV.SetClassPath(classPath)

	if debugOut != nil {
		fmt.Fprintf(debugOut, "debugOut=%v\n", debugOut)
		fmt.Fprintf(debugOut, "helpFlag=%v\n", helpFlag)
		fmt.Fprintf(debugOut, "versionFlag=%v\n", versionFlag)
		fmt.Fprintf(debugOut, "phase=%v\n", phase)
		fmt.Fprintf(debugOut, "lintFlag=%v\n", lintFlag)
		fmt.Fprintf(debugOut, "reportGloballyUnusedFlag=%v\n", reportGloballyUnusedFlag)
		fmt.Fprintf(debugOut, "dialect=%v\n", dialect)
		fmt.Fprintf(debugOut, "workingDir=%v\n", workingDir)
		fmt.Fprintf(debugOut, "HASHMAP_THRESHOLD=%v\n", HASHMAP_THRESHOLD)
		fmt.Fprintf(debugOut, "eval=%v\n", eval)
		fmt.Fprintf(debugOut, "replFlag=%v\n", replFlag)
		fmt.Fprintf(debugOut, "replSocket=%v\n", replSocket)
		fmt.Fprintf(debugOut, "classPath=%v\n", classPath)
		fmt.Fprintf(debugOut, "noReadline=%v\n", noReadline)
		fmt.Fprintf(debugOut, "filename=%v\n", filename)
		fmt.Fprintf(debugOut, "remainingArgs=%v\n", remainingArgs)
		fmt.Fprintf(debugOut, "exitToRepl=%v\n", exitToRepl)
		fmt.Fprintf(debugOut, "errorToRepl=%v\n", errorToRepl)
		fmt.Fprintf(debugOut, "saveForRepl=%v\n", saveForRepl)
	}

	if helpFlag {
		usage(Stdout)
		return
	}

	if versionFlag {
		println(VERSION)
		return
	}

	if len(remainingArgs) > 0 {
		if lintFlag {
			fmt.Fprintf(Stderr, "Error: Cannot provide arguments to code while linting it.\n")
			ExitJoker(4)
		}
		if phase != EVAL && phase != PRINT_IF_NOT_NIL {
			fmt.Fprintf(Stderr, "Error: Cannot provide arguments to code without evaluating it.\n")
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
				fmt.Fprintf(Stderr, "Error: Could not create CPU profile `%s': %v\n",
					cpuProfileName, err)
				cpuProfileName = ""
				ExitJoker(96)
			}
			if cpuProfileRateFlag {
				runtime.SetCPUProfileRate(cpuProfileRate)
			}
			pprof.StartCPUProfile(f)
			fmt.Fprintf(Stderr, "Profiling started at rate=%d. See file `%s'.\n",
				cpuProfileRate, cpuProfileName)
			defer finish()
		default:
			fmt.Fprintf(Stderr,
				"Unrecognized profiler: %s\n  Use 'pkg/profile' or 'runtime/pprof'.\n",
				profilerType)
			ExitJoker(96)
		}
	} else if memProfileName != "" {
		defer finish()
	}

	if eval != "" {
		if lintFlag {
			fmt.Fprintf(Stderr, "Error: Cannot combine --eval/-e and --lint.\n")
			ExitJoker(6)
		}
		if replFlag {
			fmt.Fprintf(Stderr, "Error: Cannot combine --eval/-e and --repl.\n")
			ExitJoker(7)
		}
		if workingDir != "" {
			fmt.Fprintf(Stderr, "Error: Cannot combine --eval/-e and --working-dir.\n")
			ExitJoker(8)
		}
		if reportGloballyUnusedFlag {
			fmt.Fprintf(Stderr, "Error: Cannot combine --eval/-e and --report-globally-unused.\n")
			ExitJoker(17)
		}
		if filename != "" {
			fmt.Fprintf(Stderr, "Error: Cannot combine --eval/-e and a <filename> argument.\n")
			ExitJoker(9)
		}
		reader := NewReader(strings.NewReader(eval), "<expr>")
		if saveForRepl {
			reader = NewReader(&replayable{reader}, "<replay>")
		}
		if err := ProcessReader(reader, "", phase); err != nil {
			if !errorToRepl {
				ExitJoker(1)
			}
		} else {
			if !exitToRepl {
				return
			}
		}
	}

	if lintFlag {
		if replFlag {
			fmt.Fprintf(Stderr, "Error: Cannot combine --lint and --repl.\n")
			ExitJoker(10)
		}
		if exitToRepl {
			fmt.Fprintf(Stderr, "Error: Cannot combine --lint and --exit-to-repl.\n")
			ExitJoker(14)
		}
		if errorToRepl {
			fmt.Fprintf(Stderr, "Error: Cannot combine --lint and --error-to-repl.\n")
			ExitJoker(15)
		}
		if dialect == UNKNOWN {
			dialect = detectDialect(filename)
		}
		if filename != "" {
			lintFile(filename, dialect, workingDir)
		} else if workingDir != "" {
			lintDir(workingDir, dialect, reportGloballyUnusedFlag)
		} else {
			fmt.Fprintf(Stderr, "Error: Missing --file or --working-dir argument.\n")
			ExitJoker(16)
		}
		if PROBLEM_COUNT > 0 {
			ExitJoker(1)
		}
		return
	}

	if workingDir != "" {
		fmt.Fprintf(Stderr, "Error: Cannot specify --working-dir option when not linting.\n")
		ExitJoker(11)
	}

	if filename != "" {
		if err := processFile(filename, phase); err != nil {
			if !errorToRepl {
				ExitJoker(1)
			}
		} else {
			if !exitToRepl {
				return
			}
		}
	}

	if replSocket != "" {
		srepl(replSocket, phase)
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
		fmt.Fprintf(Stderr, "Profiling stopped. See file `%s'.\n", cpuProfileName)
		cpuProfileName = ""
	}

	if memProfileName != "" {
		f, err := os.Create(memProfileName)
		if err != nil {
			fmt.Fprintf(Stderr, "Error: Could not create memory profile `%s': %v\n",
				memProfileName, err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(Stderr, "Error: Could not write memory profile `%s': %v\n",
				memProfileName, err)
		}
		f.Close()
		fmt.Fprintf(Stderr, "Memory profile rate=%d written to `%s'.\n",
			runtime.MemProfileRate, memProfileName)
		memProfileName = ""
	}
}
