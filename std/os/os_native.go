package os

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

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

func commandArgs() Object {
	res := EmptyVector
	for _, arg := range os.Args {
		res = res.Conjoin(String{S: arg})
	}
	return res
}

func sh(name string, args []string) Object {
	cmd := exec.Command(name, args...)
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
