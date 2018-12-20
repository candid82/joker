package main

import (
	"crypto/sha256"
	"path/filepath"
	"fmt"
	"io"
	"os"
)

var h = sha256.New()

func processFile(file string) (err error) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	_, err = io.Copy(h, f)
	return
}

func processEntry(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	name := info.Name()
	if name == ".git" {
		return filepath.SkipDir
	}
	if !info.IsDir() {
		err = processFile(path)
	}
	return err
}

func main() {
	for _, arg := range os.Args[1:] {
		info, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "os.Stat: %v\n", err)
			os.Exit(3)
		}
		if info.IsDir() {
			err = filepath.Walk(arg, processEntry)
		} else {
			err = processFile(arg)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(3)
		}
	}

	fmt.Printf("%x\n", h.Sum(nil))
}
