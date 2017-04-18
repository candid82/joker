package os

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	. "github.com/candid82/joker/core"
)

var env Proc = func(args []Object) Object {
	res := EmptyArrayMap()
	for _, v := range os.Environ() {
		parts := strings.Split(v, "=")
		res.Add(String{S: parts[0]}, String{S: parts[1]})
	}
	return res
}

var args Proc = func(args []Object) Object {
	res := EmptyVector
	for _, arg := range os.Args {
		res = res.Conjoin(String{S: arg})
	}
	return res
}

var exit Proc = func(args []Object) Object {
	CheckArity(args, 1, 1)
	code := EnsureInt(args, 0)
	os.Exit(code.I)
	return NIL
}

var sh Proc = func(args []Object) Object {
	strs := make([]string, len(args))
	for i := range args {
		strs[i] = EnsureString(args, i).S
	}
	cmd := exec.Command(strs[0], strs[1:]...)
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	if err = cmd.Start(); err != nil {
		panic(RT.NewError(err.Error()))
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdoutReader)
	stdoutString := buf.String()
	buf = new(bytes.Buffer)
	buf.ReadFrom(stderrReader)
	stderrString := buf.String()
	if err = cmd.Wait(); err != nil {
		EmptyArrayMap().Assoc(MakeKeyword("success"), Bool{B: false})
	}
	res := EmptyArrayMap()
	res.Add(MakeKeyword("success"), Bool{B: true})
	res.Add(MakeKeyword("out"), String{S: stdoutString})
	res.Add(MakeKeyword("err"), String{S: stderrString})
	return res
}

var osNamespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("joker.os"))

func intern(name string, proc Proc) {
	osNamespace.Intern(MakeSymbol(name)).Value = proc
}

func init() {
	osNamespace.ResetMeta(MakeMeta(nil, "Provides a platform-independent interface to operating system functionality.", "1.0"))
	osNamespace.InternVar("env", env, MakeMeta(NewListFrom(EmptyVector), "Returns a map representing the environment.", "1.0"))
	osNamespace.InternVar("exit", exit,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("code"))),
			"Causes the current program to exit with the given status code.", "1.0"))
	osNamespace.InternVar("args", args,
		MakeMeta(
			NewListFrom(EmptyVector),
			"Returns a sequence of the command line arguments, starting with the program name (normally, joker).", "1.0"))
	osNamespace.InternVar("sh", sh,
		MakeMeta(
			NewListFrom(
				NewVectorFrom(MakeSymbol("name"), MakeSymbol("&"), MakeSymbol("args"))),
			`Executes the named program with the given arguments. Returns a map with the following keys:
			:success - whether or not the execution was successful,
			:out - string capturing stdout of the program,
			:err - string capturing stderr of the program.`, "1.0"))
}
