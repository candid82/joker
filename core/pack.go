package core

import (
	"encoding/binary"
	"fmt"
	"maps"
	"slices"
	"sort"
)

const (
	NULL     = 100
	NOT_NULL = 101
)

type (
	PackEnv struct {
		Strings         map[*string]uint16
		nextStringIndex uint16
		objects         map[Object]int
		packing         map[Object]bool
	}

	PackHeader struct {
		GlobalEnv *Env
		Strings   []*string
		objects   []Object
	}
)

func NewPackEnv() *PackEnv {
	return &PackEnv{
		Strings: make(map[*string]uint16),
	}
}

type ByString []*string

func (a ByString) Len() int      { return len(a) }
func (a ByString) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByString) Less(i, j int) bool {
	if a[i] == nil {
		return true
	}
	if a[j] == nil {
		return false
	}
	return *a[i] < *a[j]
}

// Packed data is an internal, rebuildable format. Reject stale blobs explicitly.
const packedVersion = "JOKER-VM\x04"

func (env *PackEnv) Pack(p []byte) []byte {
	p = append(p, packedVersion...)
	p = appendInt(p, len(env.Strings))
	stringKeys := slices.Collect(maps.Keys(env.Strings))
	sort.Sort(ByString(stringKeys))
	for _, k := range stringKeys {
		p = appendUint16(p, env.Strings[k])
		if k == nil {
			p = appendInt(p, -1)
		} else {
			p = appendInt(p, len(*k))
			p = append(p, *k...)
		}
	}
	return p
}

func UnpackHeader(p []byte, env *Env) (*PackHeader, []byte) {
	defer packedRecovery()
	if len(p) < len(packedVersion) || string(p[:len(packedVersion)]) != packedVersion {
		panic(RT.NewError("Incompatible packed bytecode; regenerate embedded data"))
	}
	p = p[len(packedVersion):]
	stringCount, p := extractCount(p)
	strs := make([]*string, stringCount)
	for i := 0; i < stringCount; i++ {
		var index uint16
		var length int
		index, p = extractUInt16(p)
		length, p = extractInt(p)
		if int(index) >= len(strs) {
			panic(RT.NewError("Invalid packed string index"))
		}
		if length == -1 {
			strs[index] = nil
		} else {
			if length < 0 || length > len(p) {
				panic(RT.NewError("Invalid packed string length"))
			}
			strs[index] = STRINGS.Intern(string(p[:length]))
			p = p[length:]
		}
	}
	header := &PackHeader{
		GlobalEnv: env,
		Strings:   strs,
	}
	return header, p
}

func (env *PackEnv) stringIndex(s *string) uint16 {
	index, ok := env.Strings[s]
	if ok {
		return index
	}
	if len(env.Strings) >= 65536 {
		panic(RT.NewError("Too many strings in packed data"))
	}
	env.Strings[s] = env.nextStringIndex
	env.nextStringIndex++
	return env.nextStringIndex - 1
}

func appendBool(p []byte, b bool) []byte {
	var bb byte
	if b {
		bb = 1
	}
	return append(p, bb)
}

func extractBool(p []byte) (bool, []byte) {
	if len(p) == 0 || p[0] > 1 {
		panic(RT.NewError("Invalid packed boolean"))
	}
	var b bool
	if p[0] == 1 {
		b = true
	}
	return b, p[1:]
}

func appendUint16(p []byte, i uint16) []byte {
	pp := make([]byte, 2)
	binary.BigEndian.PutUint16(pp, i)
	p = append(p, pp...)
	return p
}

func extractUInt16(p []byte) (uint16, []byte) {
	return binary.BigEndian.Uint16(p[0:2]), p[2:]
}

func appendUint32(p []byte, i uint32) []byte {
	pp := make([]byte, 4)
	binary.BigEndian.PutUint32(pp, i)
	p = append(p, pp...)
	return p
}

func extractUInt32(p []byte) (uint32, []byte) {
	return binary.BigEndian.Uint32(p[0:4]), p[4:]
}

func appendInt(p []byte, i int) []byte {
	pp := make([]byte, 8)
	binary.BigEndian.PutUint64(pp, uint64(i))
	p = append(p, pp...)
	return p
}

func extractInt(p []byte) (int, []byte) {
	if len(p) < 8 {
		panic(RT.NewError("Truncated packed integer"))
	}
	return int(binary.BigEndian.Uint64(p[0:8])), p[8:]
}

// Counts are bounded by the remaining input before allocating. Every encoded
// element consumes at least one byte. Compressed position tables are checked
// against code length separately.
func extractCount(p []byte) (int, []byte) {
	n, rest := extractInt(p)
	if n < 0 || n > len(rest) {
		panic(RT.NewError("Invalid packed element count"))
	}
	return n, rest
}

func packedRecovery() {
	if r := recover(); r != nil {
		if _, ok := r.(Error); ok {
			panic(r)
		}
		panic(RT.NewError(fmt.Sprintf("Malformed packed bytecode: %v", r)))
	}
}

func (pos Position) Pack(p []byte, env *PackEnv) []byte {
	p = appendInt(p, pos.startLine)
	p = appendInt(p, pos.endLine)
	p = appendInt(p, pos.startColumn)
	p = appendInt(p, pos.endColumn)
	p = appendUint16(p, env.stringIndex(pos.filename))
	return p
}

func unpackPosition(p []byte, header *PackHeader) (pos Position, pp []byte) {
	pos.startLine, p = extractInt(p)
	pos.endLine, p = extractInt(p)
	pos.startColumn, p = extractInt(p)
	pos.endColumn, p = extractInt(p)
	i, p := extractUInt16(p)
	pos.filename = header.Strings[i]
	return pos, p
}

func (info *ObjectInfo) Pack(p []byte, env *PackEnv) []byte {
	if info == nil {
		return append(p, NULL)
	}
	p = append(p, NOT_NULL)
	return info.Pos().Pack(p, env)
}

func unpackObjectInfo(p []byte, header *PackHeader) (*ObjectInfo, []byte) {
	if p[0] == NULL {
		return nil, p[1:]
	}
	p = p[1:]
	pos, p := unpackPosition(p, header)
	return &ObjectInfo{Position: pos}, p
}

func PackObjectOrNull(obj Object, p []byte, env *PackEnv) []byte {
	if obj == nil {
		return append(p, NULL)
	}
	p = append(p, NOT_NULL)
	return packObject(obj, p, env)
}

func UnpackObjectOrNull(p []byte, header *PackHeader) (Object, []byte) {
	if p[0] == NULL {
		return nil, p[1:]
	}
	return unpackObject(p[1:], header)
}

func (s Symbol) Pack(p []byte, env *PackEnv) []byte {
	p = s.info.Pack(p, env)
	p = PackObjectOrNull(s.meta, p, env)
	p = appendUint16(p, env.stringIndex(s.name))
	p = appendUint16(p, env.stringIndex(s.ns))
	p = appendUint32(p, s.hash)
	return p
}

func unpackSymbol(p []byte, header *PackHeader) (Symbol, []byte) {
	info, p := unpackObjectInfo(p, header)
	meta, p := UnpackObjectOrNull(p, header)
	iname, p := extractUInt16(p)
	ins, p := extractUInt16(p)
	hash, p := extractUInt32(p)
	res := Symbol{
		InfoHolder: InfoHolder{info: info},
		name:       header.Strings[iname],
		ns:         header.Strings[ins],
		hash:       hash,
	}
	if meta != nil {
		res.meta = meta.(Map)
	}
	return res, p
}

func (t *Type) Pack(p []byte, env *PackEnv) []byte {
	s := MakeSymbol(t.name)
	return s.Pack(p, env)
}

func unpackType(p []byte, header *PackHeader) (*Type, []byte) {
	s, p := unpackSymbol(p, header)
	t := TYPES[s.name]
	if t == nil {
		panic(RT.NewError("Unknown type in packed data: " + s.ToString(false)))
	}
	return t, p
}

func (vr *Var) Pack(p []byte, env *PackEnv) []byte {
	p = vr.ns.Name.Pack(p, env)
	p = vr.name.Pack(p, env)
	return p
}

func unpackVar(p []byte, header *PackHeader) (*Var, []byte) {
	nsName, p := unpackSymbol(p, header)
	name, p := unpackSymbol(p, header)
	ns := header.GlobalEnv.FindNamespace(nsName)
	if ns == nil {
		panic(RT.NewError("Error unpacking var: cannot find namespace " + *nsName.name))
	}
	vr := ns.mappings[name.name]
	if vr == nil {
		// Var doesn't exist yet — intern it (will be defined when bytecode executes)
		varName := name
		varName.ns = nil
		vr = ns.Intern(varName)
	}
	return vr, p
}

// --- FunctionProto serialization ---

func packUpvalueInfo(p []byte, u UpvalueInfo) []byte {
	p = appendInt(p, u.Index)
	p = appendBool(p, u.IsLocal)
	return p
}

func unpackUpvalueInfo(p []byte) (UpvalueInfo, []byte) {
	index, p := extractInt(p)
	isLocal, p := extractBool(p)
	return UpvalueInfo{Index: index, IsLocal: isLocal}, p
}

func packCatchInfo(p []byte, c CatchInfo, env *PackEnv) []byte {
	if c.ExcType != nil {
		p = append(p, NOT_NULL)
		p = c.ExcType.Pack(p, env)
	} else {
		p = append(p, NULL)
	}
	p = appendInt(p, c.HandlerIP)
	p = appendInt(p, c.LocalSlot)
	return p
}

func unpackCatchInfo(p []byte, header *PackHeader) (CatchInfo, []byte) {
	var excType *Type
	if p[0] == NOT_NULL {
		p = p[1:]
		excType, p = unpackType(p, header)
	} else {
		p = p[1:]
	}
	handlerIP, p := extractInt(p)
	localSlot, p := extractInt(p)
	return CatchInfo{
		ExcType:   excType,
		HandlerIP: handlerIP,
		LocalSlot: localSlot,
	}, p
}

func packHandlerInfo(p []byte, h HandlerInfo, env *PackEnv) []byte {
	p = appendInt(p, len(h.Catches))
	for _, c := range h.Catches {
		p = packCatchInfo(p, c, env)
	}
	p = appendInt(p, h.FinallyIP)
	p = appendInt(p, h.EndIP)
	p = appendInt(p, h.TryLocalCount)
	return p
}

func unpackHandlerInfo(p []byte, header *PackHeader) (HandlerInfo, []byte) {
	catchCount, p := extractCount(p)
	catches := make([]CatchInfo, catchCount)
	for i := 0; i < catchCount; i++ {
		catches[i], p = unpackCatchInfo(p, header)
	}
	finallyIP, p := extractInt(p)
	endIP, p := extractInt(p)
	tryLocalCount, p := extractInt(p)
	return HandlerInfo{
		Catches:       catches,
		FinallyIP:     finallyIP,
		EndIP:         endIP,
		TryLocalCount: tryLocalCount,
	}, p
}

func (c *Chunk) Pack(p []byte, env *PackEnv) []byte {
	// Code
	p = appendInt(p, len(c.Code))
	p = append(p, c.Code...)
	// Constants
	p = appendInt(p, len(c.Constants))
	for _, obj := range c.Constants {
		p = packObject(obj, p, env)
	}
	// Source positions: adjacent opcode/operand bytes usually share one position.
	p = appendInt(p, len(c.Positions))
	for i := 0; i < len(c.Positions); {
		j := i + 1
		for j < len(c.Positions) && c.Positions[j] == c.Positions[i] {
			j++
		}
		p = appendInt(p, j-i)
		p = c.Positions[i].Pack(p, env)
		i = j
	}
	// Native call sites remain immutable across VM executions.
	p = appendInt(p, len(c.callSites))
	for ip := range c.Code {
		if c.callSites[ip] != nil {
			p = appendInt(p, ip)
			p = packBytes(p, []byte(c.callSites[ip].callName))
		}
	}
	// Handlers
	p = appendInt(p, len(c.Handlers))
	for _, h := range c.Handlers {
		p = packHandlerInfo(p, h, env)
	}
	return p
}

func unpackChunk(p []byte, header *PackHeader) (*Chunk, []byte) {
	codeLen, p := extractCount(p)
	code := make([]byte, codeLen)
	copy(code, p[:codeLen])
	p = p[codeLen:]

	constCount, p := extractCount(p)
	constants := make([]Object, constCount)
	for i := 0; i < constCount; i++ {
		constants[i], p = unpackObject(p, header)
	}

	positionCount, p := extractInt(p)
	if positionCount != codeLen {
		panic(RT.NewError("Invalid packed position count"))
	}
	positions := make([]Position, positionCount)
	for i := 0; i < positionCount; {
		var count int
		count, p = extractInt(p)
		if count <= 0 || count > positionCount-i {
			panic(RT.NewError("Invalid packed position run"))
		}
		var pos Position
		pos, p = unpackPosition(p, header)
		for j := 0; j < count; j++ {
			positions[i+j] = pos
		}
		i += count
	}
	callCount, p := extractCount(p)
	var callSites map[int]*CallExpr
	if callCount > 0 {
		callSites = make(map[int]*CallExpr, callCount)
	}
	for i := 0; i < callCount; i++ {
		var ip int
		ip, p = extractInt(p)
		var name []byte
		name, p = unpackBytes(p)
		callSites[ip] = &CallExpr{Position: positions[ip], callName: string(name)}
	}

	handlerCount, p := extractCount(p)
	handlers := make([]HandlerInfo, handlerCount)
	for i := 0; i < handlerCount; i++ {
		handlers[i], p = unpackHandlerInfo(p, header)
	}

	return &Chunk{
		Code:      code,
		Constants: constants,
		Positions: positions,
		callSites: callSites,
		Handlers:  handlers,
	}, p
}

func packTypePtr(p []byte, t *Type, env *PackEnv) []byte {
	if t == nil {
		return append(p, NULL)
	}
	p = append(p, NOT_NULL)
	return t.Pack(p, env)
}

func unpackTypePtr(p []byte, header *PackHeader) (*Type, []byte) {
	if p[0] == NULL {
		return nil, p[1:]
	}
	p = p[1:]
	t, p := unpackType(p, header)
	return t, p
}

func packArgTypes(p []byte, argTypes [][]*Type, env *PackEnv) []byte {
	if argTypes == nil {
		p = appendInt(p, 0)
		return p
	}
	p = appendInt(p, len(argTypes))
	for _, types := range argTypes {
		p = appendInt(p, len(types))
		for _, t := range types {
			p = t.Pack(p, env)
		}
	}
	return p
}

func unpackArgTypes(p []byte, header *PackHeader) ([][]*Type, []byte) {
	count, p := extractCount(p)
	if count == 0 {
		return nil, p
	}
	argTypes := make([][]*Type, count)
	for i := 0; i < count; i++ {
		typeCount, pp := extractCount(p)
		p = pp
		if typeCount > 0 {
			argTypes[i] = make([]*Type, typeCount)
			for j := 0; j < typeCount; j++ {
				argTypes[i][j], p = unpackType(p, header)
			}
		}
	}
	return argTypes, p
}

func (a *ArityProto) Pack(p []byte, env *PackEnv) []byte {
	p = appendInt(p, a.Arity)
	p = appendBool(p, a.IsVariadic)
	p = a.Chunk.Pack(p, env)
	// SubFunctions
	p = appendInt(p, len(a.SubFunctions))
	for _, sub := range a.SubFunctions {
		p = sub.Pack(p, env)
	}
	// ArgTypes
	p = packArgTypes(p, a.ArgTypes, env)
	// TaggedType
	p = packTypePtr(p, a.TaggedType, env)
	return p
}

func unpackArityProto(p []byte, header *PackHeader) (*ArityProto, []byte) {
	arity, p := extractInt(p)
	isVariadic, p := extractBool(p)
	chunk, p := unpackChunk(p, header)

	subCount, p := extractCount(p)
	subFunctions := make([]*FunctionProto, subCount)
	for i := 0; i < subCount; i++ {
		subFunctions[i], p = unpackFunctionProto(p, header)
	}

	// ArgTypes
	argTypes, p := unpackArgTypes(p, header)
	// TaggedType
	taggedType, p := unpackTypePtr(p, header)

	return &ArityProto{
		Arity:        arity,
		IsVariadic:   isVariadic,
		Chunk:        chunk,
		SubFunctions: subFunctions,
		ArgTypes:     argTypes,
		TaggedType:   taggedType,
	}, p
}

func (proto *FunctionProto) Pack(p []byte, env *PackEnv) []byte {
	// Name
	s := STRINGS.Intern(proto.Name)
	p = appendUint16(p, env.stringIndex(s))
	// Arities
	p = appendInt(p, len(proto.Arities))
	for _, a := range proto.Arities {
		p = a.Pack(p, env)
	}
	// VariadicArity
	if proto.VariadicArity != nil {
		p = append(p, NOT_NULL)
		p = proto.VariadicArity.Pack(p, env)
	} else {
		p = append(p, NULL)
	}
	// Optional top-level body (functions use arities instead).
	if proto.Chunk != nil {
		p = append(p, NOT_NULL)
		p = proto.Chunk.Pack(p, env)
	} else {
		p = append(p, NULL)
	}
	p = appendInt(p, len(proto.Upvalues))
	for _, u := range proto.Upvalues {
		p = packUpvalueInfo(p, u)
	}
	p = appendInt(p, len(proto.SubFunctions))
	for _, sub := range proto.SubFunctions {
		p = sub.Pack(p, env)
	}
	return p
}

func UnpackFunctionProto(p []byte, header *PackHeader) (*FunctionProto, []byte) {
	defer packedRecovery()
	proto, rest := unpackFunctionProto(p, header)
	if err := ValidateFunctionProto(proto); err != nil {
		panic(RT.NewError("Invalid packed bytecode: " + err.Error()))
	}
	return proto, rest
}

func unpackFunctionProto(p []byte, header *PackHeader) (*FunctionProto, []byte) {
	nameIdx, p := extractUInt16(p)
	name := ""
	if int(nameIdx) < len(header.Strings) && header.Strings[nameIdx] != nil {
		name = *header.Strings[nameIdx]
	}

	arityCount, p := extractCount(p)
	arities := make([]*ArityProto, arityCount)
	for i := 0; i < arityCount; i++ {
		arities[i], p = unpackArityProto(p, header)
	}

	var variadicArity *ArityProto
	if p[0] == NOT_NULL {
		p = p[1:]
		variadicArity, p = unpackArityProto(p, header)
	} else {
		p = p[1:]
	}

	var topLevelChunk *Chunk
	if p[0] == NOT_NULL {
		p = p[1:]
		topLevelChunk, p = unpackChunk(p, header)
	} else {
		p = p[1:]
	}
	upvalueCount, p := extractCount(p)
	upvalues := make([]UpvalueInfo, upvalueCount)
	for i := 0; i < upvalueCount; i++ {
		upvalues[i], p = unpackUpvalueInfo(p)
	}
	subCount, p := extractCount(p)
	subFunctions := make([]*FunctionProto, subCount)
	for i := 0; i < subCount; i++ {
		subFunctions[i], p = unpackFunctionProto(p, header)
	}

	return &FunctionProto{
		Name:          name,
		Arities:       arities,
		VariadicArity: variadicArity,
		Chunk:         topLevelChunk,
		Upvalues:      upvalues,
		SubFunctions:  subFunctions,
	}, p
}
