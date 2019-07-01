package os

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	. "github.com/candid82/joker/core"
)

func env() Object {
	res := EmptyArrayMap()
	for _, v := range os.Environ() {
		parts := strings.Split(v, "=")
		res.Add(String{S: parts[0]}, String{S: parts[1]})
	}
	return res
}

func setEnv(key string, value string) Object {
	err := os.Setenv(key, value)
	PanicOnErr(err)
	return NIL
}

func commandArgs() Object {
	res := EmptyVector()
	for _, arg := range os.Args {
		res = res.Conjoin(String{S: arg})
	}
	return res
}

const defaultFailedCode = 127 // seen from 'sh no-such-file' on OS X and Ubuntu

func execute(name string, opts Map) Object {
	var dir string
	var args []string
	var stdin io.Reader
	ok, dirObj := opts.Get(MakeKeyword("dir"))
	if ok {
		dir = AssertString(dirObj, "dir must be a string").S
	}
	ok, argsObj := opts.Get(MakeKeyword("args"))
	if ok {
		s := AssertSeqable(argsObj, "args must be Seqable").Seq()
		for !s.IsEmpty() {
			args = append(args, AssertString(s.First(), "args must be strings").S)
			s = s.Rest()
		}
	}
	ok, stdinObj := opts.Get(MakeKeyword("stdin"))
	if ok {
		if stdinObj.Equals(MakeKeyword("pipe")) {
			stdin = Stdin
		} else {
			stdin = strings.NewReader(AssertString(stdinObj, "stdin must be either :pipe keyword or a string").S)
		}
	}
	return sh(dir, stdin, name, args)
}

func sh(dir string, stdin io.Reader, name string, args []string) Object {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = stdin

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	PanicOnErr(err)

	err = cmd.Wait()

	stdoutString := string(stdout.Bytes())
	stderrString := string(stderr.Bytes())

	res := EmptyArrayMap()
	res.Add(MakeKeyword("success"), Boolean{B: err == nil})

	var exitCode int
	if err != nil {
		res.Add(MakeKeyword("err-msg"), String{S: err.Error()})
		if exiterr, ok := err.(*exec.ExitError); ok {
			ws := exiterr.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			exitCode = defaultFailedCode
			if stderrString == "" {
				stderrString = err.Error()
			}
		}
	} else {
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	res.Add(MakeKeyword("exit"), Int{I: exitCode})
	res.Add(MakeKeyword("out"), String{S: stdoutString})
	res.Add(MakeKeyword("err"), String{S: stderrString})
	return res
}

func mkdir(name string, perm int) Object {
	err := os.Mkdir(name, os.FileMode(perm))
	PanicOnErr(err)
	return NIL
}

func readDir(dirname string) Object {
	files, err := ioutil.ReadDir(dirname)
	PanicOnErr(err)
	res := EmptyVector()
	name := MakeKeyword("name")
	size := MakeKeyword("size")
	mode := MakeKeyword("mode")
	isDir := MakeKeyword("dir?")
	modTime := MakeKeyword("modtime")
	for _, f := range files {
		m := EmptyArrayMap()
		m.Add(name, MakeString(f.Name()))
		m.Add(size, MakeInt(int(f.Size())))
		m.Add(mode, MakeInt(int(f.Mode())))
		m.Add(isDir, MakeBoolean(f.IsDir()))
		m.Add(modTime, MakeInt(int(f.ModTime().Unix())))
		res = res.Conjoin(m)
	}
	return res
}

func getwd() string {
	res, err := os.Getwd()
	PanicOnErr(err)
	return res
}

func chdir(dirname string) Object {
	err := os.Chdir(dirname)
	PanicOnErr(err)
	return NIL
}

func stat(filename string) Object {
	info, err := os.Stat(filename)
	PanicOnErr(err)
	return FileInfoMap(info.Name(), info)
}
