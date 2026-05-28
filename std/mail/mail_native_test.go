package mail

import (
	"strings"
	"testing"

	. "github.com/candid82/joker/core"
)

func TestReadMessageFromReaderPreservesHeadersAndBody(t *testing.T) {
	source := MakeIOReader(strings.NewReader(
		"From: Alice <alice@example.com>\r\n" +
			"Received: one\r\n" +
			"Received: two\r\n" +
			"\r\n" +
			"line one\r\nline two\r\n",
	))

	message := readMessage(source)
	ok, headersObject := message.Get(MakeKeyword("headers"))
	if !ok {
		t.Fatal("message has no :headers value")
	}
	headers := EnsureObjectIsMap(headersObject, "headers: %s")
	ok, received := headers.Get(MakeString("Received"))
	if !ok || !received.Equals(MakeStringVector([]string{"one", "two"})) {
		t.Fatalf("unexpected Received headers: %v", received)
	}

	ok, body := message.Get(MakeKeyword("body"))
	if !ok || !body.Equals(MakeString("line one\r\nline two\r\n")) {
		t.Fatalf("unexpected body: %v", body)
	}
}
