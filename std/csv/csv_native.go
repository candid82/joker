package csv

import (
	"encoding/csv"
	"unsafe"

	. "github.com/candid82/joker/core"
)

var csvReaderType *Type = RegRefType("CSVReader", (*CSVReader)(nil))

type (
	CSVReader struct {
		*csv.Reader
	}
)

func (rdr *CSVReader) ToString(escape bool) string {
	return "#object[CSVReader]"
}

func (rdr *CSVReader) Equals(other interface{}) bool {
	return rdr == other
}

func (rdr *CSVReader) GetInfo() *ObjectInfo {
	return nil
}

func (rdr *CSVReader) GetType() *Type {
	return csvReaderType
}

func (rdr *CSVReader) Hash() uint32 {
	return HashPtr(uintptr(unsafe.Pointer(rdr)))
}

func (rdr *CSVReader) WithInfo(info *ObjectInfo) Object {
	return rdr
}

func MakeCSVReader(r *csv.Reader) *CSVReader {
	return &CSVReader{r}
}

func EnsureCSVReader(args []Object, index int) *CSVReader {
	switch c := args[index].(type) {
	case *CSVReader:
		return c
	default:
		panic(RT.NewArgTypeError(index, c, "CSVReader"))
	}
}

func ExtractCSVReader(args []Object, index int) *CSVReader {
	return EnsureCSVReader(args, index)
}
