package csv

import (
	"encoding/csv"
	"io"

	. "github.com/candid82/joker/core"
)

func csvLazySeq(rdr *csv.Reader) *LazySeq {
	var c Proc = func(args []Object) Object {
		t, err := rdr.Read()
		if err == io.EOF {
			return EmptyList
		}
		PanicOnErr(err)
		return NewConsSeq(MakeStringVector(t), csvLazySeq(rdr))
	}
	return NewLazySeq(c)
}

func csvSeq(rdr io.Reader) Object {
	csvReader := csv.NewReader(rdr)
	return csvLazySeq(csvReader)
}
