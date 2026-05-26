package pop3

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	. "github.com/candid82/joker/core"
)

func writePOP3Line(rw *bufio.ReadWriter, line string) {
	_, _ = rw.WriteString(line + "\r\n")
	_ = rw.Flush()
}

func servePOP3(conn net.Conn, startTLS *tls.Config, commands chan<- []string) {
	defer conn.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	writePOP3Line(rw, "+OK ready")
	var seen []string
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			commands <- seen
			return
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		seen = append(seen, line)
		switch line {
		case "STLS":
			writePOP3Line(rw, "+OK begin TLS")
			tlsConn := tls.Server(conn, startTLS)
			if tlsConn.Handshake() != nil {
				commands <- seen
				return
			}
			conn = tlsConn
			rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		case "USER alice", "PASS secret", "DELE 1", "RSET", "NOOP":
			writePOP3Line(rw, "+OK")
		case "CAPA":
			_, _ = rw.WriteString("+OK capabilities\r\nUIDL\r\nTOP\r\nSASL PLAIN\r\n.\r\n")
			_ = rw.Flush()
		case "STAT":
			writePOP3Line(rw, "+OK 2 320")
		case "LIST":
			_, _ = rw.WriteString("+OK list\r\n1 120\r\n2 200\r\n.\r\n")
			_ = rw.Flush()
		case "LIST 1":
			writePOP3Line(rw, "+OK 1 120")
		case "UIDL":
			_, _ = rw.WriteString("+OK uidl\r\n1 first\r\n2 second\r\n.\r\n")
			_ = rw.Flush()
		case "UIDL 1":
			writePOP3Line(rw, "+OK 1 first")
		case "RETR 1":
			_, _ = rw.WriteString("+OK message\r\nSubject: test\r\n\r\n..dot-start\r\n.\r\n")
			_ = rw.Flush()
		case "TOP 1 0":
			_, _ = rw.WriteString("+OK top\r\nSubject: test\r\n\r\n.\r\n")
			_ = rw.Flush()
		case "QUIT":
			writePOP3Line(rw, "+OK bye")
			commands <- seen
			return
		default:
			writePOP3Line(rw, "-ERR unexpected")
		}
	}
}

func startPlainServer(t *testing.T, startTLS *tls.Config) (string, <-chan []string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	commands := make(chan []string, 1)
	go func() {
		defer listener.Close()
		conn, err := listener.Accept()
		if err == nil {
			servePOP3(conn, startTLS, commands)
		}
	}()
	return listener.Addr().String(), commands
}

func testCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}),
	)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots.AddCert(parsed)
	return cert, roots
}

func useTestRoots(t *testing.T, roots *x509.CertPool) {
	t.Helper()
	previous := makeTLSConfig
	makeTLSConfig = func(serverName string) *tls.Config {
		return &tls.Config{RootCAs: roots, ServerName: serverName}
	}
	t.Cleanup(func() {
		makeTLSConfig = previous
	})
}

func connectOptionsMap(addr string, mode string) Map {
	opts := EmptyArrayMap()
	opts.Add(MakeKeyword("addr"), MakeString(addr))
	opts.Add(MakeKeyword("username"), MakeString("alice"))
	opts.Add(MakeKeyword("password"), MakeString("secret"))
	opts.Add(MakeKeyword("tls"), MakeKeyword(mode))
	if mode != "none" {
		opts.Add(MakeKeyword("server-name"), MakeString("localhost"))
	}
	return opts
}

func connectForTest(t *testing.T, addr string, mode string) *pop3Client {
	t.Helper()
	RT.GIL.Lock()
	client := connect(connectOptionsMap(addr, mode))
	RT.GIL.Unlock()
	return client
}

func expectPanicWithGIL(t *testing.T, expected string, fn func()) {
	t.Helper()
	var recovered interface{}
	func() {
		RT.GIL.Lock()
		defer func() {
			recovered = recover()
			RT.GIL.Unlock()
		}()
		fn()
	}()
	if recovered == nil || !strings.Contains(fmt.Sprint(recovered), expected) {
		t.Fatalf("expected panic containing %q, got %v", expected, recovered)
	}
}

func TestClientCommandsAndRawMessages(t *testing.T) {
	addr, commands := startPlainServer(t, nil)
	client := connectForTest(t, addr, "none")

	RT.GIL.Lock()
	caps := capabilities(client)
	stats := stat(client)
	_ = listAll(client)
	_ = listOne(client, 1)
	_ = uidlAll(client)
	_ = uidlOne(client, 1)
	message := retrieve(client, 1)
	headers := top(client, 1, 0)
	_ = deleteMessage(client, 1)
	_ = reset(client)
	_ = noop(client)
	_ = quit(client)
	RT.GIL.Unlock()

	if ok, _ := caps.Get(MakeString("UIDL")); !ok {
		t.Fatalf("expected UIDL capability, got %s", caps.ToString(false))
	}
	if ok, _ := stats.Get(MakeKeyword("count")); !ok {
		t.Fatalf("expected count in STAT result, got %s", stats.ToString(false))
	}
	if message != "Subject: test\r\n\r\n.dot-start\r\n" {
		t.Fatalf("unexpected RETR value: %q", message)
	}
	if headers != "Subject: test\r\n\r\n" {
		t.Fatalf("unexpected TOP value: %q", headers)
	}
	seen := <-commands
	if strings.Join(seen, ",") != "USER alice,PASS secret,CAPA,STAT,LIST,LIST 1,UIDL,UIDL 1,RETR 1,TOP 1 0,DELE 1,RSET,NOOP,QUIT" {
		t.Fatalf("unexpected commands: %v", seen)
	}
}

func TestCloseDoesNotCommitDeletions(t *testing.T) {
	addr, commands := startPlainServer(t, nil)
	client := connectForTest(t, addr, "none")

	RT.GIL.Lock()
	_ = deleteMessage(client, 1)
	_ = close(client)
	_ = close(client)
	RT.GIL.Unlock()

	seen := <-commands
	if strings.Join(seen, ",") != "USER alice,PASS secret,DELE 1" {
		t.Fatalf("unexpected commands before close: %v", seen)
	}
	expectPanicWithGIL(t, "client is closed", func() {
		noop(client)
	})
}

func TestServerRejectionKeepsSessionUsable(t *testing.T) {
	addr, commands := startPlainServer(t, nil)
	client := connectForTest(t, addr, "none")

	expectPanicWithGIL(t, "-ERR unexpected", func() {
		top(client, 2, 0)
	})
	RT.GIL.Lock()
	_ = quit(client)
	RT.GIL.Unlock()

	seen := <-commands
	if strings.Join(seen, ",") != "USER alice,PASS secret,TOP 2 0,QUIT" {
		t.Fatalf("unexpected commands after server rejection: %v", seen)
	}
}

func TestInvalidOptionsRejectCommandInjection(t *testing.T) {
	opts := EmptyArrayMap()
	opts.Add(MakeKeyword("addr"), MakeString("unused:110"))
	opts.Add(MakeKeyword("username"), MakeString("alice\r\nDELE 1"))
	opts.Add(MakeKeyword("password"), MakeString("secret"))
	opts.Add(MakeKeyword("tls"), MakeKeyword("none"))
	expectPanicWithGIL(t, "username must not contain CR or LF", func() {
		connect(opts)
	})
}

func TestImplicitTLSAndSTARTTLS(t *testing.T) {
	cert, roots := testCertificate(t)
	useTestRoots(t, roots)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}}

	t.Run("implicit", func(t *testing.T) {
		listener, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
		if err != nil {
			t.Fatal(err)
		}
		commands := make(chan []string, 1)
		go func() {
			defer listener.Close()
			conn, err := listener.Accept()
			if err == nil {
				servePOP3(conn, nil, commands)
			}
		}()
		client := connectForTest(t, listener.Addr().String(), "implicit")
		RT.GIL.Lock()
		_ = quit(client)
		RT.GIL.Unlock()
		if strings.Join(<-commands, ",") != "USER alice,PASS secret,QUIT" {
			t.Fatal("implicit TLS command sequence is incorrect")
		}
	})

	t.Run("starttls", func(t *testing.T) {
		addr, commands := startPlainServer(t, serverTLS)
		client := connectForTest(t, addr, "starttls")
		RT.GIL.Lock()
		_ = quit(client)
		RT.GIL.Unlock()
		if strings.Join(<-commands, ",") != "STLS,USER alice,PASS secret,QUIT" {
			t.Fatal("STARTTLS command sequence is incorrect")
		}
	})

	t.Run("certificate-name-mismatch", func(t *testing.T) {
		listener, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
		if err != nil {
			t.Fatal(err)
		}
		commands := make(chan []string, 1)
		go func() {
			defer listener.Close()
			conn, err := listener.Accept()
			if err == nil {
				servePOP3(conn, nil, commands)
			}
		}()
		opts := connectOptionsMap(listener.Addr().String(), "implicit")
		opts = opts.Assoc(MakeKeyword("server-name"), MakeString("wrong.example")).(Map)
		expectPanicWithGIL(t, "certificate", func() {
			connect(opts)
		})
	})
}

func TestConnectTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			time.Sleep(100 * time.Millisecond)
		}
	}()
	_, err = connectNative(connectOptions{
		addr:     listener.Addr().String(),
		username: "alice",
		password: "secret",
		tlsMode:  "none",
		timeout:  10 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "greeting") {
		t.Fatalf("expected greeting timeout, got %v", err)
	}
}
