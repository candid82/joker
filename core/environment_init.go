// +build !fast_init

package core

import ()

/* Called by parse_init.go in an outer var block, this runs before any
/* func init() as well as before func main(). */
func NewEnv() (env *Env) {
	features := EmptySet()
	features.Add(MakeKeyword("default"))
	features.Add(MakeKeyword("joker"))
	env = &Env{
		Namespaces: make(map[*string]*Namespace),
		Features:   features,
	}
	env.CoreNamespace = env.EnsureNamespace(SYMBOLS.joker_core)
	env.CoreNamespace.meta = MakeMeta(nil, "Core library of Joker.", "1.0")

	/* This runs during invariant initialization.  InitEnv() and
	/* others are called at runtime to set some of these Values
	/* based on the current invocation. NOTE: Any changes to the
	/* list of run-time initializations must be reflected in
	/* gen_code/gen_code.go.  */

	env.NS_VAR = env.CoreNamespace.Intern(MakeSymbol("ns"))
	env.IN_NS_VAR = env.CoreNamespace.Intern(MakeSymbol("in-ns"))
	env.ns = env.CoreNamespace.Intern(MakeSymbol("*ns*"))
	env.stdin = env.CoreNamespace.Intern(MakeSymbol("*in*"))
	env.stdout = env.CoreNamespace.Intern(MakeSymbol("*out*"))
	env.stderr = env.CoreNamespace.Intern(MakeSymbol("*err*"))
	env.file = env.CoreNamespace.Intern(MakeSymbol("*file*"))
	env.MainFile = env.CoreNamespace.Intern(MakeSymbol("*main-file*"))
	env.version = env.CoreNamespace.InternVar("*joker-version*", versionMap(),
		MakeMeta(nil, `The version info for Clojure core, as a map containing :major :minor
			:incremental and :qualifier keys. Feature releases may increment
			:minor and/or :major, bugfix releases will increment :incremental.`, "1.0"))
	env.args = env.CoreNamespace.Intern(MakeSymbol("*command-line-args*"))
	env.classPath = env.CoreNamespace.Intern(MakeSymbol("*classpath*"))
	env.classPath.Value = NIL
	env.classPath.isPrivate = true
	env.printReadably = env.CoreNamespace.Intern(MakeSymbol("*print-readably*"))
	env.printReadably.Value = Boolean{B: true}
	env.CoreNamespace.InternVar("*linter-mode*", Boolean{B: LINTER_MODE},
		MakeMeta(nil, "true if Joker is running in linter mode", "1.0"))
	env.CoreNamespace.InternVar("*linter-config*", EmptyArrayMap(),
		MakeMeta(nil, "Map of configuration key/value pairs for linter mode", "1.0"))
	env.verbose = env.CoreNamespace.Intern(MakeSymbol("*verbose*"))
	env.verbose.Value = Int{I: 0}
	env.verbose.isPrivate = true
	return
}
