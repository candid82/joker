//go:build !plan9
// +build !plan9

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	. "github.com/candid82/joker/core"
	"github.com/candid82/liner"
)

var qualifiedSymbolRe *regexp.Regexp = regexp.MustCompile(`([0-9A-Za-z_\-\+\*\'\.]+)/([0-9A-Za-z_\-\+\*\']*$)`)
var callRe *regexp.Regexp = regexp.MustCompile(`\(\s*([0-9A-Za-z_\-\+\*\'\.]*$)`)

func completer(line string, pos int) (head string, c []string, tail string) {
	head = line[:pos]
	tail = line[pos:]
	var match []string
	var prefix string
	var ns *Namespace
	var addNamespaces bool
	if match = qualifiedSymbolRe.FindStringSubmatch(head); match != nil {
		nsName := match[1]
		prefix = match[2]
		ns = GLOBAL_ENV.NamespaceFor(GLOBAL_ENV.CurrentNamespace(), MakeSymbol(nsName+"/"+prefix))
	} else if match = callRe.FindStringSubmatch(head); match != nil {
		prefix = match[1]
		ns = GLOBAL_ENV.CurrentNamespace()
		addNamespaces = true
	}
	if ns == nil {
		return
	}
	for k, _ := range ns.Mappings() {
		if strings.HasPrefix(*k, prefix) {
			c = append(c, *k)
		}
	}
	if addNamespaces {
		for k, _ := range GLOBAL_ENV.Namespaces {
			if strings.HasPrefix(*k, prefix) {
				c = append(c, *k)
			}
		}
		for k, _ := range ns.Aliases() {
			if strings.HasPrefix(*k, prefix) {
				c = append(c, *k)
			}
		}
	}
	if len(c) > 0 {
		head = head[:len(head)-len(prefix)]
	}
	sort.Strings(c)
	return
}

func saveReplHistory(rl *liner.State, filename string) {
	if filename == "" {
		return
	}
	if f, err := os.Create(filename); err == nil {
		rl.WriteHistory(f)
		f.Close()
	}
}

func repl(phase Phase) {
	ProcessReplData()
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.FindNamespace(MakeSymbol("joker.repl")))
	GLOBAL_ENV.CoreNamespace.Resolve("*repl*").Value = Boolean{B: true}
	fmt.Printf("Welcome to joker %s. Use '(exit)', %s to exit.\n", VERSION, EXITERS)
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	replContext := NewReplContext(parseContext.GlobalEnv)

	var runeReader io.RuneReader
	var rl *liner.State
	var historyFilename string
	if noReadline {
		runeReader = bufio.NewReader(Stdin)
	} else {
		home := HomeDir()
		jokerd := filepath.Join(home, ".jokerd")
		if _, err := os.Stat(jokerd); os.IsNotExist(err) {
			if err := os.MkdirAll(jokerd, 0777); err != nil {
				fmt.Fprintf(Stderr, "WARNING: could not create %s \n", jokerd)
			}
		}
		if !noReplHistory {
			historyFilename = filepath.Join(jokerd, ".repl_history")
		}
		rl = liner.NewLiner()
		OnExit(func() {
			saveReplHistory(rl, historyFilename)
			rl.Close()
		})
		defer rl.Close()
		rl.SetCtrlCAborts(true)
		rl.SetWordCompleter(completer)
		rl.SetTabCompletionStyle(liner.TabPrints)

		if !noReplHistory {
			if f, err := os.Open(historyFilename); err == nil {
				rl.ReadHistory(f)
				f.Close()
			}
		}

		runeReader = NewLineRuneReader(rl)

		for _, line := range strings.Split(string(dataRead), "\n") {
			if strings.TrimSpace(line) != "" {
				rl.AppendHistory(line)
			}
		}
		dataRead = []rune{}
	}

	reader := NewReader(runeReader, "<repl>")

	for {
		namespace := GLOBAL_ENV.CurrentNamespace().Name.ToString(false)
		if noReadline {
			print(namespace + "=> ")
		} else {
			runeReader.(*LineRuneReader).Prompt = (namespace + "=> ")
		}
		if processReplCommand(reader, phase, parseContext, replContext) {
			saveReplHistory(rl, historyFilename)
			return
		}
	}
}
