// This file is generated by generate-std.joke script. Do not edit manually!

package io

import (
	"fmt"
	. "github.com/candid82/joker/core"
	"os"
)

func InternsOrThunks() {
	if Verbosity() > 0 {
		fmt.Fprintln(os.Stderr, "Lazily running slow version of io.InternsOrThunks().")
	}
	ioNamespace.ResetMeta(MakeMeta(nil, `Provides basic interfaces to I/O primitives.`, "1.0"))

	ioNamespace.InternVar("copy", copy_,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("dst"), MakeSymbol("src"))),
			`Copies from src to dst until either EOF is reached on src or an error occurs.
  Returns the number of bytes copied or throws an error.
  src must be IOReader, e.g. as returned by joker.os/open.
  dst must be IOWriter, e.g. as returned by joker.os/create.`, "1.0").Plus(MakeKeyword("tag"), String{S: "Int"}))

}