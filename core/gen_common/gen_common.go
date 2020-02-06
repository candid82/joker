package gen_common

type FileInfo struct {
	Name     string
	Filename string
}

/* The entries must be ordered such that a given namespace depends
/* only upon namespaces loaded above it. E.g. joker.template depends
/* on joker.walk, so is listed afterwards, not in alphabetical
/* order. */
var CoreSourceFiles []FileInfo = []FileInfo{
	{
		Name:     "<joker.core>",
		Filename: "core.joke",
	},
	{
		Name:     "<joker.repl>",
		Filename: "repl.joke",
	},
	{
		Name:     "<joker.walk>",
		Filename: "walk.joke",
	},
	{
		Name:     "<joker.template>",
		Filename: "template.joke",
	},
	{
		Name:     "<joker.test>",
		Filename: "test.joke",
	},
	{
		Name:     "<joker.set>",
		Filename: "set.joke",
	},
	{
		Name:     "<joker.tools.cli>",
		Filename: "tools_cli.joke",
	},
	{
		Name:     "<joker.core>",
		Filename: "linter_all.joke",
	},
	{
		Name:     "<joker.core>",
		Filename: "linter_joker.joke",
	},
	{
		Name:     "<joker.core>",
		Filename: "linter_cljx.joke",
	},
	{
		Name:     "<joker.core>",
		Filename: "linter_clj.joke",
	},
	{
		Name:     "<joker.core>",
		Filename: "linter_cljs.joke",
	},
	{
		Name:     "<joker.hiccup>",
		Filename: "hiccup.joke",
	},
	{
		Name:     "<joker.pprint>",
		Filename: "pprint.joke",
	},
	{
		Name:     "<joker.better-cond>",
		Filename: "better_cond.joke",
	},
}
