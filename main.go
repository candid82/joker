package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func readFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return
	}
	reader := NewReader(bufio.NewReader(f))
	for {
		_, err := TryRead(reader)
		switch {
		case err == io.EOF:
			return
		case err != nil:
			fmt.Fprintln(os.Stderr, "Error: ", err)
			return
		}
	}
}

func repl() {
	fmt.Println("Welcome to gclojure. Use ctrl-c to exit.")
	reader := NewReader(bufio.NewReader(os.Stdin))
	for {
		fmt.Print("> ")
		obj, err := TryRead(reader)
		switch {
		case err == io.EOF:
			return
		case err != nil:
			fmt.Fprintln(os.Stderr, "Error: ", err)
		default:
			fmt.Println(obj.ToString(false))
		}
	}
}

func main() {
	if len(os.Args) > 1 {
		readFile(os.Args[1])
	} else {
		repl()
	}
}
