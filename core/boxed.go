package core

// These values are already interface-boxed. Use them only for fresh
// runtime results with no source information; reader values retain their info.
var (
	trueObject  Object = Boolean{B: true}
	falseObject Object = Boolean{B: false}
	// Number entries let arithmetic return the existing box directly. The
	// bounded cache adds about 84 KiB on 64-bit systems over the former
	// 0..255 cache.
	smallInts = func() [2048]Number {
		var values [2048]Number
		for i := range values {
			values[i] = Int{I: i}
		}
		return values
	}()
	smallChars = func() [128]Object {
		var values [128]Object
		for i := range values {
			values[i] = Char{Ch: rune(i)}
		}
		return values
	}()
)

func boxBoolean(value bool) Object {
	if value {
		return trueObject
	}
	return falseObject
}

func boxInt(value int) Number {
	if uint(value) < uint(len(smallInts)) {
		return smallInts[value]
	}
	return Int{I: value}
}

func boxChar(value rune) Object {
	if uint32(value) < uint32(len(smallChars)) {
		return smallChars[value]
	}
	return Char{Ch: value}
}
