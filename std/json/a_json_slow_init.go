// This file is generated by generate-std.joke script. Do not edit manually!

package json

import (
	"fmt"
	. "github.com/candid82/joker/core"
	"os"
)

func InternsOrThunks() {
	if VerbosityLevel > 0 {
		fmt.Fprintln(os.Stderr, "Lazily running slow version of json.InternsOrThunks().")
	}
	jsonNamespace.ResetMeta(MakeMeta(nil, `Implements encoding and decoding of JSON as defined in RFC 4627.`, "1.0"))

	jsonNamespace.InternVar("json-seq", json_seq_,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("rdr")), NewVectorFrom(MakeSymbol("rdr"), MakeSymbol("opts"))),
			`Returns the json records from rdr as a lazy sequence.
  rdr must be a string or implement io.Reader.
  Optional opts map may have the following keys:
  :keywords? - if true, JSON keys will be converted from strings to keywords.`, "1.0"))

	jsonNamespace.InternVar("read-string", read_string_,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s")), NewVectorFrom(MakeSymbol("s"), MakeSymbol("opts"))),
			`Parses the JSON-encoded data and return the result as a Joker value.
  Optional opts map may have the following keys:
  :keywords? - if true, JSON keys will be converted from strings to keywords.`, "1.0"))

	jsonNamespace.InternVar("write-string", write_string_,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("v"))),
			`Returns the JSON encoding of v.`, "1.0"))

}
