// +build gen_code

package core

var (
	GLOBAL_ENV = NewEnv()
	KEYWORDS   = Keywords{
		tag:                MakeKeyword("tag"),
		skipUnused:         MakeKeyword("skip-unused"),
		private:            MakeKeyword("private"),
		line:               MakeKeyword("line"),
		column:             MakeKeyword("column"),
		file:               MakeKeyword("file"),
		ns:                 MakeKeyword("ns"),
		macro:              MakeKeyword("macro"),
		message:            MakeKeyword("message"),
		form:               MakeKeyword("form"),
		data:               MakeKeyword("data"),
		cause:              MakeKeyword("cause"),
		arglist:            MakeKeyword("arglists"),
		doc:                MakeKeyword("doc"),
		added:              MakeKeyword("added"),
		meta:               MakeKeyword("meta"),
		knownMacros:        MakeKeyword("known-macros"),
		rules:              MakeKeyword("rules"),
		ifWithoutElse:      MakeKeyword("if-without-else"),
		unusedFnParameters: MakeKeyword("unused-fn-parameters"),
		fnWithEmptyBody:    MakeKeyword("fn-with-empty-body"),
		_prefix:            MakeKeyword("_prefix"),
		pos:                MakeKeyword("pos"),
		startLine:          MakeKeyword("start-line"),
		endLine:            MakeKeyword("end-line"),
		startColumn:        MakeKeyword("start-column"),
		endColumn:          MakeKeyword("end-column"),
		filename:           MakeKeyword("filename"),
		object:             MakeKeyword("object"),
		type_:              MakeKeyword("type"),
		var_:               MakeKeyword("var"),
		value:              MakeKeyword("value"),
		vector:             MakeKeyword("vector"),
		name:               MakeKeyword("name"),
		dynamic:            MakeKeyword("dynamic"),
	}
	SYMBOLS = Symbols{
		joker_core:         MakeSymbol("joker.core"),
		underscore:         MakeSymbol("_"),
		catch:              MakeSymbol("catch"),
		finally:            MakeSymbol("finally"),
		amp:                MakeSymbol("&"),
		_if:                MakeSymbol("if"),
		quote:              MakeSymbol("quote"),
		fn_:                MakeSymbol("fn*"),
		fn:                 MakeSymbol("fn"),
		let_:               MakeSymbol("let*"),
		letfn_:             MakeSymbol("letfn*"),
		loop_:              MakeSymbol("loop*"),
		recur:              MakeSymbol("recur"),
		setMacro_:          MakeSymbol("set-macro__"),
		def:                MakeSymbol("def"),
		defLinter:          MakeSymbol("def-linter__"),
		_var:               MakeSymbol("var"),
		do:                 MakeSymbol("do"),
		throw:              MakeSymbol("throw"),
		try:                MakeSymbol("try"),
		unquoteSplicing:    MakeSymbol("unquote-splicing"),
		list:               MakeSymbol("list"),
		concat:             MakeSymbol("concat"),
		seq:                MakeSymbol("seq"),
		apply:              MakeSymbol("apply"),
		emptySymbol:        MakeSymbol(""),
		unquote:            MakeSymbol("unquote"),
		vector:             MakeSymbol("vector"),
		hashMap:            MakeSymbol("hash-map"),
		hashSet:            MakeSymbol("hash-set"),
		defaultDataReaders: MakeSymbol("default-data-readers"),
		backslash:          MakeSymbol("/"),
		deref:              MakeSymbol("deref"),
	}
	STR = Str{
		_if:          STRINGS.Intern("if"),
		quote:        STRINGS.Intern("quote"),
		fn_:          STRINGS.Intern("fn*"),
		let_:         STRINGS.Intern("let*"),
		letfn_:       STRINGS.Intern("letfn*"),
		loop_:        STRINGS.Intern("loop*"),
		recur:        STRINGS.Intern("recur"),
		setMacro_:    STRINGS.Intern("set-macro__"),
		def:          STRINGS.Intern("def"),
		defLinter:    STRINGS.Intern("def-linter__"),
		_var:         STRINGS.Intern("var"),
		do:           STRINGS.Intern("do"),
		throw:        STRINGS.Intern("throw"),
		try:          STRINGS.Intern("try"),
		coreFilename: STRINGS.Intern("<joker.core>"),
	}
	SPECIAL_SYMBOLS = make(map[*string]bool)
)

func init() {
	SPECIAL_SYMBOLS[SYMBOLS._if.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.quote.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.fn_.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.let_.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.letfn_.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.loop_.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.recur.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.setMacro_.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.def.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.defLinter.name] = true
	SPECIAL_SYMBOLS[SYMBOLS._var.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.do.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.throw.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.try.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.catch.name] = true
	SPECIAL_SYMBOLS[SYMBOLS.finally.name] = true
}
