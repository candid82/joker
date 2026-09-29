//go:build packed_core
// +build packed_core

package core

type packedCoreImage struct {
	code      []byte
	summaries []byte
}

var havePreparedPackedCore bool

func preparePackedCore() {
	if havePreparedPackedCore {
		return
	}
	havePreparedPackedCore = true

	for _, name := range coreNamespaces {
		ns := GLOBAL_ENV.EnsureSymbolIsNamespace(MakeSymbol(name))
		image, found := packedCoreData[name]
		if !found || name == "joker.core" {
			continue
		}
		ns.Lazy = func() {
			processData(image.code)
			ApplyVarSummaries(image.summaries)
		}
	}
}

func ProcessCoreData() {
	preparePackedCore()
	image := packedCoreData["joker.core"]
	processData(image.code)
	ApplyVarSummaries(image.summaries)
	setCoreNamespaces()
}

func ProcessReplData() {
	preparePackedCore()
	GLOBAL_ENV.FindNamespace(MakeSymbol("joker.repl"))
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
