package core

// These values already have an Object interface box. Use them only for fresh
// runtime results with no source information; reader values retain their info.
var (
	trueObject  Object = Boolean{B: true}
	falseObject Object = Boolean{B: false}
	smallInts          = func() [256]Object {
		var values [256]Object
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

func boxInt(value int) Object {
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
