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
	csvReader.ReuseRecord = true
	return csvLazySeq(csvReader)
}

func csvSeqOpts(rdr io.Reader, opts Map) Object {
	csvReader := csv.NewReader(rdr)
	csvReader.ReuseRecord = true
	if ok, c := opts.Get(MakeKeyword("comma")); ok {
		csvReader.Comma = AssertChar(c, "comma must be a char").Ch
	}
	if ok, c := opts.Get(MakeKeyword("comment")); ok {
		csvReader.Comment = AssertChar(c, "comment must be a char").Ch
	}
	if ok, c := opts.Get(MakeKeyword("fields-per-record")); ok {
		csvReader.FieldsPerRecord = AssertInt(c, "fields-per-record must be an integer").I
	}
	if ok, c := opts.Get(MakeKeyword("lazy-quotes")); ok {
		csvReader.LazyQuotes = AssertBoolean(c, "lazy-quotes must be an boolean").B
	}
	if ok, c := opts.Get(MakeKeyword("trim-leading-space")); ok {
		csvReader.TrimLeadingSpace = AssertBoolean(c, "trim-leading-space must be an boolean").B
	}
	return csvLazySeq(csvReader)
}
