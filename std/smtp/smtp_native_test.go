package smtp

import (
	"bufio"
	"net"
	"strings"
	"testing"

	. "github.com/candid82/joker/core"
)

func startSMTPServer(t *testing.T) (string, <-chan []string, <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	commands := make(chan []string, 1)
	message := make(chan string, 1)
	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		writeLine := func(line string) {
			_, _ = rw.WriteString(line + "\r\n")
			_ = rw.Flush()
		}
		writeLine("220 localhost ESMTP")
		var seen []string
		for {
			line, err := rw.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			seen = append(seen, line)
			switch {
			case strings.HasPrefix(line, "EHLO ") || strings.HasPrefix(line, "HELO "):
				_, _ = rw.WriteString("250-localhost\r\n250 OK\r\n")
				_ = rw.Flush()
			case strings.HasPrefix(line, "MAIL FROM:"):
				writeLine("250 OK")
			case strings.HasPrefix(line, "RCPT TO:"):
				writeLine("250 OK")
			case line == "DATA":
				writeLine("354 End data")
				var sb strings.Builder
				for {
					msgLine, err := rw.ReadString('\n')
					if err != nil {
						return
					}
					if msgLine == ".\r\n" {
						break
					}
					sb.WriteString(msgLine)
				}
				message <- sb.String()
				writeLine("250 OK")
			case line == "QUIT":
				writeLine("221 Bye")
				commands <- seen
				return
			default:
				writeLine("250 OK")
			}
		}
	}()
	return ln.Addr().String(), commands, message
}

func TestSendMessage(t *testing.T) {
	addr, commands, message := startSMTPServer(t)
	request := EmptyArrayMap()
	request.Add(MakeKeyword("addr"), MakeString(addr))
	request.Add(MakeKeyword("from"), MakeString("sender@example.com"))
	request.Add(MakeKeyword("to"), NewVectorFrom(MakeString("one@example.com"), MakeString("two@example.com")))
	request.Add(MakeKeyword("message"), MakeString("From: sender@example.com\r\nTo: one@example.com\r\nSubject: Hello\r\n\r\nBody\r\n"))

	RT.GIL.Lock()
	got := sendMessage(request)
	RT.GIL.Unlock()

	if got != NIL {
		t.Fatalf("expected nil, got %s", got.ToString(false))
	}
	if got := <-message; got != "From: sender@example.com\r\nTo: one@example.com\r\nSubject: Hello\r\n\r\nBody\r\n" {
		t.Fatalf("unexpected message:\n%s", got)
	}
	seen := <-commands
	if len(seen) < 5 {
		t.Fatalf("expected SMTP commands, got %v", seen)
	}
	if !strings.HasPrefix(seen[1], "MAIL FROM:<sender@example.com>") {
		t.Fatalf("unexpected MAIL command: %v", seen)
	}
	if !strings.HasPrefix(seen[2], "RCPT TO:<one@example.com>") {
		t.Fatalf("unexpected first RCPT command: %v", seen)
	}
	if !strings.HasPrefix(seen[3], "RCPT TO:<two@example.com>") {
		t.Fatalf("unexpected second RCPT command: %v", seen)
	}
}
