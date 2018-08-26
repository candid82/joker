package core

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type (
	Env struct {
		Namespaces    map[*string]*Namespace
		CoreNamespace *Namespace
		stdout        *Var
		stdin         *Var
		stderr        *Var
		printReadably *Var
		file          *Var
		args          *Var
		ns            *Var
		version       *Var
		Features      Set
	}
)

func versionMap() Map {
	res := EmptyArrayMap()
	parts := strings.Split(VERSION[1:], ".")
	i, _ := strconv.ParseInt(parts[0], 10, 64)
	res.Add(MakeKeyword("major"), Int{I: int(i)})
	i, _ = strconv.ParseInt(parts[1], 10, 64)
	res.Add(MakeKeyword("minor"), Int{I: int(i)})
	i, _ = strconv.ParseInt(parts[2], 10, 64)
	res.Add(MakeKeyword("incremental"), Int{I: int(i)})
	return res
}

func NewEnv(currentNs Symbol, stdout *os.File, stdin *os.File, stderr *os.File) *Env {
	features := EmptySet()
	features.Add(MakeKeyword("default"))
	features.Add(MakeKeyword("joker"))
	res := &Env{
		Namespaces: make(map[*string]*Namespace),
		Features:   features,
	}
	res.CoreNamespace = res.EnsureNamespace(SYMBOLS.joker_core)
	res.CoreNamespace.meta = MakeMeta(nil, "Core library of Joker.", "1.0")
	res.ns = res.CoreNamespace.Intern(MakeSymbol("*ns*"))
	res.ns.Value = res.EnsureNamespace(currentNs)
	res.stdout = res.CoreNamespace.Intern(MakeSymbol("*out*"))
	res.stdout.Value = &File{stdout}
	res.stdin = res.CoreNamespace.Intern(MakeSymbol("*in*"))
	res.stdin.Value = &BufferedReader{bufio.NewReader(stdin)}
	res.stderr = res.CoreNamespace.Intern(MakeSymbol("*err*"))
	res.stderr.Value = &File{stderr}
	res.file = res.CoreNamespace.Intern(MakeSymbol("*file*"))
	res.version = res.CoreNamespace.InternVar("*joker-version*", versionMap(),
		MakeMeta(nil, `The version info for Clojure core, as a map containing :major :minor
			:incremental and :qualifier keys. Feature releases may increment
			:minor and/or :major, bugfix releases will increment :incremental.`, "1.0"))
	res.args = res.CoreNamespace.Intern(MakeSymbol("*command-line-args*"))
	args := EmptyVector
	for _, arg := range os.Args[1:] {
		args = args.Conjoin(String{S: arg})
	}
	if args.Count() > 0 {
		res.args.Value = args
	} else {
		res.args.Value = NIL
	}
	res.printReadably = res.CoreNamespace.Intern(MakeSymbol("*print-readably*"))
	res.printReadably.Value = Bool{B: true}
	res.CoreNamespace.Intern(MakeSymbol("*linter-mode*")).Value = Bool{B: LINTER_MODE}
	res.CoreNamespace.Intern(MakeSymbol("*linter-config*")).Value = EmptyArrayMap()
	return res
}

func (env *Env) CurrentNamespace() *Namespace {
	return AssertNamespace(env.ns.Value, "")
}

func (env *Env) EnsureNamespace(sym Symbol) *Namespace {
	if sym.ns != nil {
		panic(RT.NewError("Namespace's name cannot be qualified: " + sym.ToString(false)))
	}
	if env.Namespaces[sym.name] == nil {
		env.Namespaces[sym.name] = NewNamespace(sym)
		return env.Namespaces[sym.name]
	}
	res := env.Namespaces[sym.name]
	// In linter mode the latest reference to the ns overrides position info.
	// This is for the cases when (ns ...) is called in .jokerd/linter.clj file and alike.
	if LINTER_MODE {
		res.Name = res.Name.WithInfo(sym.GetInfo()).(Symbol)
	}
	return res
}

func (env *Env) NamespaceFor(ns *Namespace, s Symbol) *Namespace {
	var res *Namespace
	if s.ns == nil {
		res = ns
	} else {
		res = ns.aliases[s.ns]
		if res == nil {
			res = env.Namespaces[s.ns]
		}
	}
	return res
}

func (env *Env) ResolveIn(n *Namespace, s Symbol) (*Var, bool) {
	ns := env.NamespaceFor(n, s)
	if ns == nil {
		return nil, false
	}
	v, ok := ns.mappings[s.name]
	return v, ok
}

func (env *Env) Resolve(s Symbol) (*Var, bool) {
	return env.ResolveIn(env.CurrentNamespace(), s)
}

func (env *Env) FindNamespace(s Symbol) *Namespace {
	if s.ns != nil {
		return nil
	}
	return env.Namespaces[s.name]
}

func (env *Env) RemoveNamespace(s Symbol) *Namespace {
	if s.ns != nil {
		return nil
	}
	if s.Equals(SYMBOLS.joker_core) {
		panic(RT.NewError("Cannot remove core namespace"))
	}
	ns := env.Namespaces[s.name]
	delete(env.Namespaces, s.name)
	return ns
}

func (env *Env) ResolveSymbol(s Symbol) Symbol {
	if strings.ContainsRune(*s.name, '.') {
		return s
	}
	currentNs := env.CurrentNamespace()
	if s.ns != nil {
		ns := env.NamespaceFor(currentNs, s)
		if ns == nil || ns.Name.name == s.ns {
			if ns != nil {
				ns.isUsed = true
			}
			return s
		}
		ns.isUsed = true
		return Symbol{
			name: s.name,
			ns:   ns.Name.name,
		}
	}
	vr, ok := currentNs.mappings[s.name]
	if !ok {
		return Symbol{
			name: s.name,
			ns:   currentNs.Name.name,
		}
	}
	vr.isUsed = true
	vr.ns.isUsed = true
	return Symbol{
		name: vr.name.name,
		ns:   vr.ns.Name.name,
	}
}
