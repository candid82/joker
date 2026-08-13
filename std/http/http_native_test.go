package http

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/candid82/joker/core"
)

func mapValue(t *testing.T, m Map, key string) Object {
	t.Helper()
	ok, value := m.Get(MakeKeyword(key))
	if !ok {
		t.Fatalf("expected map to contain :%s", key)
	}
	return value
}

func requestMap(url string) Map {
	request := EmptyArrayMap()
	request.Add(MakeKeyword("url"), MakeString(url))
	return request
}

func optionsMap(values map[string]Object) Map {
	opts := EmptyArrayMap()
	for key, value := range values {
		opts.Add(MakeKeyword(key), value)
	}
	return opts
}

func sendForTest(request Map, opts Map) (result Map, recovered interface{}) {
	RT.GIL.Lock()
	defer func() {
		recovered = recover()
		RT.GIL.Unlock()
	}()
	result = sendRequest(request, opts)
	return result, nil
}

func expectPanic(t *testing.T, expected string, fn func()) {
	t.Helper()
	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		fn()
	}()
	if recovered == nil || !strings.Contains(fmt.Sprint(recovered), expected) {
		t.Fatalf("expected panic containing %q, got %v", expected, recovered)
	}
}

func TestParseRequestOptionsAndClientConfiguration(t *testing.T) {
	defaultTimeout := client.Timeout
	opts := parseRequestOptions(optionsMap(map[string]Object{
		"timeout-ms":                 MakeInt(1250),
		"connect-timeout-ms":         MakeInt(250),
		"response-header-timeout-ms": MakeInt(500),
		"max-response-bytes":         MakeInt(4096),
	}))
	if opts.timeout != 1250*time.Millisecond || opts.connectTimeout != 250*time.Millisecond ||
		opts.responseHeaderTimeout != 500*time.Millisecond {
		t.Fatalf("unexpected parsed timeouts: %+v", opts)
	}
	if !opts.hasMaxResponseBytes || opts.maxResponseBytes != 4096 {
		t.Fatalf("unexpected response limit: %+v", opts)
	}

	requestClient, transport := clientForRequest(opts)
	if requestClient == client || transport == nil {
		t.Fatal("expected a per-request client and transport")
	}
	defer transport.CloseIdleConnections()
	if requestClient.Timeout != 1250*time.Millisecond {
		t.Fatalf("expected 1250ms timeout, got %s", requestClient.Timeout)
	}
	if transport.ResponseHeaderTimeout != 500*time.Millisecond {
		t.Fatalf("expected 500ms response header timeout, got %s", transport.ResponseHeaderTimeout)
	}
	if client.Timeout != defaultTimeout || client.Transport != nil {
		t.Fatal("expected default client to remain unchanged")
	}

	defaultClient, defaultTransport := clientForRequest(parseRequestOptions(nil))
	if defaultClient != client || defaultTransport != nil {
		t.Fatal("expected omitted options to use the default client")
	}
}

func TestParseRequestOptionsRejectsInvalidValues(t *testing.T) {
	for _, name := range []string{
		"timeout-ms", "connect-timeout-ms", "response-header-timeout-ms",
		"max-redirects", "max-response-bytes",
	} {
		name := name
		for _, value := range []int{0, -1} {
			value := value
			expectPanic(t, ":"+name+" must be positive", func() {
				parseRequestOptions(optionsMap(map[string]Object{name: MakeInt(value)}))
			})
		}
	}
	expectPanic(t, ":max-redirects cannot be used", func() {
		parseRequestOptions(optionsMap(map[string]Object{
			"follow-redirects?": MakeBoolean(false),
			"max-redirects":     MakeInt(2),
		}))
	})
	expectPanic(t, "follow-redirects?", func() {
		parseRequestOptions(optionsMap(map[string]Object{"follow-redirects?": MakeInt(1)}))
	})
}

func TestSendRequestTimeoutIncludesResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("late body"))
	}))
	defer server.Close()

	_, recovered := sendForTest(requestMap(server.URL), optionsMap(map[string]Object{
		"timeout-ms": MakeInt(20),
	}))
	if recovered == nil || !strings.Contains(strings.ToLower(fmt.Sprint(recovered)), "timeout") {
		t.Fatalf("expected response body timeout, got %v", recovered)
	}
}

func TestSendRequestResponseHeaderTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("late response"))
	}))
	defer server.Close()

	_, recovered := sendForTest(requestMap(server.URL), optionsMap(map[string]Object{
		"response-header-timeout-ms": MakeInt(20),
	}))
	if recovered == nil || !strings.Contains(strings.ToLower(fmt.Sprint(recovered)), "timeout") {
		t.Fatalf("expected response header timeout, got %v", recovered)
	}
}

func TestSendRequestRedirectOptions(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, server.URL+"/middle", http.StatusFound)
		case "/middle":
			http.Redirect(w, r, server.URL+"/end", http.StatusFound)
		default:
			_, _ = w.Write([]byte("done"))
		}
	}))
	defer server.Close()

	response, recovered := sendForTest(requestMap(server.URL+"/start"), optionsMap(map[string]Object{
		"follow-redirects?": MakeBoolean(false),
	}))
	if recovered != nil {
		t.Fatalf("unexpected redirect-disabled error: %v", recovered)
	}
	if got := mapValue(t, response, "status"); !got.Equals(MakeInt(http.StatusFound)) {
		t.Fatalf("expected first redirect response, got status %s", got.ToString(false))
	}

	response, recovered = sendForTest(requestMap(server.URL+"/start"), optionsMap(map[string]Object{
		"max-redirects": MakeInt(2),
	}))
	if recovered != nil || !mapValue(t, response, "body").Equals(MakeString("done")) {
		t.Fatalf("expected two redirects to succeed, got response %v and error %v", response, recovered)
	}

	_, recovered = sendForTest(requestMap(server.URL+"/start"), optionsMap(map[string]Object{
		"max-redirects": MakeInt(1),
	}))
	if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "stopped after 1 redirects") {
		t.Fatalf("expected redirect limit error, got %v", recovered)
	}
}

func TestSendRequestMaxResponseBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("12345"))
	}))
	defer server.Close()

	response, recovered := sendForTest(requestMap(server.URL), optionsMap(map[string]Object{
		"max-response-bytes": MakeInt(5),
	}))
	if recovered != nil || !mapValue(t, response, "body").Equals(MakeString("12345")) {
		t.Fatalf("expected body at limit to succeed, got response %v and error %v", response, recovered)
	}

	_, recovered = sendForTest(requestMap(server.URL), optionsMap(map[string]Object{
		"max-response-bytes": MakeInt(4),
	}))
	if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "exceeds :max-response-bytes") {
		t.Fatalf("expected response size limit error, got %v", recovered)
	}
}

func TestSendRequestMaxResponseBytesUsesDecodedSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(w)
		_, _ = writer.Write([]byte("12345"))
		_ = writer.Close()
	}))
	defer server.Close()

	response, recovered := sendForTest(requestMap(server.URL), optionsMap(map[string]Object{
		"max-response-bytes": MakeInt(5),
	}))
	if recovered != nil || !mapValue(t, response, "body").Equals(MakeString("12345")) {
		t.Fatalf("expected decoded body at limit to succeed, got response %v and error %v", response, recovered)
	}

	_, recovered = sendForTest(requestMap(server.URL), optionsMap(map[string]Object{
		"max-response-bytes": MakeInt(4),
	}))
	if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "exceeds :max-response-bytes") {
		t.Fatalf("expected decoded response size limit error, got %v", recovered)
	}
}

func TestStreamSSEFormatsEventsAndReportsChannelClose(t *testing.T) {
	events := MakeChannel(make(chan FutureResult, 3))
	events.Send(MakeString("hello"))

	note := EmptyArrayMap()
	note.Add(MakeKeyword("event"), MakeString("note"))
	note.Add(MakeKeyword("id"), MakeString("42"))
	note.Add(MakeKeyword("retry"), MakeInt(1500))
	note.Add(MakeKeyword("data"), MakeString("line 1\nline 2"))
	events.Send(note)

	comment := EmptyArrayMap()
	comment.Add(MakeKeyword("comment"), MakeString("done"))
	events.Send(comment)
	events.Close()

	var closeInfo Map
	response := EmptyArrayMap()
	response.Add(MakeKeyword("status"), MakeInt(202))
	response.Add(MakeKeyword("sse"), events)
	response.Add(MakeKeyword("on-close"), Proc{
		Fn: func(args []Object) Object {
			closeInfo = EnsureObjectIsMap(args[0], "close info: %s")
			return NIL
		},
		Name:    "on-close",
		Package: "std/http",
	})

	recorder := httptest.NewRecorder()
	RT.GIL.Lock()
	defer RT.GIL.Unlock()
	streamSSE(response, recorder, nil)

	if recorder.Code != 202 {
		t.Fatalf("expected status 202, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected text/event-stream content type, got %q", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("expected no-cache cache control, got %q", got)
	}
	if got := recorder.Header().Get("Connection"); got != "keep-alive" {
		t.Fatalf("expected keep-alive connection header, got %q", got)
	}

	const expectedBody = "data: hello\n\n" +
		"event: note\n" +
		"id: 42\n" +
		"retry: 1500\n" +
		"data: line 1\n" +
		"data: line 2\n\n" +
		": done\n\n"
	if got := recorder.Body.String(); got != expectedBody {
		t.Fatalf("unexpected SSE body:\n%s", got)
	}

	if got := mapValue(t, closeInfo, "reason"); !got.Equals(MakeKeyword("channel-closed")) {
		t.Fatalf("expected :channel-closed close reason, got %s", got.ToString(false))
	}
}

func TestStreamSSEReportsClientClose(t *testing.T) {
	events := MakeChannel(make(chan FutureResult))
	done := make(chan struct{})
	close(done)

	var closeInfo Map
	response := EmptyArrayMap()
	response.Add(MakeKeyword("sse"), events)
	response.Add(MakeKeyword("on-close"), Proc{
		Fn: func(args []Object) Object {
			closeInfo = EnsureObjectIsMap(args[0], "close info: %s")
			return NIL
		},
		Name:    "on-close",
		Package: "std/http",
	})

	recorder := httptest.NewRecorder()
	RT.GIL.Lock()
	defer RT.GIL.Unlock()
	streamSSE(response, recorder, done)

	if got := mapValue(t, closeInfo, "reason"); !got.Equals(MakeKeyword("client-closed")) {
		t.Fatalf("expected :client-closed close reason, got %s", got.ToString(false))
	}
}

func TestStreamSSEReportsFormattingErrorsToOnClose(t *testing.T) {
	events := MakeChannel(make(chan FutureResult, 1))
	events.Send(MakeInt(42))

	var closeInfo Map
	response := EmptyArrayMap()
	response.Add(MakeKeyword("sse"), events)
	response.Add(MakeKeyword("on-close"), Proc{
		Fn: func(args []Object) Object {
			closeInfo = EnsureObjectIsMap(args[0], "close info: %s")
			return NIL
		},
		Name:    "on-close",
		Package: "std/http",
	})

	recorder := httptest.NewRecorder()
	RT.GIL.Lock()
	defer RT.GIL.Unlock()
	defer func() {
		if recover() == nil {
			t.Fatal("expected invalid SSE event to panic")
		}
		if got := mapValue(t, closeInfo, "reason"); !got.Equals(MakeKeyword("error")) {
			t.Fatalf("expected :error close reason, got %s", got.ToString(false))
		}
		if ok, _ := closeInfo.Get(MakeKeyword("error")); !ok {
			t.Fatal("expected close info to contain :error")
		}
	}()

	streamSSE(response, recorder, nil)
}
