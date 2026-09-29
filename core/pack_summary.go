//go:build gen_code || packed_core
// +build gen_code packed_core

package core

import "sort"

func packTypes(p []byte, types []*Type, env *PackEnv) []byte {
	p = appendInt(p, len(types))
	for _, t := range types {
		p = t.Pack(p, env)
	}
	return p
}

func unpackTypes(p []byte, header *PackHeader) ([]*Type, []byte) {
	count, p := extractCount(p)
	res := make([]*Type, count)
	for i := range res {
		res[i], p = unpackType(p, header)
	}
	return res, p
}

func packBools(p []byte, values []bool) []byte {
	p = appendInt(p, len(values))
	for _, value := range values {
		p = appendBool(p, value)
	}
	return p
}

func unpackBools(p []byte) ([]bool, []byte) {
	count, p := extractCount(p)
	res := make([]bool, count)
	for i := range res {
		res[i], p = extractBool(p)
	}
	return res, p
}

func packFnAritySummary(p []byte, summary *FnAritySummary, env *PackEnv) []byte {
	p = appendInt(p, summary.argCount)
	p = appendBool(p, summary.variadic)
	p = appendBool(p, summary.returnUnknown)
	p = packTypes(p, summary.returnTypes, env)
	p = packBools(p, summary.returnArgDeps)
	p = packArgTypes(p, summary.inferredArgTypes, env)
	p = packArgTypes(p, summary.declaredArgTypes, env)
	return packTypes(p, summary.declaredReturnTypes, env)
}

func unpackFnAritySummary(p []byte, header *PackHeader) (*FnAritySummary, []byte) {
	argCount, p := extractInt(p)
	variadic, p := extractBool(p)
	returnUnknown, p := extractBool(p)
	returnTypes, p := unpackTypes(p, header)
	returnArgDeps, p := unpackBools(p)
	inferredArgTypes, p := unpackArgTypes(p, header)
	declaredArgTypes, p := unpackArgTypes(p, header)
	declaredReturnTypes, p := unpackTypes(p, header)
	return &FnAritySummary{
		argCount:            argCount,
		variadic:            variadic,
		returnUnknown:       returnUnknown,
		returnTypes:         returnTypes,
		returnArgDeps:       returnArgDeps,
		inferredArgTypes:    inferredArgTypes,
		declaredArgTypes:    declaredArgTypes,
		declaredReturnTypes: declaredReturnTypes,
	}, p
}

func packFnSummary(p []byte, summary *FnSummary, env *PackEnv) []byte {
	p = appendInt(p, len(summary.arities))
	for _, arity := range summary.arities {
		p = packFnAritySummary(p, arity, env)
	}
	p = appendBool(p, summary.variadic != nil)
	if summary.variadic != nil {
		p = packFnAritySummary(p, summary.variadic, env)
	}
	return p
}

func unpackFnSummary(p []byte, header *PackHeader) (*FnSummary, []byte) {
	count, p := extractCount(p)
	res := &FnSummary{analyzed: true, arities: make([]*FnAritySummary, count)}
	for i := range res.arities {
		res.arities[i], p = unpackFnAritySummary(p, header)
	}
	var hasVariadic bool
	hasVariadic, p = extractBool(p)
	if hasVariadic {
		res.variadic, p = unpackFnAritySummary(p, header)
	}
	return res, p
}

// PackVarSummaries serializes the linter metadata owned by ns.
func PackVarSummaries(ns *Namespace) []byte {
	names := make([]string, 0)
	vars := make(map[string]*Var)
	for name, vr := range ns.mappings {
		if vr.ns == ns && vr.hasDefinition {
			names = append(names, *name)
			vars[*name] = vr
		}
	}
	sort.Strings(names)

	env := NewPackEnv()
	p := appendInt(nil, len(names))
	for _, name := range names {
		vr := vars[name]
		p = vr.Pack(p, env)
		p = appendBool(p, vr.isMacro)
		p = appendBool(p, vr.isPrivate)
		p = appendBool(p, vr.isDynamic)
		p = packTypes(p, vr.taggedTypes, env)
		p = appendBool(p, vr.inferredUnknown)
		p = appendBool(p, vr.hasInferredValue)
		p = packTypes(p, vr.inferredTypes, env)
		p = appendBool(p, vr.fnSummary != nil)
		if vr.fnSummary != nil {
			p = packFnSummary(p, vr.fnSummary, env)
		}
	}
	return append(env.Pack(nil), p...)
}

// ApplyVarSummaries installs linter metadata after loading a packed namespace.
func ApplyVarSummaries(data []byte) {
	if len(data) == 0 {
		return
	}
	header, p := UnpackHeader(data, GLOBAL_ENV)
	count, p := extractCount(p)
	for i := 0; i < count; i++ {
		var vr *Var
		vr, p = unpackVar(p, header)
		vr.hasDefinition = true
		vr.isMacro, p = extractBool(p)
		vr.isPrivate, p = extractBool(p)
		vr.isDynamic, p = extractBool(p)
		vr.taggedTypes, p = unpackTypes(p, header)
		vr.inferredUnknown, p = extractBool(p)
		vr.hasInferredValue, p = extractBool(p)
		vr.inferredTypes, p = unpackTypes(p, header)
		var hasFnSummary bool
		hasFnSummary, p = extractBool(p)
		if hasFnSummary {
			vr.fnSummary, p = unpackFnSummary(p, header)
		}
	}
	if len(p) != 0 {
		panic(RT.NewError("Trailing packed var summary data"))
	}
}
