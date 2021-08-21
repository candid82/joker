//go:build !gen_code
// +build !gen_code

package core

var haveSetCoreNamespaces bool

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
	processData(linter_allData)
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
}
