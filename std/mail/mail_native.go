package mail

import (
	"io"
	netmail "net/mail"
	"strings"
	"time"

	. "github.com/candid82/joker/core"
)

func mailReader(source Object) io.Reader {
	switch source := source.(type) {
	case String:
		return strings.NewReader(source.S)
	case io.Reader:
		return source
	default:
		panic(RT.NewError("source must be a string or io.Reader"))
	}
}

func messageHeaders(headers netmail.Header) Map {
	res := EmptyArrayMap()
	for key, values := range headers {
		res.Add(MakeString(key), MakeStringVector(values))
	}
	return res
}

func readMessage(source Object) Map {
	message, err := netmail.ReadMessage(mailReader(source))
	PanicOnErr(err)
	body, err := io.ReadAll(message.Body)
	PanicOnErr(err)

	res := EmptyArrayMap()
	res.Add(MakeKeyword("headers"), messageHeaders(message.Header))
	res.Add(MakeKeyword("body"), MakeString(string(body)))
	return res
}

func addressMap(address *netmail.Address) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("name"), MakeString(address.Name))
	res.Add(MakeKeyword("address"), MakeString(address.Address))
	return res
}

func parseAddress(s string) Map {
	address, err := netmail.ParseAddress(s)
	PanicOnErr(err)
	return addressMap(address)
}

func parseAddressList(s string) Object {
	addresses, err := netmail.ParseAddressList(s)
	PanicOnErr(err)
	res := EmptyVector()
	for _, address := range addresses {
		res = res.Conjoin(addressMap(address))
	}
	return res
}

func parseDate(s string) time.Time {
	t, err := netmail.ParseDate(s)
	PanicOnErr(err)
	return t
}
