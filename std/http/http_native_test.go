package http

import (
	"net/http/httptest"
	"testing"

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
