package core

import "fmt"

type InferredValue struct {
	unknown bool
	types   []*Type
	fn      *FnSummary
}

type FnSummary struct {
	fn        *FnExpr
	analyzing bool
	analyzed  bool
	arities   []*FnAritySummary
	variadic  *FnAritySummary
}

type FnAritySummary struct {
	arity            *FnArityExpr
	variadic         bool
	returnUnknown    bool
	returnTypes      []*Type
	returnArgDeps    []bool
	inferredArgTypes [][]*Type
	declaredArgTypes [][]*Type
}

type InferEnv struct {
	analyzingBindings map[*Binding]bool
	requiredTypes     map[*Binding][]*Type
}

func newInferEnv() *InferEnv {
	return &InferEnv{
		analyzingBindings: map[*Binding]bool{},
		requiredTypes:     map[*Binding][]*Type{},
	}
}

func unknownInferredValue() InferredValue {
	return InferredValue{unknown: true}
}

func typeInferredValue(t *Type) InferredValue {
	if t == nil {
		return unknownInferredValue()
	}
	return InferredValue{types: []*Type{t}}
}

func typesInferredValue(types []*Type) InferredValue {
	if len(types) == 0 {
		return unknownInferredValue()
	}
	return InferredValue{types: copyTypes(types)}
}

func fnInferredValue(fn *FnExpr) InferredValue {
	return InferredValue{types: []*Type{TYPE.Fn}, fn: getFnSummary(fn)}
}

func addType(types []*Type, t *Type) []*Type {
	if t == nil {
		return types
	}
	for _, existing := range types {
		if existing == t {
			return types
		}
	}
	return append(types, t)
}

func joinTypes(a, b []*Type) []*Type {
	res := a
	for _, t := range b {
		res = addType(res, t)
	}
	return res
}

func joinInferredValues(a, b InferredValue) InferredValue {
	return InferredValue{
		unknown: a.unknown || b.unknown,
		types:   joinTypes(a.types, b.types),
	}
}

func inferredTypesString(types []*Type) string {
	return typesString(types)
}

func inferredTypesCompatible(expected []*Type, actual []*Type) bool {
	if len(expected) == 0 || len(actual) == 0 {
		return true
	}
	for _, actualType := range actual {
		if isTypeOneOf(expected, actualType) {
			return true
		}
	}
	return false
}

func typeInList(types []*Type, t *Type) bool {
	for _, existing := range types {
		if existing == t {
			return true
		}
	}
	return false
}

func diffTypes(types, base []*Type) []*Type {
	var res []*Type
	for _, t := range types {
		if !typeInList(base, t) {
			res = addType(res, t)
		}
	}
	return res
}

func copyTypes(types []*Type) []*Type {
	if len(types) == 0 {
		return nil
	}
	res := make([]*Type, len(types))
	copy(res, types)
	return res
}

func (env *InferEnv) clone() *InferEnv {
	if env == nil {
		return newInferEnv()
	}
	res := newInferEnv()
	for binding, analyzing := range env.analyzingBindings {
		res.analyzingBindings[binding] = analyzing
	}
	for binding, types := range env.requiredTypes {
		res.requiredTypes[binding] = copyTypes(types)
	}
	return res
}

func (env *InferEnv) addRequiredTypes(binding *Binding, expected []*Type) {
	if env == nil || binding == nil {
		return
	}
	for _, t := range expected {
		env.requiredTypes[binding] = addType(env.requiredTypes[binding], t)
	}
}

func (env *InferEnv) mergeBranches(positive, negative *InferEnv) {
	if env == nil || positive == nil || negative == nil {
		return
	}
	for binding, positiveTypes := range positive.requiredTypes {
		baseTypes := env.requiredTypes[binding]
		negativeTypes := negative.requiredTypes[binding]
		positiveAdded := diffTypes(positiveTypes, baseTypes)
		negativeAdded := diffTypes(negativeTypes, baseTypes)
		if len(positiveAdded) == 0 || len(negativeAdded) == 0 {
			continue
		}
		env.requiredTypes[binding] = joinTypes(env.requiredTypes[binding], joinTypes(positiveAdded, negativeAdded))
	}
}

func requireExpr(expr Expr, expected []*Type, env *InferEnv) {
	if len(expected) == 0 || expr == nil {
		return
	}
	switch expr := expr.(type) {
	case *BindingExpr:
		env.addRequiredTypes(expr.binding, expected)
		if expr.binding != nil && expr.binding.valueExpr != nil {
			requireExpr(expr.binding.valueExpr, expected, env)
		}
	case *MetaExpr:
		requireExpr(expr.expr, expected, env)
	case *DoExpr:
		if len(expr.body) > 0 {
			requireExpr(expr.body[len(expr.body)-1], expected, env)
		}
	case *IfExpr:
		requireExpr(expr.positive, expected, env)
		requireExpr(expr.negative, expected, env)
	case *LetExpr:
		if len(expr.body) > 0 {
			requireExpr(expr.body[len(expr.body)-1], expected, env)
		}
	case *LoopExpr:
		le := (*LetExpr)(expr)
		if len(le.body) > 0 {
			requireExpr(le.body[len(le.body)-1], expected, env)
		}
	case *TryExpr:
		if len(expr.body) > 0 {
			requireExpr(expr.body[len(expr.body)-1], expected, env)
		}
		for _, catchExpr := range expr.catches {
			if len(catchExpr.body) > 0 {
				requireExpr(catchExpr.body[len(catchExpr.body)-1], expected, env)
			}
		}
	case *CallExpr:
		if !shouldCheckInferredSummary(expr.callable) {
			return
		}
		_, arity := callableFnSummary(expr.callable, len(expr.args))
		if arity == nil {
			return
		}
		for i, dep := range arity.returnArgDeps {
			if dep && i < len(expr.args) {
				requireExpr(expr.args[i], expected, env)
			}
		}
	}
}

func bindingValue(binding *Binding, env *InferEnv) InferredValue {
	if binding == nil {
		return unknownInferredValue()
	}
	if binding.inferredValue != nil {
		return *binding.inferredValue
	}
	if binding.valueExpr == nil {
		return unknownInferredValue()
	}
	if env.analyzingBindings[binding] {
		return unknownInferredValue()
	}
	env.analyzingBindings[binding] = true
	value := binding.valueExpr.InferValue(env)
	delete(env.analyzingBindings, binding)
	binding.inferredValue = &value
	return value
}

func inferBodyValue(body []Expr, env *InferEnv) InferredValue {
	if len(body) == 0 {
		return typeInferredValue(TYPE.Nil)
	}
	var res InferredValue
	for _, expr := range body {
		res = expr.InferValue(env)
	}
	return res
}

func (expr *LiteralExpr) InferValue(env *InferEnv) InferredValue {
	if expr.isSurrogate {
		return unknownInferredValue()
	}
	return typeInferredValue(expr.obj.GetType())
}

func (expr *VectorExpr) InferValue(env *InferEnv) InferredValue {
	return typeInferredValue(TYPE.Vec)
}

func (expr *MapExpr) InferValue(env *InferEnv) InferredValue {
	return typeInferredValue(TYPE.ArrayMap)
}

func (expr *SetExpr) InferValue(env *InferEnv) InferredValue {
	return typeInferredValue(TYPE.MapSet)
}

func (expr *IfExpr) InferValue(env *InferEnv) InferredValue {
	expr.cond.InferValue(env)
	positiveEnv := env.clone()
	negativeEnv := env.clone()
	positive := expr.positive.InferValue(positiveEnv)
	negative := expr.negative.InferValue(negativeEnv)
	env.mergeBranches(positiveEnv, negativeEnv)
	return joinInferredValues(positive, negative)
}

func (expr *DefExpr) InferValue(env *InferEnv) InferredValue {
	return typeInferredValue(TYPE.Var)
}

func (expr *MacroCallExpr) InferValue(env *InferEnv) InferredValue {
	return unknownInferredValue()
}

func (expr *RecurExpr) InferValue(env *InferEnv) InferredValue {
	return unknownInferredValue()
}

func (expr *VarRefExpr) InferValue(env *InferEnv) InferredValue {
	if expr.vr == nil || expr.vr.isDynamic {
		return unknownInferredValue()
	}
	if expr.vr.Value != nil {
		if fn, ok := expr.vr.Value.(*Fn); ok {
			return fnInferredValue(fn.fnExpr)
		}
		if _, ok := expr.vr.Value.(Callable); ok {
			return typeInferredValue(expr.vr.Value.GetType())
		}
		if len(expr.vr.taggedTypes) != 0 {
			return typesInferredValue(expr.vr.taggedTypes)
		}
	}
	if expr.vr.expr != nil {
		return expr.vr.expr.InferValue(env)
	}
	if len(expr.vr.taggedTypes) != 0 {
		return typesInferredValue(expr.vr.taggedTypes)
	}
	return unknownInferredValue()
}

func (expr *SetMacroExpr) InferValue(env *InferEnv) InferredValue {
	return unknownInferredValue()
}

func (expr *BindingExpr) InferValue(env *InferEnv) InferredValue {
	return bindingValue(expr.binding, env)
}

func (expr *MetaExpr) InferValue(env *InferEnv) InferredValue {
	return expr.expr.InferValue(env)
}

func (expr *DoExpr) InferValue(env *InferEnv) InferredValue {
	return inferBodyValue(expr.body, env)
}

func (expr *FnExpr) InferValue(env *InferEnv) InferredValue {
	return fnInferredValue(expr)
}

func (expr *FnArityExpr) InferValue(env *InferEnv) InferredValue {
	return unknownInferredValue()
}

func (expr *LetExpr) InferValue(env *InferEnv) InferredValue {
	for i, valueExpr := range expr.values {
		if i < len(expr.bindings) && expr.bindings[i] != nil {
			expr.bindings[i].inferredValue = nil
		}
		valueExpr.InferValue(env)
	}
	return inferBodyValue(expr.body, env)
}

func (expr *LoopExpr) InferValue(env *InferEnv) InferredValue {
	le := (*LetExpr)(expr)
	for i, valueExpr := range le.values {
		if i < len(le.bindings) && le.bindings[i] != nil {
			le.bindings[i].inferredValue = nil
		}
		valueExpr.InferValue(env)
	}
	return inferBodyValue(le.body, env)
}

func (expr *ThrowExpr) InferValue(env *InferEnv) InferredValue {
	return unknownInferredValue()
}

func (expr *CatchExpr) InferValue(env *InferEnv) InferredValue {
	return inferBodyValue(expr.body, env)
}

func (expr *TryExpr) InferValue(env *InferEnv) InferredValue {
	res := inferBodyValue(expr.body, env)
	for _, catchExpr := range expr.catches {
		res = joinInferredValues(res, catchExpr.InferValue(env))
	}
	for _, finallyExpr := range expr.finallyExpr {
		finallyExpr.InferValue(env)
	}
	return res
}

func callableFnSummary(callable Expr, passedArgsCount int) (*FnSummary, *FnAritySummary) {
	switch callable := callable.(type) {
	case *FnExpr:
		summary := getFnSummary(callable)
		return summary, summary.selectArity(passedArgsCount)
	case *MetaExpr:
		return callableFnSummary(callable.expr, passedArgsCount)
	case *VarRefExpr:
		if callable.vr == nil {
			return nil, nil
		}
		if fn, ok := callable.vr.Value.(*Fn); ok {
			summary := getFnSummary(fn.fnExpr)
			return summary, summary.selectArity(passedArgsCount)
		}
		if callable.vr.expr != nil {
			return callableFnSummary(callable.vr.expr, passedArgsCount)
		}
	case *BindingExpr:
		if callable.binding != nil && callable.binding.valueExpr != nil {
			return callableFnSummary(callable.binding.valueExpr, passedArgsCount)
		}
	}
	return nil, nil
}

func arglistTypes(vr *Var, passedArgsCount int) [][]*Type {
	if vr == nil {
		return nil
	}
	meta := vr.GetMeta()
	if meta == nil {
		return nil
	}
	ok, obj := meta.Get(KEYWORDS.arglist)
	if !ok {
		return nil
	}
	arglists, ok := obj.(Seq)
	if !ok {
		return nil
	}
	for !arglists.IsEmpty() {
		arglist, ok := arglists.First().(Vec)
		if !ok {
			arglists = arglists.Rest()
			continue
		}
		variadic := arglist.Count() >= 2 && arglist.At(arglist.Count()-2).Equals(SYMBOLS.amp)
		if (!variadic && arglist.Count() == passedArgsCount) ||
			(variadic && passedArgsCount >= arglist.Count()-2) {
			res := make([][]*Type, passedArgsCount)
			if variadic {
				for i := 0; i < arglist.Count()-2 && i < passedArgsCount; i++ {
					if sym, ok := arglist.At(i).(Symbol); ok {
						res[i] = getTaggedTypes(sym)
					}
				}
				if sym, ok := arglist.At(arglist.Count() - 1).(Symbol); ok {
					variadicTypes := getTaggedTypes(sym)
					for i := arglist.Count() - 2; i < passedArgsCount; i++ {
						res[i] = variadicTypes
					}
				}
			} else {
				for i := 0; i < passedArgsCount; i++ {
					if sym, ok := arglist.At(i).(Symbol); ok {
						res[i] = getTaggedTypes(sym)
					}
				}
			}
			return res
		}
		arglists = arglists.Rest()
	}
	return nil
}

func declaredArgTypesForCallable(callable Expr, passedArgsCount int) [][]*Type {
	switch callable := callable.(type) {
	case *MetaExpr:
		return declaredArgTypesForCallable(callable.expr, passedArgsCount)
	case *VarRefExpr:
		if callable.vr != nil {
			if _, ok := callable.vr.Value.(*Fn); ok {
				return nil
			}
			if _, ok := callable.vr.expr.(*FnExpr); ok {
				return nil
			}
		}
		return arglistTypes(callable.vr, passedArgsCount)
	case *BindingExpr:
		if callable.binding != nil && callable.binding.valueExpr != nil {
			return declaredArgTypesForCallable(callable.binding.valueExpr, passedArgsCount)
		}
	}
	return nil
}

func declaredReturnTypes(callable Expr, passedArgsCount int) []*Type {
	switch callable := callable.(type) {
	case *MetaExpr:
		return declaredReturnTypes(callable.expr, passedArgsCount)
	case *VarRefExpr:
		if callable.vr == nil {
			return nil
		}
		if fn, ok := callable.vr.Value.(*Fn); ok {
			if arity := selectActualArity(fn.fnExpr, passedArgsCount); arity != nil && len(arity.taggedTypes) != 0 {
				return arity.taggedTypes
			}
		}
		if len(callable.vr.taggedTypes) != 0 {
			return callable.vr.taggedTypes
		}
		if callable.vr.expr != nil {
			return declaredReturnTypes(callable.vr.expr, passedArgsCount)
		}
	case *FnExpr:
		if arity := selectActualArity(callable, passedArgsCount); arity != nil && len(arity.taggedTypes) != 0 {
			return arity.taggedTypes
		}
	case *BindingExpr:
		if callable.binding != nil && callable.binding.valueExpr != nil {
			return declaredReturnTypes(callable.binding.valueExpr, passedArgsCount)
		}
	}
	return nil
}

func (expr *CallExpr) InferValue(env *InferEnv) InferredValue {
	expr.callable.InferValue(env)
	for _, arg := range expr.args {
		arg.InferValue(env)
	}

	for i, expected := range declaredArgTypesForCallable(expr.callable, len(expr.args)) {
		if i < len(expr.args) {
			requireExpr(expr.args[i], expected, env)
		}
	}

	if _, aritySummary := callableFnSummary(expr.callable, len(expr.args)); aritySummary != nil {
		for i, expected := range aritySummary.declaredArgTypes {
			if i < len(expr.args) {
				requireExpr(expr.args[i], expected, env)
			}
		}
		for i, expected := range aritySummary.inferredArgTypes {
			if i < len(expr.args) {
				requireExpr(expr.args[i], expected, env)
			}
		}
		if declared := declaredReturnTypes(expr.callable, len(expr.args)); len(declared) != 0 {
			return typesInferredValue(declared)
		}
		return InferredValue{unknown: aritySummary.returnUnknown, types: aritySummary.returnTypes}
	}
	if declared := declaredReturnTypes(expr.callable, len(expr.args)); len(declared) != 0 {
		return typesInferredValue(declared)
	}
	return unknownInferredValue()
}

func selectActualArity(expr *FnExpr, passedArgsCount int) *FnArityExpr {
	for i := range expr.arities {
		if len(expr.arities[i].args) == passedArgsCount {
			return &expr.arities[i]
		}
	}
	if expr.variadic != nil && passedArgsCount >= len(expr.variadic.args)-1 {
		return expr.variadic
	}
	return nil
}

func getFnSummary(fn *FnExpr) *FnSummary {
	if fn.summary == nil {
		fn.summary = &FnSummary{fn: fn}
	}
	fn.summary.infer()
	return fn.summary
}

func (summary *FnSummary) selectArity(passedArgsCount int) *FnAritySummary {
	for _, arity := range summary.arities {
		if len(arity.arity.args) == passedArgsCount {
			return arity
		}
	}
	if summary.variadic != nil && passedArgsCount >= len(summary.variadic.arity.args)-1 {
		return summary.variadic
	}
	return nil
}

func (summary *FnSummary) infer() {
	if summary == nil || summary.analyzed || summary.analyzing {
		return
	}
	summary.analyzing = true
	defer func() {
		summary.analyzing = false
		summary.analyzed = true
	}()
	summary.arities = make([]*FnAritySummary, 0, len(summary.fn.arities))
	for i := range summary.fn.arities {
		summary.arities = append(summary.arities, inferFnArity(&summary.fn.arities[i], false))
	}
	if summary.fn.variadic != nil {
		summary.variadic = inferFnArity(summary.fn.variadic, true)
	}
}

func inferFnArity(arity *FnArityExpr, variadic bool) *FnAritySummary {
	for _, binding := range arity.bindings {
		if binding != nil {
			binding.inferredValue = nil
		}
	}
	env := newInferEnv()
	value := inferBodyValue(arity.body, env)
	if arity.stubReturnUnknown {
		value = unknownInferredValue()
	}
	res := &FnAritySummary{
		arity:            arity,
		variadic:         variadic,
		returnUnknown:    value.unknown,
		returnTypes:      value.types,
		returnArgDeps:    returnArgDeps(arity.body, arity.bindings),
		inferredArgTypes: make([][]*Type, len(arity.args)),
		declaredArgTypes: make([][]*Type, len(arity.args)),
	}
	for _, taggedType := range arity.taggedTypes {
		res.returnTypes = addType(res.returnTypes, taggedType)
	}
	for i, arg := range arity.args {
		res.declaredArgTypes[i] = getTaggedTypes(arg)
		if i < len(arity.bindings) && arity.bindings[i] != nil {
			res.inferredArgTypes[i] = env.requiredTypes[arity.bindings[i]]
		}
	}
	if variadic && len(arity.args) > 0 {
		res.inferredArgTypes[len(arity.args)-1] = nil
		res.declaredArgTypes[len(arity.args)-1] = nil
	}
	return res
}

func returnArgDeps(body []Expr, args []*Binding) []bool {
	res := make([]bool, len(args))
	if len(body) == 0 {
		return res
	}
	returnExprArgDeps(body[len(body)-1], args, res, map[*Binding]bool{})
	return res
}

func returnExprArgDeps(expr Expr, args []*Binding, res []bool, seen map[*Binding]bool) {
	switch expr := expr.(type) {
	case *BindingExpr:
		for i, arg := range args {
			if expr.binding == arg {
				res[i] = true
				return
			}
		}
		if expr.binding != nil && expr.binding.valueExpr != nil && !seen[expr.binding] {
			seen[expr.binding] = true
			returnExprArgDeps(expr.binding.valueExpr, args, res, seen)
		}
	case *MetaExpr:
		returnExprArgDeps(expr.expr, args, res, seen)
	case *DoExpr:
		if len(expr.body) > 0 {
			returnExprArgDeps(expr.body[len(expr.body)-1], args, res, seen)
		}
	case *IfExpr:
		returnExprArgDeps(expr.positive, args, res, seen)
		returnExprArgDeps(expr.negative, args, res, seen)
	case *LetExpr:
		if len(expr.body) > 0 {
			returnExprArgDeps(expr.body[len(expr.body)-1], args, res, seen)
		}
	case *LoopExpr:
		le := (*LetExpr)(expr)
		if len(le.body) > 0 {
			returnExprArgDeps(le.body[len(le.body)-1], args, res, seen)
		}
	case *TryExpr:
		if len(expr.body) > 0 {
			returnExprArgDeps(expr.body[len(expr.body)-1], args, res, seen)
		}
		for _, catchExpr := range expr.catches {
			if len(catchExpr.body) > 0 {
				returnExprArgDeps(catchExpr.body[len(catchExpr.body)-1], args, res, seen)
			}
		}
	case *CallExpr:
		if !shouldCheckInferredSummary(expr.callable) {
			return
		}
		_, arity := callableFnSummary(expr.callable, len(expr.args))
		if arity == nil {
			return
		}
		for i, dep := range arity.returnArgDeps {
			if dep && i < len(expr.args) {
				returnExprArgDeps(expr.args[i], args, res, seen)
			}
		}
	}
}

func shouldCheckInferredSummary(expr Expr) bool {
	switch expr := expr.(type) {
	case *MetaExpr:
		return shouldCheckInferredSummary(expr.expr)
	case *BindingExpr, *FnExpr:
		return true
	case *VarRefExpr:
		return expr.vr != nil && expr.vr.ns != GLOBAL_ENV.CoreNamespace
	}
	return false
}

func checkInferredCall(call *CallExpr) bool {
	_, arity := callableFnSummary(call.callable, len(call.args))
	res := false
	checkExpected := func(expected [][]*Type) {
		for i, expectedTypes := range expected {
			if len(expectedTypes) == 0 || i >= len(call.args) {
				continue
			}
			passedValue := call.args[i].InferValue(newInferEnv())
			if !passedValue.unknown && len(passedValue.types) != 0 && !inferredTypesCompatible(expectedTypes, passedValue.types) {
				printParseWarning(call.args[i].Pos(), fmt.Sprintf("arg[%d] of %s must have type %s, got %s", i, call.Name(), inferredTypesString(expectedTypes), inferredTypesString(passedValue.types)))
				res = true
			}
		}
	}
	if arity != nil && shouldCheckInferredSummary(call.callable) {
		checkExpected(arity.inferredArgTypes)
	}
	if arity != nil {
		checkExpected(arity.declaredArgTypes)
	}
	if declaredArgTypes := declaredArgTypesForCallable(call.callable, len(call.args)); len(declaredArgTypes) > 0 {
		checkExpected(declaredArgTypes)
	}
	return res
}
