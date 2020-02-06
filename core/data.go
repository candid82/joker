// +build !gen_data

package core

func InitInternalLibs() {
	internalLibs = map[string][]byte{
		"joker.walk":        walkData,
		"joker.template":    templateData,
		"joker.repl":        replData,
		"joker.test":        testData,
		"joker.set":         setData,
		"joker.tools.cli":   tools_cliData,
		"joker.hiccup":      hiccupData,
		"joker.pprint":      pprintData,
		"joker.better-cond": better_condData,
	}
}

func ProcessCoreData() {
	processData(coreData)
	if !haveSetCoreNamespaces {
		setCoreNamespaces()
		haveSetCoreNamespaces = true
	}
}

func ProcessReplData() {
	processData(replData)
}

func ProcessLinterData(dialect Dialect) {
	if dialect == EDN {
		markJokerNamespacesAsUsed()
		return
	}
	processData(linter_allData)
	GLOBAL_ENV.CoreNamespace.Resolve("*loaded-libs*").Value = EmptySet()
	if dialect == JOKER {
		markJokerNamespacesAsUsed()
		processData(linter_jokerData)
		return
	}
	processData(linter_cljxData)
	switch dialect {
	case CLJ:
		processData(linter_cljData)
	case CLJS:
		processData(linter_cljsData)
	}
	removeJokerNamespaces()
}
