// +build !gen_data

package core

func ProcessCoreData() {
	// Let MaybeLazy() handle initialization.
	if !haveSetCoreNamespaces {
		setCoreNamespaces()
		haveSetCoreNamespaces = true
	}
}

func ProcessReplData() {
	// Let MaybeLazy() handle initialization.
}

func ProcessLinterData(dialect Dialect) {
	if dialect == EDN {
		markJokerNamespacesAsUsed()
		return
	}
	processData(linter_allData, "linter_all.joke")
	GLOBAL_ENV.CoreNamespace.Resolve("*loaded-libs*").Value = EmptySet()
	if dialect == JOKER {
		markJokerNamespacesAsUsed()
		processData(linter_jokerData, "linter_joker.joke")
		return
	}
	processData(linter_cljxData, "linter_cljx.joke")
	switch dialect {
	case CLJ:
		processData(linter_cljData, "linter_clj.joke")
	case CLJS:
		processData(linter_cljsData, "linter_cljs.joke")
	}
	removeJokerNamespaces()
}
