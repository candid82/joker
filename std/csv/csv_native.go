package csv

import (
	"encoding/csv"
	"io"
	"strings"

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

func csvSeqOpts(src Object, opts Map) Object {
	var rdr io.Reader
	switch src := src.(type) {
	case String:
		rdr = strings.NewReader(src.S)
	case io.Reader:
		rdr = src
	default:
		panic(RT.NewError("src must be a string or io.Reader"))
	}
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
		csvReader.LazyQuotes = AssertBoolean(c, "lazy-quotes must be a boolean").B
	}
	if ok, c := opts.Get(MakeKeyword("trim-leading-space")); ok {
		csvReader.TrimLeadingSpace = AssertBoolean(c, "trim-leading-space must be a boolean").B
	}
	return csvLazySeq(csvReader)
}

func sliceOfStrings(obj Object) (res []string) {
	s := AssertSeqable(obj, "CSV record must be Seqable").Seq()
	for !s.IsEmpty() {
		res = append(res, s.First().ToString(false))
		s = s.Rest()
	}
	return
}

func writeWriter(wr io.Writer, data Seqable, opts Map) {
	csvWriter := csv.NewWriter(wr)
	if ok, c := opts.Get(MakeKeyword("comma")); ok {
		csvWriter.Comma = AssertChar(c, "comma must be a char").Ch
	}
	if ok, c := opts.Get(MakeKeyword("use-crlf")); ok {
		csvWriter.UseCRLF = AssertBoolean(c, "use-crlf must be a boolean").B
	}
	s := data.Seq()
	for !s.IsEmpty() {
		err := csvWriter.Write(sliceOfStrings(s.First()))
		PanicOnErr(err)
		s = s.Rest()
	}
	csvWriter.Flush()
}

func write(wr io.Writer, data Seqable, opts Map) Object {
	writeWriter(wr, data, opts)
	return NIL
}

func writeString(data Seqable, opts Map) string {
	var b strings.Builder
	writeWriter(&b, data, opts)
	return b.String()
}
