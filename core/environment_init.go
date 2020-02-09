// +build !fast_init

package core

/* Called by parse_init.go in an outer var block, this runs before any
   func init() as well as before func main(). InitEnv() and others are
   called at runtime to set some of these Values based on the current
   invocation. */
func NewEnv() *Env {
	features := EmptySet()
	features.Add(MakeKeyword("default"))
	features.Add(MakeKeyword("joker"))
	res := &Env{
		Namespaces: make(map[*string]*Namespace),
		Features:   features,
	}
	res.CoreNamespace = res.EnsureNamespace(SYMBOLS.joker_core)
	res.CoreNamespace.meta = MakeMeta(nil, "Core library of Joker.", "1.0")
	res.NS_VAR = res.CoreNamespace.Intern(MakeSymbol("ns"))
	res.IN_NS_VAR = res.CoreNamespace.Intern(MakeSymbol("in-ns"))
	res.ns = res.CoreNamespace.Intern(MakeSymbol("*ns*"))
	res.stdin = res.CoreNamespace.Intern(MakeSymbol("*in*"))
	res.stdout = res.CoreNamespace.Intern(MakeSymbol("*out*"))
	res.stderr = res.CoreNamespace.Intern(MakeSymbol("*err*"))
	res.file = res.CoreNamespace.Intern(MakeSymbol("*file*"))
	res.MainFile = res.CoreNamespace.Intern(MakeSymbol("*main-file*"))
	res.version = res.CoreNamespace.InternVar("*joker-version*", versionMap(),
		MakeMeta(nil, `The version info for Clojure core, as a map containing :major :minor
			:incremental and :qualifier keys. Feature releases may increment
			:minor and/or :major, bugfix releases will increment :incremental.`, "1.0"))
	res.args = res.CoreNamespace.Intern(MakeSymbol("*command-line-args*"))
	res.classPath = res.CoreNamespace.Intern(MakeSymbol("*classpath*"))
	res.classPath.Value = NIL
	res.classPath.isPrivate = true
	res.printReadably = res.CoreNamespace.Intern(MakeSymbol("*print-readably*"))
	res.printReadably.Value = Boolean{B: true}
	res.CoreNamespace.InternVar("*linter-mode*", Boolean{B: LINTER_MODE},
		MakeMeta(nil, "true if Joker is running in linter mode", "1.0"))
	res.CoreNamespace.InternVar("*linter-config*", EmptyArrayMap(),
		MakeMeta(nil, "Map of configuration key/value pairs for linter mode", "1.0"))
	return res
}

func (env *Env) ReferCoreToUser() {
	env.FindNamespace(MakeSymbol("user")).ReferAll(env.CoreNamespace)
}
