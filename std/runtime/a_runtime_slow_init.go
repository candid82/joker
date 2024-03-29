// This file is generated by generate-std.joke script. Do not edit manually!

package runtime

import (
	"fmt"
	. "github.com/candid82/joker/core"
	"os"
)

func InternsOrThunks() {
	if VerbosityLevel > 0 {
		fmt.Fprintln(os.Stderr, "Lazily running slow version of runtime.InternsOrThunks().")
	}
	runtimeNamespace.ResetMeta(MakeMeta(nil, `Provides access to Go and Joker runtime information.`, "1.0"))

	runtimeNamespace.InternVar("go-root", go_root_,
		MakeMeta(
			NewListFrom(NewVectorFrom()),
			`Returns the GOROOT string (as returned by runtime/GOROOT()).`, "1.0").Plus(MakeKeyword("tag"), String{S: "String"}))

	runtimeNamespace.InternVar("go-version", go_version_,
		MakeMeta(
			NewListFrom(NewVectorFrom()),
			`Returns the Go version string (as returned by runtime/Version()).`, "1.0").Plus(MakeKeyword("tag"), String{S: "String"}))

	runtimeNamespace.InternVar("joker-version", joker_version_,
		MakeMeta(
			NewListFrom(NewVectorFrom()),
			`Returns the raw Joker version string (including the leading 'v',
  which joker.core/joker-version omits).`, "1.0").Plus(MakeKeyword("tag"), String{S: "String"}))

}
