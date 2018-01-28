package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/candid82/joker/core"
)

var client = &http.Client{}

func extractMethod(request Map) string {
	if ok, m := request.Get(MakeKeyword("method")); ok {
		switch m := m.(type) {
		case String:
			return m.S
		case Keyword:
			return m.ToString(false)[1:]
		case Symbol:
			return m.ToString(false)
		default:
			panic(RT.NewError(fmt.Sprintf("method must be a string, keyword or symbol, got %s", m.GetType().ToString(false))))
		}
	}
	return "get"
}

func getOrPanic(m Map, k Object, errMsg string) Object {
	if ok, v := m.Get(k); ok {
		return v
	}
	panic(RT.NewError(errMsg))
}

func sendRequest(request Map) Map {
	method := strings.ToUpper(extractMethod(request))
	url := AssertString(getOrPanic(request, MakeKeyword("url"), ":url key must be present in request map"), "url must be a string").S
	var reqBody io.Reader
	if ok, b := request.Get(MakeKeyword("body")); ok {
		reqBody = strings.NewReader(AssertString(b, "body must be a string").S)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	if ok, headers := request.Get(MakeKeyword("headers")); ok {
		h := AssertMap(headers, "headers must be a map")
		for iter := h.Iter(); iter.HasNext(); {
			p := iter.Next()
			req.Header.Add(AssertString(p.Key, "header name must be a string").S, AssertString(p.Value, "header value must be a string").S)
		}
	}
	if ok, host := request.Get(MakeKeyword("host")); ok {
		req.Host = AssertString(host, "host must be a string").S
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	defer resp.Body.Close()
	res := EmptyArrayMap()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	res.Add(MakeKeyword("body"), String{S: string(body)})
	return res
}
