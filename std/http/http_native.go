package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
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

func mapToReq(request Map) *http.Request {
	method := strings.ToUpper(extractMethod(request))
	url := EnsureObjectIsString(getOrPanic(request, MakeKeyword("url"), ":url key must be present in request map"), "url: %s").S
	var reqBody io.Reader
	if ok, b := request.Get(MakeKeyword("body")); ok {
		reqBody = strings.NewReader(EnsureObjectIsString(b, "body: %s").S)
	}
	req, err := http.NewRequest(method, url, reqBody)
	PanicOnErr(err)
	if ok, headers := request.Get(MakeKeyword("headers")); ok {
		h := EnsureObjectIsMap(headers, "headers: %s")
		for iter := h.Iter(); iter.HasNext(); {
			p := iter.Next()
			req.Header.Add(EnsureObjectIsString(p.Key, "header name: %s").S, EnsureObjectIsString(p.Value, "header value: %s").S)
		}
	}
	if ok, host := request.Get(MakeKeyword("host")); ok {
		req.Host = EnsureObjectIsString(host, "host: %s").S
	}
	return req
}

func reqToMap(host String, port String, req *http.Request) Map {
	defer req.Body.Close()
	res := EmptyArrayMap()
	body, err := ioutil.ReadAll(req.Body)
	PanicOnErr(err)
	res.Add(MakeKeyword("request-method"), MakeKeyword(strings.ToLower(req.Method)))
	res.Add(MakeKeyword("body"), MakeString(string(body)))
	res.Add(MakeKeyword("uri"), MakeString(req.URL.Path))
	res.Add(MakeKeyword("query-string"), MakeString(req.URL.RawQuery))
	res.Add(MakeKeyword("server-name"), host)
	res.Add(MakeKeyword("server-port"), port)
	res.Add(MakeKeyword("remote-addr"), MakeString(req.RemoteAddr[:strings.LastIndexByte(req.RemoteAddr, byte(':'))]))
	res.Add(MakeKeyword("protocol"), MakeString(req.Proto))
	res.Add(MakeKeyword("scheme"), MakeKeyword("http"))
	res.Add(MakeKeyword("host"), MakeString(req.Host))
	headers := EmptyArrayMap()
	for k, v := range req.Header {
		headers.Add(MakeString(strings.ToLower(k)), MakeString(strings.Join(v, ",")))
	}
	res.Add(MakeKeyword("headers"), headers)
	return res
}

func respToMap(resp *http.Response) Map {
	defer resp.Body.Close()
	res := EmptyArrayMap()
	body, err := ioutil.ReadAll(resp.Body)
	PanicOnErr(err)
	res.Add(MakeKeyword("body"), MakeString(string(body)))
	res.Add(MakeKeyword("status"), MakeInt(resp.StatusCode))
	respHeaders := EmptyArrayMap()
	for k, v := range resp.Header {
		respHeaders.Add(MakeString(k), MakeStringVector(v))
	}
	res.Add(MakeKeyword("headers"), respHeaders)
	// TODO: 32-bit issue
	res.Add(MakeKeyword("content-length"), MakeInt(int(resp.ContentLength)))
	return res
}

func addHeaders(headers Object, w http.ResponseWriter) {
	header := w.Header()
	h := EnsureObjectIsMap(headers, "HTTP response headers: %s")
	for iter := h.Iter(); iter.HasNext(); {
		p := iter.Next()
		hname := EnsureObjectIsString(p.Key, "HTTP response header name %s").S
		switch pvalue := p.Value.(type) {
		case String:
			header.Add(hname, pvalue.S)
		case Seqable:
			s := pvalue.Seq()
			for !s.IsEmpty() {
				header.Add(hname, EnsureObjectIsString(s.First(), "HTTP response header value: %s").S)
				s = s.Rest()
			}
		default:
			panic(RT.NewError("HTTP response header value must be a string or a seq of strings"))
		}
	}
}

func responseStatus(response Map) int {
	status := 0
	if ok, s := response.Get(MakeKeyword("status")); ok {
		status = EnsureObjectIsInt(s, "HTTP response status: %s").I
	}
	return status
}

func appendSSEField(sb *strings.Builder, field string, value string) {
	for _, line := range strings.Split(value, "\n") {
		if field == "" {
			sb.WriteString(": ")
		} else {
			sb.WriteString(field)
			sb.WriteString(": ")
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
}

func formatSSEEvent(event Object) string {
	var sb strings.Builder
	wrote := false
	switch event := event.(type) {
	case String:
		appendSSEField(&sb, "data", event.S)
		wrote = true
	case Map:
		if ok, value := event.Get(MakeKeyword("comment")); ok {
			appendSSEField(&sb, "", EnsureObjectIsString(value, "SSE event comment: %s").S)
			wrote = true
		}
		if ok, value := event.Get(MakeKeyword("event")); ok {
			appendSSEField(&sb, "event", EnsureObjectIsString(value, "SSE event type: %s").S)
			wrote = true
		}
		if ok, value := event.Get(MakeKeyword("id")); ok {
			appendSSEField(&sb, "id", EnsureObjectIsString(value, "SSE event id: %s").S)
			wrote = true
		}
		if ok, value := event.Get(MakeKeyword("retry")); ok {
			appendSSEField(&sb, "retry", strconv.Itoa(EnsureObjectIsInt(value, "SSE event retry: %s").I))
			wrote = true
		}
		if ok, value := event.Get(MakeKeyword("data")); ok {
			appendSSEField(&sb, "data", EnsureObjectIsString(value, "SSE event data: %s").S)
			wrote = true
		}
		if !wrote {
			panic(RT.NewError("SSE event map must contain at least one of :data, :event, :id, :retry or :comment"))
		}
	default:
		panic(RT.NewError(fmt.Sprintf("SSE event must be a string or map, got %s", event.GetType().ToString(false))))
	}
	sb.WriteByte('\n')
	return sb.String()
}

func sseCloseInfo(reason string, err Error) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("reason"), MakeKeyword(reason))
	if err != nil {
		res.Add(MakeKeyword("error"), err)
	}
	return res
}

func streamSSE(response Map, w http.ResponseWriter, done <-chan struct{}) {
	ch := EnsureObjectIsChannel(getOrPanic(response, MakeKeyword("sse"), ":sse key must be present in SSE response map"), "SSE channel: %s")
	var onClose Callable
	if ok, value := response.Get(MakeKeyword("on-close")); ok {
		onClose = EnsureObjectIsCallable(value, "SSE on-close callback: %s")
	}
	var closeInfo Map
	defer func() {
		if r := recover(); r != nil {
			if closeInfo == nil {
				if err, ok := r.(Error); ok {
					closeInfo = sseCloseInfo("error", err)
				} else {
					closeInfo = sseCloseInfo("error", RT.NewError(fmt.Sprint(r)))
				}
			}
			if onClose != nil {
				onClose.Call([]Object{closeInfo})
			}
			panic(r)
		}
		if onClose != nil {
			onClose.Call([]Object{closeInfo})
		}
	}()
	flusher, ok := w.(http.Flusher)
	if !ok {
		panic(RT.NewError("HTTP response writer does not support streaming"))
	}
	header := w.Header()
	if ok, headers := response.Get(MakeKeyword("headers")); ok {
		addHeaders(headers, w)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "text/event-stream")
	}
	if header.Get("Cache-Control") == "" {
		header.Set("Cache-Control", "no-cache")
	}
	if header.Get("Connection") == "" {
		header.Set("Connection", "keep-alive")
	}
	if status := responseStatus(response); status != 0 {
		w.WriteHeader(status)
	}
	for {
		RT.GIL.Unlock()
		event, status, err := ch.Receive(done)
		RT.GIL.Lock()
		if err != nil {
			closeInfo = sseCloseInfo("error", err)
			panic(err)
		}
		if status == ChannelReceiveClosed {
			closeInfo = sseCloseInfo("channel-closed", nil)
			return
		}
		if status == ChannelReceiveDone {
			closeInfo = sseCloseInfo("client-closed", nil)
			return
		}
		msg := formatSSEEvent(event)
		RT.GIL.Unlock()
		_, writeErr := io.WriteString(w, msg)
		if writeErr == nil {
			flusher.Flush()
		}
		RT.GIL.Lock()
		if writeErr != nil {
			closeInfo = sseCloseInfo("write-error", RT.NewError(writeErr.Error()))
			return
		}
	}
}

func mapToResp(response Map, w http.ResponseWriter, done <-chan struct{}) {
	if ok, _ := response.Get(MakeKeyword("sse")); ok {
		streamSSE(response, w, done)
		return
	}
	status := responseStatus(response)
	body := ""
	if ok, b := response.Get(MakeKeyword("body")); ok {
		body = EnsureObjectIsString(b, "HTTP response body: %s").S
	}
	if ok, headers := response.Get(MakeKeyword("headers")); ok {
		addHeaders(headers, w)
	}
	if status != 0 {
		w.WriteHeader(status)
	}
	io.WriteString(w, body)
}

func sendRequest(request Map) Map {
	req := mapToReq(request)
	RT.GIL.Unlock()
	resp, err := client.Do(req)
	RT.GIL.Lock()
	PanicOnErr(err)
	return respToMap(resp)
}

func startServer(addr string, handler Callable) Object {
	i := strings.LastIndexByte(addr, byte(':'))
	host, port := MakeString(addr), MakeString("")
	if i != -1 {
		host = MakeString(addr[:i])
		port = MakeString(addr[i+1:])
	}
	RT.GIL.Unlock()
	defer RT.GIL.Lock()
	err := http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		RT.GIL.Lock()
		defer func() {
			RT.GIL.Unlock()
			if r := recover(); r != nil {
				w.WriteHeader(500)
				io.WriteString(w, "Internal server error")
				fmt.Fprintln(os.Stderr, r)
			}
		}()
		response := handler.Call([]Object{reqToMap(host, port, req)})
		mapToResp(EnsureObjectIsMap(response, "HTTP response: %s"), w, req.Context().Done())
	}))
	PanicOnErr(err)
	return NIL
}

func startFileServer(addr string, root string) Object {
	err := http.ListenAndServe(addr, http.FileServer(http.Dir(root)))
	PanicOnErr(err)
	return NIL
}
