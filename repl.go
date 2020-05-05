// +build !plan9

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

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
	fmt.Printf("Welcome to joker %s. Use '(exit)', EOF (Ctrl-D), or SIGINT (Ctrl-C) to exit; '(suspend)' to suspend.\n", VERSION)
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	replContext := NewReplContext(parseContext.GlobalEnv)

	var runeReader io.RuneReader
	var rl *liner.State
	var historyFilename string
	var reader *Reader
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
		defer rl.Close()

		OnExit(func() {
			saveReplHistory(rl, historyFilename)
			rl.Close()
		})

		stop := make(chan os.Signal, 1)
		cont := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTSTP)
		signal.Notify(cont, syscall.SIGCONT)
		go func() {
			for {
				<-stop
				err := syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
				PanicOnErr(err)
			}
		}()

		rl.SetCtrlCAborts(true)
		rl.SetWordCompleter(completer)
		rl.SetTabCompletionStyle(liner.TabPrints)
		rl.SetSuspendFn(func() {
			fmt.Println("^Z [Joker]")
			err := syscall.Kill(syscall.Getpid(), syscall.SIGTSTP)
			PanicOnErr(err)
			<-cont
		})

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

	reader = NewReader(runeReader, "<repl>")

	for {
		if noReadline {
			print(GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		} else {
			runeReader.(*LineRuneReader).Prompt = (GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		}
		if processReplCommand(reader, phase, parseContext, replContext) {
			saveReplHistory(rl, historyFilename)
			return
		}
	}
}
